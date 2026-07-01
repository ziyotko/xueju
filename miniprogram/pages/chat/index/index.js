const { getEvents, getMessages, getNotifications, markNotificationsRead } = require('../../../utils/store')

Page({
  data: {
    conversations: [],
    unreadCount: 0
  },

  onShow() {
    const tabBar = this.getTabBar && this.getTabBar()
    if (tabBar) tabBar.setData({ active: 2 })
    this.loadData()
  },

  loadData() {
    const events = getEvents().slice(0, 4)
    const conversations = events.map((event, index) => {
      const messages = getMessages(event.id)
      const last = messages[messages.length - 1]
      return {
        id: `conv-${event.id}`,
        eventId: event.id,
        title: event.resort,
        image: event.image,
        time: last ? last.time : event.time,
        lastMessage: last ? `${last.displayName || last.name}：${last.content}` : '还没有消息，先打个招呼吧',
        status: event.badge || '招募中',
        memberText: event.people || event.joinedText,
        unread: index === 0 ? 2 : 0
      }
    })
    const unreadCount = getNotifications().filter((item) => !item.read).length
    this.setData({ conversations, unreadCount })
  },

  openRoom(event) {
    const id = event.currentTarget.dataset.id
    wx.navigateTo({ url: `/pages/chat/room/room?eventId=${id}` })
  },

  markAllRead() {
    markNotificationsRead()
    this.setData({ unreadCount: 0, conversations: this.data.conversations.map((item) => ({ ...item, unread: 0 })) })
    wx.showToast({ title: '已全部标记已读', icon: 'success' })
  },

  goNotifications() {
    wx.navigateTo({ url: '/pages/notifications/notifications' })
  },

  goHome() {
    wx.switchTab({ url: '/pages/index/index' })
  }
})
