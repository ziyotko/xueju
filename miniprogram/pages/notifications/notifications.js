const api = require('../../utils/api')

Page({
  data: {
    notifications: [],
    loading: false
  },

  onShow() {
    this.load()
  },

  async load() {
    this.setData({ loading: true })
    try {
      const data = await api.notifications()
      this.setData({ notifications: data.list || [] })
    } catch (error) {
      this.setData({ notifications: [] })
    } finally {
      this.setData({ loading: false })
    }
  },

  async markRead() {
    try {
      await api.markNotificationsRead()
      wx.showToast({ title: '已全部已读', icon: 'success' })
      this.load()
    } catch (error) {}
  },

  clearAll() {
    wx.showModal({
      title: '清空通知？',
      success: async (res) => {
        if (!res.confirm) return
        try {
          await api.clearNotifications()
          this.load()
        } catch (error) {}
      }
    })
  }
})
