const api = require('../../utils/api')

Page({
  data: {
    targetType: 'app',
    targetId: 0,
    reasons: ['骚扰或不适内容', '虚假行程', '爽约风险', '危险行为', '其他问题'],
    reason: '骚扰或不适内容',
    content: ''
  },
  onLoad(options) {
    this.setData({ targetType: options.targetType || 'app', targetId: Number(options.targetId || 0) })
  },
  setReason(event) { this.setData({ reason: event.currentTarget.dataset.value }) },
  onInput(event) { this.setData({ content: event.detail.value }) },
  async submit() {
    if (!this.data.content.trim()) {
      wx.showToast({ title: '请填写详细说明', icon: 'none' })
      return
    }
    try {
      await api.createReport({ targetType: this.data.targetType, targetId: this.data.targetId, reason: this.data.reason, content: this.data.content })
      wx.showToast({ title: '已提交', icon: 'success' })
      setTimeout(() => wx.navigateBack(), 500)
    } catch (error) {
    }
  }
})
