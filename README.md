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

如需连接 MySQL，可复制 `backend/.env.example` 并按本地环境设置 `MYSQL_DSN` 后再启动。

## 审核合规设计

- 用户默认处于“未完成手机号认证”状态；发布/修改行程、申请加入、审批成员、查看群聊和群聊发言均由后端强制校验手机号认证状态。
- 个人主体小程序使用阿里云号码认证服务的 `SendSmsVerifyCode` / `CheckSmsVerifyCode` 完成短信认证，不使用微信 `getPhoneNumber`，也不设置线下身份证核验或管理员确认实名。
- 完整手机号使用 AES-GCM 加密保存，精确去重使用服务端密钥 HMAC-SHA256，前台与普通后台仅展示脱敏号码；发送、核验、变更、重认证和撤销均写入独立审计日志。
- 每个滑雪局（局内群聊）总人数最多 20 人，创建、修改和成员审批三个入口均校验上限。
- 文本先经过本地敏感词拦截，再调用微信 `msgSecCheck`；群聊消息只有检测通过后才入库，同时保留后台复核、隐藏和审计能力。
- 产品定位统一为“滑雪行程组局工具”。
- 第一版只保留滑雪行程组织能力，不提供交易、担保、票务或住宿撮合能力。
- 同行交通只作为行程说明，不提供营运车辆撮合、派单、资金结算或抽成。
- 住宿仅保留需求备注，不做撮合闭环。
- 小程序不展示个人联系方式或联系码。
- 个人信息相关页面通过 `wx.requirePrivacyAuthorize` 配合微信小程序隐私保护指引。
- 后端预留微信 `msgSecCheck`、`mediaCheckAsync` 内容安全服务。
- 文本检测范围：滑雪局标题、滑雪局备注、用户昵称、用户简介、群聊消息、评价内容、举报内容。
- 内容检测失败统一返回 `40010`，前端提示“内容可能包含不适宜信息，请修改后重试”。
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

## 生产部署

- 复制 `backend/.env.example` 的变量到密钥管理系统，不要提交真实密钥。
- 生产环境会拒绝默认 JWT、默认后台账号、非 HTTPS 公网地址、未开启微信内容安全或未配置 S3 兼容对象存储的配置。
- `docker compose up --build -d` 可启动 MySQL、API 和管理后台；公网 TLS 应在负载均衡或反向代理层终止。
- 小程序正式版把同一 HTTPS 域名配置为 request/download 合法域名，并通过 `extConfig.apiBaseUrl` 指向其 `/api` 路径。
- 使用 `scripts/backup-mysql.ps1` 创建数据库备份，并由系统计划任务上传到异地存储；上线前必须实际验证一次恢复流程。
- 使用 `scripts/restore-mysql.ps1 -BackupFile <path>` 在隔离数据库执行恢复演练，脚本要求输入 `RESTORE` 二次确认。
- `/api/health` 用于存活和数据库就绪检查，所有请求响应携带 `X-Request-ID`，服务端输出结构化请求日志。

媒体上传默认处于 `pending`。将微信媒体审核回调配置为 `POST /api/callbacks/wechat/media?token=<WECHAT_MEDIA_CALLBACK_TOKEN>`，审核通过后系统自动公开；运营人员也可在“媒体审核”中人工批准或拒绝。只有 `approved` 文件可以通过公开地址访问。
