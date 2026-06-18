# API 接口约定

## 1. 通用规范

Base URL：

```text
/api
```

统一返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

错误示例：

```json
{
  "code": 40001,
  "message": "未登录",
  "data": null
}
```

分页：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [],
    "page": 1,
    "pageSize": 10,
    "total": 0
  }
}
```

## 2. Auth 用户认证

### 2.1 微信登录

POST `/auth/wechat-login`

请求：

```json
{
  "code": "wx_login_code"
}
```

返回：

```json
{
  "token": "jwt_token",
  "user": {}
}
```

### 2.2 获取当前用户

GET `/user/me`

Header：

```text
Authorization: Bearer <token>
```

### 2.3 更新个人资料

PUT `/user/me`

请求：

```json
{
  "nickname": "大力",
  "avatarUrl": "",
  "gender": 1,
  "genderVisible": true,
  "city": "北京",
  "skiType": "snowboard",
  "skiLevel": "intermediate",
  "styleTags": ["刷道", "互拍"],
  "favoriteResorts": ["万龙", "南山"],
  "hasCar": true
}
```

## 3. 滑雪局

### 3.1 滑雪局列表

GET `/events`

Query：
- page
- pageSize
- city
- resortId
- date
- skiType
- level
- trafficType
- allowCarPool
- allowRoomShare
- sameGenderOnly
- keyword
- sort=recommend/latest

### 3.2 滑雪局详情

GET `/events/:id`

### 3.3 创建滑雪局

POST `/events`

请求：

```json
{
  "title": "周六万龙单板中级局",
  "resortId": 1,
  "eventDate": "2026-12-19",
  "startTime": "2026-12-19 06:30:00",
  "departCity": "北京",
  "departArea": "朝阳",
  "meetPlace": "朝阳大悦城停车场",
  "trafficType": "self_drive",
  "maxMembers": 4,
  "skiTypeReq": "snowboard",
  "levelReq": "intermediate",
  "purposeTags": ["刷道", "互拍"],
  "allowBeginner": false,
  "sameGenderOnly": false,
  "allowCarPool": true,
  "allowRoomShare": false,
  "allowPhoto": true,
  "costDesc": "油费高速费 AA",
  "remark": "希望节奏差不多，不带纯新手"
}
```

### 3.4 更新滑雪局

PUT `/events/:id`

仅发起人可修改。

### 3.5 取消滑雪局

POST `/events/:id/cancel`

### 3.6 结束滑雪局

POST `/events/:id/finish`

## 4. 加入申请

### 4.1 申请加入

POST `/events/:id/apply`

请求：

```json
{
  "skiLevel": "intermediate",
  "skiType": "snowboard",
  "hasCar": false,
  "canCarryPeople": false,
  "departArea": "海淀",
  "message": "单板中级，能连续换刃，想一起刷道互拍"
}
```

### 4.2 我的申请

GET `/join-requests/my`

### 4.3 某个滑雪局的申请列表

GET `/events/:id/applications`

仅发起人可看。

### 4.4 同意申请

POST `/join-requests/:id/approve`

### 4.5 拒绝申请

POST `/join-requests/:id/reject`

请求：

```json
{
  "reason": "人数已满"
}
```

## 5. 我的行程

### 5.1 我发起的滑雪局

GET `/trips/created`

### 5.2 我加入的滑雪局

GET `/trips/joined`

### 5.3 待确认行程

GET `/trips/pending`

### 5.4 已结束行程

GET `/trips/finished`

## 6. 局内消息

### 6.1 消息列表

GET `/events/:id/messages`

### 6.2 发送消息

POST `/events/:id/messages`

请求：

```json
{
  "messageType": "text",
  "content": "大家好，周六见"
}
```

## 7. 评价

### 7.1 创建评价

POST `/reviews`

请求：

```json
{
  "eventId": 1,
  "revieweeId": 2,
  "score": 5,
  "positiveTags": ["准时", "友好", "水平真实"],
  "negativeTags": [],
  "content": "节奏合适，下次还愿意一起滑",
  "isAnonymous": false
}
```

### 7.2 用户评价列表

GET `/users/:id/reviews`

## 8. 举报

### 8.1 提交举报

POST `/reports`

请求：

```json
{
  "targetType": "user",
  "targetId": 2,
  "reason": "言语不适",
  "content": "详细说明"
}
```

## 9. 字典

### 9.1 雪场列表

GET `/dict/resorts`

### 9.2 标签字典

GET `/dict/tags`

### 9.3 城市列表

GET `/dict/cities`
