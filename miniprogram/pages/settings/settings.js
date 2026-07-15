const api = require('../../utils/api')
const { openPrivacyContract } = require('../../utils/privacy')

Page({
  data: { deleting: false },

  goProfileEdit() { wx.navigateTo({ url: '/pages/profile/edit/edit' }) },
  goFavorites() { wx.navigateTo({ url: '/pages/favorites/favorites' }) },
  goNotifications() { wx.navigateTo({ url: '/pages/notifications/notifications' }) },
  goReport() { wx.navigateTo({ url: '/pages/report/report?targetType=app&targetId=0' }) },

  async showPrivacy() {
    try {
      await openPrivacyContract()
    } catch (error) {
      wx.showToast({ title: '暂时无法打开隐私保护指引', icon: 'none' })
    }
  },

  deleteAccount() {
    if (this.data.deleting) return
    wx.showModal({
      title: '确认注销账号？',
      content: '注销后个人资料会匿名化，进行中的行程会取消，申请、成员关系、收藏和关注会清理。该操作不可撤销。',
      confirmText: '确认注销',
      confirmColor: '#EF4444',
      success: async (result) => {
        if (!result.confirm || this.data.deleting) return
        this.setData({ deleting: true })
        wx.showLoading({ title: '正在注销', mask: true })
        try {
          await api.deleteMe()
          const app = getApp()
          app.globalData.token = ''
          app.globalData.user = null
          wx.clearStorageSync()
          wx.hideLoading()
          wx.showToast({ title: '账号已注销', icon: 'success' })
          setTimeout(() => wx.reLaunch({ url: '/pages/index/index' }), 500)
        } catch (error) {
          wx.hideLoading()
        } finally {
          this.setData({ deleting: false })
        }
      }
    })
  }
})
