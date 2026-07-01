const { getEvents, getFavorites } = require('../../utils/store')

function findEventById(events, id) {
  return (events || []).find((item) => Number(item.id) === Number(id))
}

Page({
  data: {
    events: []
  },

  onShow() {
    const fav = getFavorites()
    this.setData({ events: getEvents().filter((item) => fav.includes(Number(item.id))) })
  },

  goDetail(e) {
    wx.navigateTo({ url: `/pages/event/detail/detail?id=${e.detail.id}` })
  },

  goApply(e) {
    const id = e.detail.id
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
