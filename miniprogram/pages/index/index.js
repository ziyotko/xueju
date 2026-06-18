const { get } = require("../../utils/request")
const { events } = require("../../data/mock")

Page({
  data: {
    events,
    heroImage: events[0].image,
    health: null,
    loading: false,
    categories: [
      { icon: "⌘", text: "全部" },
      { icon: "◴", text: "周末局" },
      { icon: "▣", text: "同行交通" },
      { icon: "⌂", text: "住宿备注" },
      { icon: "♀", text: "同性同行" },
      { icon: "☆", text: "新手友好" }
    ],
    activeTab: "推荐"
  },

  onLoad() {
    this.loadHealth()
  },

  loadHealth() {
    get("/health")
      .then((health) => {
        this.setData({ health })
      })
      .catch(() => {
        this.setData({ health: null })
      })
  },

  goCreate() {
    wx.navigateTo({ url: "/pages/event/create/create" })
  },

  goDetail(event) {
    const id = event.currentTarget.dataset.id
    wx.navigateTo({ url: `/pages/event/detail/detail?id=${id}` })
  }
})
