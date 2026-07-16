const api = require('../../../utils/api')

function markTags(list, selected) {
  return list.map((text) => ({ text, selected: selected.includes(text) }))
}

function currentUserId() {
  const app = getApp()
  const user = app.globalData.user || wx.getStorageSync('xueju_user') || {}
  return Number(user.id || 0)
}

function memberName(member) {
  return member.name || member.nickname || '雪友'
}

function mapReviewTarget(member) {
  const name = memberName(member)
  return {
    id: Number(member.id || member.userId),
    name,
    initial: member.initial || name.slice(0, 1),
    avatarUrl: member.avatarUrl || '',
    meta: member.level || [member.skiType, member.skiLevel].filter(Boolean).join(' · ') || '同行成员'
  }
}

Page({
  data: {
    eventId: 1,
    revieweeId: 0,
    event: {},
    targets: [],
    selectedTarget: null,
    targetMeta: '滑雪局 · 同行成员',
    score: 5,
    scores: [1, 2, 3, 4, 5],
    positiveTags: markTags(['准时', '友好', '水平真实', '沟通顺畅', '安全意识好', '愿意再次同滑'], ['准时', '友好']),
    negativeTags: markTags(['爽约', '迟到严重', '水平虚假', '临时改计划', '言语不适', '危险行为'], []),
    content: '',
    anonymous: false,
    submitting: false
  },

  async onLoad(options) {
    const eventId = Number(options.eventId || 1)
    const preferredUserId = Number(options.userId || 0)
    let event = {}
    try { event = await api.event(eventId) } catch (error) {}

    const myId = currentUserId()
    const targets = (event.members || [])
      .map(mapReviewTarget)
      .filter((item) => item.id && item.id !== myId)
    const selectedTarget = targets.find((item) => item.id === preferredUserId) || targets[0] || null

    this.setData({
      eventId,
      event,
      targets,
      selectedTarget,
      revieweeId: selectedTarget ? selectedTarget.id : 0,
      targetMeta: `${event.resort || '滑雪局'} · ${selectedTarget ? selectedTarget.name : '暂无可评价成员'}`
    })
  },

  selectTarget(event) {
    const id = Number(event.currentTarget.dataset.id)
    const selectedTarget = this.data.targets.find((item) => item.id === id)
    if (!selectedTarget) return
    this.setData({
      selectedTarget,
      revieweeId: selectedTarget.id,
      targetMeta: `${this.data.event.resort || '滑雪局'} · ${selectedTarget.name}`
    })
  },

  setScore(event) { this.setData({ score: Number(event.currentTarget.dataset.score) }) },
  togglePositive(event) { this.toggleTag('positiveTags', Number(event.currentTarget.dataset.index)) },
  toggleNegative(event) { this.toggleTag('negativeTags', Number(event.currentTarget.dataset.index)) },
  toggleTag(key, index) { this.setData({ [key]: this.data[key].map((item, i) => i === index ? { ...item, selected: !item.selected } : item) }) },
  onInput(event) { this.setData({ content: event.detail.value }) },
  onAnonymous(event) { this.setData({ anonymous: event.detail.value }) },

  async submitReview() {
    if (this.data.submitting) return
    if (!this.data.revieweeId) { wx.showToast({ title: '请选择评价对象', icon: 'none' }); return }
    if (!this.data.content.trim()) { wx.showToast({ title: '请填写评价内容', icon: 'none' }); return }

    const positiveTags = this.data.positiveTags.filter((item) => item.selected).map((item) => item.text)
    const negativeTags = this.data.negativeTags.filter((item) => item.selected).map((item) => item.text)
    const payload = {
      eventId: this.data.eventId,
      revieweeId: this.data.revieweeId,
      score: this.data.score,
      positiveTags,
      negativeTags,
      content: this.data.content.trim(),
      isAnonymous: this.data.anonymous
    }

    this.setData({ submitting: true })
    try {
      await api.createReview(payload)
      wx.showToast({ title: '评价已提交', icon: 'success' })
      setTimeout(() => wx.switchTab({ url: '/pages/trips/index/index' }), 600)
    } catch (error) {
      wx.showToast({ title: (error && error.message) || '评价提交失败', icon: 'none' })
    } finally {
      this.setData({ submitting: false })
    }
  }
})
