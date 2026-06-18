# 雪局 MVP

雪局是一个滑雪行程组局工具 MVP。本仓库按阶段开发，当前完成阶段 0：项目初始化，并已加入微信小程序审核合规基础设计。

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

## 管理后台启动

```bash
cd admin
npm install
npm run dev
```

默认地址为 `http://127.0.0.1:5173`。

## 小程序启动

使用微信开发者工具导入 `miniprogram` 目录。阶段 0 首页会请求 `http://127.0.0.1:8080/api/health` 检查后端连通性。

## 阶段 0 验收

- 后端可启动
- 小程序可导入并启动
- 后台可启动
- `/api/health` 返回统一成功结构
