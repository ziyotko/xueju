const { events } = require("../../../data/mock")

Page({
  data: {
    events,
    upcoming: events.slice(0, 2),
    tabs: ["我发起的", "我加入的", "待确认", "已结束"],
    active: 1
  },

  goCreate() {
    wx.navigateTo({ url: "/pages/event/create/create" })
  },

  goDetail(event) {
    const id = event.currentTarget.dataset.id
    wx.navigateTo({ url: `/pages/event/detail/detail?id=${id}` })
  }
})
