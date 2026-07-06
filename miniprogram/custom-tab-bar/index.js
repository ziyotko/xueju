Component({
  data: {
    active: 0,
    items: [
      { pagePath: "/pages/index/index", text: "发现", iconName: "home", type: "tab" },
      { pagePath: "/pages/trips/index/index", text: "行程", iconName: "calendar", type: "tab" },
      { pagePath: "/pages/event/create/create", text: "发布", iconName: "add-circle", type: "page" },
      { pagePath: "/pages/chat/index/index", text: "消息", iconName: "chat", type: "tab" },
      { pagePath: "/pages/profile/profile", text: "我的", iconName: "user", type: "tab" }
    ]
  },

  methods: {
    switchTab(event) {
      const index = event.currentTarget.dataset.index
      const item = this.data.items[index]
      if (item.type === "page") {
        wx.navigateTo({ url: item.pagePath })
        return
      }
      wx.switchTab({ url: item.pagePath })
    }
  }
})
