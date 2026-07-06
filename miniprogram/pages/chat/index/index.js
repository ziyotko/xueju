const api = require('../../../utils/api')
const { getEvents, getMessages, getNotifications, markNotificationsRead } = require('../../../utils/store')

const statusText = {
  recruiting: '招募中',
  full: '已满员',
  finished: '已结束',
  cancelled: '已取消',
  removed: '已下架'
}

function fallbackConversations() {
  return getEvents().slice(0, 4).map((event) => {
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
      unread: 0
    }
  })
}

Page({
  data: {
    conversations: [],
    unreadCount: 0,
    loading: false
  },

  onShow() {
    const tabBar = this.getTabBar && this.getTabBar()
    if (tabBar) tabBar.setData({ active: 3 })
    this.loadData()
  },

  async loadData() {
    this.setData({ loading: true })
    try {
      const data = await api.conversations()
      const conversations = (data.list || []).map((item) => ({
        ...item,
        status: statusText[item.status] || item.status,
        image: item.image || 'https://images.unsplash.com/photo-1740137660688-3d3f2b5422b6?auto=format&fit=crop&w=900&q=80'
      }))
      this.setData({ conversations, unreadCount: data.unreadCount || 0 })
    } catch (error) {
      const unreadCount = getNotifications().filter((item) => !item.read).length
      this.setData({ conversations: fallbackConversations(), unreadCount })
    } finally {
      this.setData({ loading: false })
    }
  },

  openRoom(event) {
    const id = event.currentTarget.dataset.id
    wx.navigateTo({ url: `/pages/chat/room/room?eventId=${id}` })
  },

  async markAllRead() {
    try {
      await api.markAllConversationsRead()
    } catch (error) {
      markNotificationsRead()
    }
    this.setData({
      unreadCount: 0,
      conversations: this.data.conversations.map((item) => ({ ...item, unread: 0 }))
    })
    wx.showToast({ title: '已全部标记已读', icon: 'success' })
  },

  goNotifications() {
    wx.navigateTo({ url: '/pages/notifications/notifications' })
  },

  goHome() {
    wx.switchTab({ url: '/pages/index/index' })
  }
})
