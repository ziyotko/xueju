const { events, members } = require("../../../data/mock")

Page({
  data: {
    event: events[0],
    members,
    memberCount: members.length,
    rules: [
      "遵守雪场通行和水平要求，安全第一",
      "不线下爽约，互相尊重",
      "费用 AA，按实际产生均摊",
      "有任何问题及时沟通"
    ]
  },

  onLoad(options) {
    const id = Number(options.id || 1)
    const event = events.find((item) => item.id === id) || events[0]
    this.setData({ event })
  },

  goChat() {
    wx.switchTab({ url: "/pages/chat/room/room" })
  }
})
