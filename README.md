# 雪局 MVP

雪局是一个滑雪行程组局工具。仓库包含可联调的小程序、Go API 和运营管理后台，覆盖发布、申请审核、群聊、滑后评价、举报和内容治理闭环。

## 目录结构

- `backend`：Go + Gin API 服务
- `miniprogram`：微信小程序原生项目
- `admin`：Vue3 + Element Plus 管理后台
- `ski_partner_codex_docs`：产品与开发交接文档

## 后端启动

```bash
cd backend
go mod tidy
go run ./cmd/api
```

默认端口为 `8080`，健康检查：

```bash
curl http://127.0.0.1:8080/api/health
```

返回格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "app": "xueju-api",
    "env": "development",
    "database": "not_configured",
    "timestamp": "2026-06-18T16:00:00+08:00"
  }
}
```

后端要求 Go 1.25。若需连接 MySQL，可复制 `backend/.env.example` 并按本地环境设置 `MYSQL_DSN` 后再启动。

## 审核合规设计

- 用户默认处于“未完成手机号认证”状态；发布/修改行程、申请加入、审批成员、查看群聊和群聊发言均由后端强制校验手机号认证状态。
- 个人主体小程序使用阿里云号码认证服务的 `SendSmsVerifyCode` / `CheckSmsVerifyCode` 完成短信认证，不使用微信 `getPhoneNumber`，也不设置线下身份证核验或管理员确认实名。
- 完整手机号使用 AES-GCM 加密保存，精确去重使用服务端密钥 HMAC-SHA256，前台与普通后台仅展示脱敏号码；发送、核验、变更、重认证和撤销均写入独立审计日志。
- 每个滑雪局（局内群聊）总人数最多 20 人，创建、修改和成员审批三个入口均校验上限。
- 文本先经过本地敏感词拦截，再同步调用阿里云内容安全增强版；群聊消息只有检测通过后才入库，同时保留后台复核、隐藏和审计能力。
- 产品定位统一为“滑雪行程组局工具”。
- 第一版只保留滑雪行程组织能力，不提供交易、担保、票务或住宿撮合能力。
- 同行交通只作为行程说明，不提供营运车辆撮合、派单、资金结算或抽成。
- 住宿仅保留需求备注，不做撮合闭环。
- 小程序不展示个人联系方式或联系码。
- 个人信息相关页面通过 `wx.requirePrivacyAuthorize` 配合微信小程序隐私保护指引。
- 后端使用阿里云文本和图片审核增强版；微信 `msgSecCheck`、`mediaCheckAsync` 仅作为兼容的旧提供方保留。
- 文本检测范围：滑雪局标题、滑雪局备注、用户昵称、用户简介、群聊消息、评价内容、举报内容。
- 文本命中中高风险时返回 `40010`；云端不可用时拒绝写入。图片高风险直接拒绝，中风险或云端不可用时进入待人工审核，低风险才公开。
- 后台预留用户管理、滑雪行程下架、消息隐藏、评价隐藏、内容审核、举报处理、用户禁用入口。

### 手机号认证上线配置

1. 使用个人实名认证的阿里云账号开通“号码认证服务 → 短信认证”，选择控制台提供的系统签名与标准验证码模板。
2. 创建仅允许 `dypns:SendSmsVerifyCode`、`dypns:CheckSmsVerifyCode` 的 RAM 用户 AccessKey，不使用主账号 AccessKey。
3. 在服务端环境配置 `ALIYUN_ACCESS_KEY_ID`、`ALIYUN_ACCESS_KEY_SECRET`、`ALIYUN_SMS_VERIFY_SIGN_NAME`、`ALIYUN_SMS_VERIFY_TEMPLATE_CODE`，按需配置 `ALIYUN_SMS_VERIFY_SCHEME_NAME`。
4. 配置彼此独立、至少 32 字符的 `PHONE_ENCRYPTION_KEY` 与 `PHONE_HASH_SECRET`；密钥不得提交到仓库，变更密钥前需制定历史手机号密文迁移方案。
5. 在微信小程序管理后台的《隐私保护指引》中同步声明手机号、阿里云号码认证服务、处理目的、保存期限和用户权利。

默认频控为同一手机号 60 秒一次、手机号/用户每天 10 次、IP 每小时 30 次；可通过 `SMS_*` 环境变量调整。

## 管理后台启动

```bash
cd admin
npm install
npm run dev
```

默认地址为 `http://127.0.0.1:5173`。

