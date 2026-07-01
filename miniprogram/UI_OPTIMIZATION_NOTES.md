# 雪局小程序 UI 优化说明

本次主要针对首页视觉粗糙、图片不贴合滑雪主题、按钮换行、卡片排版不稳定、TabBar 选中态错误等问题做了调整。

## 已完成

1. 首页高保真重构
   - 重构 `pages/index/index.*`
   - 首页背景改为冰雪浅色渐变
   - Banner 标题改为明确两行展示
   - Banner 按钮改为 `view` 实现，避免微信默认 button 样式导致换行
   - 补充推荐/最新 Tab 交互事件

2. Banner 组件优化
   - 重构 `components/home-banner/*`
   - 使用滑雪主题图
   - 增加深色蒙层和渐变按钮
   - 修复「发布滑雪行程」按钮换行问题

3. 城市搜索栏优化
   - 重构 `components/city-search-bar/*`
   - 统一搜索框、通知按钮阴影、圆角、对齐
   - 增加点击事件抛出

4. 首页筛选栏优化
   - 重构 `components/home-section-tabs/*`
   - 增加推荐/最新切换事件
   - 增加筛选点击事件

5. 滑雪局卡片优化
   - 重构 `components/ski-event-card/*`
   - 右侧图片尺寸固定
   - 人数状态移到图片下方
   - 发起人、头像、加入按钮底部稳定对齐
   - 标签、标题、信息行统一样式
   - 卡片支持 `displayTags` 和 `memberInitials`

6. 图片资源替换
   - 重新生成以下滑雪主题本地图片：
     - `assets/images/banner-ski.jpg`
     - `assets/images/resort-wanlong.jpg`
     - `assets/images/resort-nanshan.jpg`
     - `assets/images/resort-yunding.jpg`
     - `assets/images/resort-taiwu.jpg`
   - 不再使用代码/电脑/办公类图片

7. TabBar 状态修复
   - 修复聊天页 active index：2
   - 修复个人页 active index：3

8. 外链图片处理
   - 个人主页封面从外部 Unsplash 链接改成本地滑雪图片，避免小程序域名白名单问题。

## 建议后续

1. 在微信开发者工具中重新构建 npm。
2. 清理模拟器缓存后重新预览，避免旧图片缓存影响效果。
3. 后续所有新页面继续使用 `app-icon` 组件，不要直接手写 icon。
4. 如果要更接近真实产品，可替换为真实雪场照片。
