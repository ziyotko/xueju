const { addReview, getEventById } = require('../../../utils/store')
const api = require('../../../utils/api')

function markTags(list, selected) {
  return list.map((text) => ({ text, selected: selected.includes(text) }))
}

Page({
  data: {
    eventId: 1,
    revieweeId: 0,
    event: {},
    targetMeta: '滑雪局 · 发起人',
    score: 5,
    scores: [1, 2, 3, 4, 5],
    positiveTags: markTags(['准时', '友好', '水平真实', '沟通顺畅', '安全意识好', '愿意再次同滑'], ['准时', '友好']),
    negativeTags: markTags(['爽约', '迟到严重', '水平虚假', '临时改计划', '言语不适', '危险行为'], []),
    content: '节奏合适，集合准时，沟通也很顺畅。',
    anonymous: false
  },

  async onLoad(options) {
    const eventId = Number(options.eventId || 1)
    let event = {}
    try { event = await api.event(eventId) } catch (error) { event = getEventById(eventId) }
    const revieweeId = Number(options.userId || event.creatorId || 1)
    this.setData({ eventId, event, revieweeId, targetMeta: `${event.resort || '滑雪局'} · ${event.host || '发起人'}` })
  },
  setScore(event) { this.setData({ score: Number(event.currentTarget.dataset.score) }) },
  togglePositive(event) { this.toggleTag('positiveTags', Number(event.currentTarget.dataset.index)) },
  toggleNegative(event) { this.toggleTag('negativeTags', Number(event.currentTarget.dataset.index)) },
  toggleTag(key, index) { this.setData({ [key]: this.data[key].map((item, i) => i === index ? { ...item, selected: !item.selected } : item) }) },
  onInput(event) { this.setData({ content: event.detail.value }) },
  onAnonymous(event) { this.setData({ anonymous: event.detail.value }) },
  async submitReview() {
    if (!this.data.content.trim()) { wx.showToast({ title: '请填写评价内容', icon: 'none' }); return }
    const positiveTags = this.data.positiveTags.filter((item) => item.selected).map((item) => item.text)
    const negativeTags = this.data.negativeTags.filter((item) => item.selected).map((item) => item.text)
    const payload = { eventId: this.data.eventId, revieweeId: this.data.revieweeId, score: this.data.score, positiveTags, negativeTags, content: this.data.content, isAnonymous: this.data.anonymous }
    try {
      await api.createReview(payload)
      wx.showToast({ title: '评价已提交', icon: 'success' })
    } catch (error) {
      addReview({ eventId: this.data.eventId, eventTitle: this.data.event.resort, score: this.data.score, positive: positiveTags, negative: negativeTags, content: this.data.content, anonymous: this.data.anonymous })
      wx.showToast({ title: '评价已保存本地', icon: 'none' })
    }
    setTimeout(() => wx.switchTab({ url: '/pages/trips/index/index' }), 600)
  }
})
