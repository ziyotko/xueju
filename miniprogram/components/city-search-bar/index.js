Component({
  properties: {
    city: {
      type: String,
      value: "北京"
    },
    placeholder: {
      type: String,
      value: "搜索雪场 / 日期 / 关键词"
    }
  },
  methods: {
    onCityTap() {
      this.triggerEvent("citytap")
    },
    onSearchTap() {
      this.triggerEvent("searchtap")
    },
    onNotifyTap() {
      this.triggerEvent("notifytap")
    }
  }
})
