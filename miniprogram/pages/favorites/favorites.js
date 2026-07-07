const api = require('../../utils/api')

function findEventById(events, id) {
  return (events || []).find((item) => Number(item.id) === Number(id))
}

Page({
  data: {
    events: [],
    loading: false
  },

  async onShow() {
    this.setData({ loading: true })
    try {
      const page = await api.favorites()
      this.setData({ events: page.list || [] })
    } catch (error) {
      this.setData({ events: [] })
    } finally {
      this.setData({ loading: false })
    }
  },

  goDetail(event) {
    wx.navigateTo({ url: `/pages/event/detail/detail?id=${event.detail.id}` })
  },

  goApply(event) {
    const id = event.detail.id
    const target = findEventById(this.data.events, id)
    if (target && target.isCreatedByMe) {
      wx.showToast({ title: '不能申请自己发布的行程', icon: 'none' })
      return
    }
    wx.navigateTo({ url: `/pages/event/apply/apply?eventId=${id}` })
  },

  goHome() {
    wx.switchTab({ url: '/pages/index/index' })
  }
})
