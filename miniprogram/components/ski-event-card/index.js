Component({
  properties: {
    event: {
      type: Object,
      value: {}
    }
  },

  methods: {
    onCardTap() {
      this.triggerEvent("cardtap", { id: this.data.event.id })
    },

    onJoinTap() {
      if (this.data.event && this.data.event.isCreatedByMe) {
        wx.showToast({ title: '不能申请自己发布的行程', icon: 'none' })
        return
      }
      this.triggerEvent("jointap", { id: this.data.event.id })
    }
  }
})