## 小程序启动

使用微信开发者工具导入 `miniprogram` 目录。开发版默认请求 `http://127.0.0.1:8080/api`；体验版和正式版必须通过小程序 `extConfig.apiBaseUrl` 或构建时替换 `miniprogram/config/runtime.js` 注入 HTTPS API 地址，代码不会回退到开发 IP。

## 测试

```bash
cd backend
go test ./...

cd ../admin
npm ci
npm run build
```

设置 `XUEJU_TEST_MYSQL_DSN` 后，`go test ./integration -v` 会执行真实 MySQL 状态机测试。CI 会自动运行后端测试、管理端正式构建、小程序 JavaScript 语法检查和 MySQL 集成测试。

需要演示数据时，可仅在开发库执行 `backend/scripts/seed-dev.sql`，它会创建 5 个用户和 10 条未来行程；生产迁移只初始化基础雪场字典。

## 生产部署（不依赖 Docker）

1. 准备 MySQL 8 数据库，将 `backend/.env.example` 中的变量写入主机环境或密钥管理系统。不要提交真实密钥；`APP_TIMEZONE` 保持为 `Asia/Shanghai`。
2. 编译并启动 API：

   ```bash
   cd backend
   CGO_ENABLED=0 go build -trimpath -o xueju-api ./cmd/api
   APP_ENV=production ./xueju-api
   ```

   生产环境数据库连接失败时进程会直接退出；`/api/health` 只有在数据库可用时才返回 HTTP 200。
3. 构建管理端：

   ```bash
   cd admin
   npm ci
   npm run build
   ```

   将 `admin/dist` 发布到静态 Web 服务的 `/admin/` 目录，并把同域 `/api` 与 `/uploads` 转发到 API 服务。管理端入口为 `/admin/`，SPA 路由刷新必须回退到 `/admin/index.html`。若由外层 Nginx 代理 `admin` 容器，应保留路径前缀，例如：

   ```nginx
   location = /admin { return 301 /admin/; }
   location /admin/ {
     proxy_pass http://127.0.0.1:8081;
   }
   ```

   `proxy_pass` 末尾不要添加 `/`，否则会剥离 `/admin/` 前缀。若管理端和 API 使用不同域名，需要额外配置受限 CORS；推荐保持同域。
4. 小程序正式版把 HTTPS 域名配置为 request/download 合法域名，并通过 `extConfig.apiBaseUrl` 指向其 `/api` 路径。
5. 生产环境仍会拒绝默认 JWT、默认后台账号、非 HTTPS 公网地址、未开启内容安全或未配置 S3 兼容对象存储的配置。
6. 使用 `scripts/backup-mysql.ps1` 创建数据库备份，并由系统计划任务上传到异地存储；上线前必须在隔离数据库验证一次恢复流程。

开发环境也可以使用 `scripts/start-backend.sh`、`scripts/start-admin.sh`；Windows PowerShell 使用同名 `.ps1` 脚本。Docker 文件仅作为可选部署参考，不是运行本工程的前置条件。

所有请求响应携带 `X-Request-ID`，服务端输出结构化请求日志。

媒体上传会先同步调用阿里云图片审核：低风险直接标记为 `approved`，中风险或服务异常标记为 `pending`，高风险不保存。运营人员可在“媒体审核”中人工批准或拒绝待审图片；只有 `approved` 文件可以通过公开地址访问。
