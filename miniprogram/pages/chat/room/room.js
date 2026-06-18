const { messages } = require("../../../data/mock")
const { CONTENT_RISK_MESSAGE } = require("../../../constants/compliance")

Page({
  data: {
    messages,
    inputValue: "",
    quickActions: [
      { icon: "ⓘ", text: "集合信息" },
      { icon: "￥", text: "费用说明" },
      { icon: "✎", text: "雪场须知" },
      { icon: "▤", text: "行程备注" },
      { icon: "▣", text: "相册" },
      { icon: "⌖", text: "位置" },
      { icon: "▥", text: "投票" },
      { icon: "•••", text: "更多" }
    ],
    contentRiskMessage: CONTENT_RISK_MESSAGE
  },

  onInput(event) {
    this.setData({ inputValue: event.detail.value })
  },

  send() {
    if (!this.data.inputValue.trim()) {
      return
    }
    wx.showToast({ title: "消息接口待接入", icon: "none" })
  }
})
