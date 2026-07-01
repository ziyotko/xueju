# 交互闭环修复说明

本版针对以下问题做了集中修复：

1. 行程、消息、我的页面底部被自定义 TabBar 遮挡
   - 增加 `safe-tab-page` 和 `tabbar-spacer`。
   - 行程、首页、我的页面都补了底部滚动占位。
   - 消息详情页改为非 Tab 页面，输入框不再和 TabBar 抢空间。

2. 占位图片改为 Unsplash 真实图片
   - `data/mock.js` 中统一使用 `https://images.unsplash.com/...`。
   - 上线微信小程序时需要在小程序后台配置 `images.unsplash.com` 合法域名。

3. 消息页面改为真实交互闭环
   - 新增 `pages/chat/index/index` 消息会话列表。
   - 原 `pages/chat/room/room` 改为局内群聊详情。
   - 支持输入发送、回车发送、快捷动作发送、自动模拟回复、清空本地聊天记录。
   - 消息按 eventId 存入本地 storage。

4. 补充必要二级页面
   - 搜索：`pages/search/search`
   - 筛选：`pages/filter/filter`
   - 城市选择：`pages/city/select/select`
   - 通知：`pages/notifications/notifications`
   - 收藏：`pages/favorites/favorites`
   - 设置：`pages/settings/settings`
   - 举报反馈：`pages/report/report`
   - 雪友主页：`pages/user/detail/detail`

5. 主要按钮交互补齐
   - 首页城市、搜索、通知、快捷入口、筛选、卡片、加入按钮可点。
   - 详情页收藏、分享提示、更多操作、关注、成员主页、申请、申请列表、群聊可点。
   - 发布页 picker、标签、开关、人数、提交可用，并能新增本地行程。
   - 申请页提交后进入我的行程待确认。
   - 申请审核页同意/拒绝可持久化。
   - 行程页 tab 可切换。
   - 我的页设置、通知、收藏、编辑资料可点。
   - 评价页提交后写入本地评价。

说明：当前仍是前端本地闭环 MVP，后续接后端时可将 `utils/store.js` 替换为接口请求。
