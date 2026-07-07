const api = require('../../../utils/api')

const statusText = {
  recruiting: '招募中',
  full: '已满员',
  finished: '已结束',
  cancelled: '已取消',
  removed: '已下架'
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
      this.setData({ conversations: [], unreadCount: 0 })
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
    } catch (error) {}
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
