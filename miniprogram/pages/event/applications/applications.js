const api = require('../../../utils/api')

const skiTypeText = { snowboard: '单板', ski: '双板', both: '单双板' }
const skiLevelText = { beginner: '新手', primary: '初级', intermediate: '中级', advanced: '高级' }

Page({
  data: {
    eventId: 1,
    event: {},
    summaryText: '',
    requests: [],
    removingId: 0,
    statusText: { pending: '待审核', approved: '已通过', rejected: '已拒绝', removed: '已移出' }
  },

  async onLoad(options) {
    const eventId = Number(options.eventId || 1)
    let event = {}
    try { event = await api.event(eventId) } catch (error) {}
    const currentText = event.joinedText && event.maxPeople
      ? `当前 ${event.joinedText}，最多 ${event.maxPeople} 人`
      : '审核通过后可进入局内群聊'
    this.setData({ eventId, event, summaryText: currentText })
    this.loadRequests()
  },

  onShow() { this.loadRequests() },

  async loadRequests() {
    try {
      const requests = await api.applications(this.data.eventId)
      this.setData({ requests: requests.map((item) => {
        const name = item.name || item.nickname || '雪友'
        const skiType = skiTypeText[item.skiType] || item.skiType || ''
        const skiLevel = skiLevelText[item.level] || skiLevelText[item.skiLevel] || item.level || item.skiLevel || ''
        const metaText = [skiType, skiLevel, item.departArea].filter(Boolean).join(' · ')
        return { ...item, initial: item.initial || name.slice(0, 1), metaText, statusLabel: this.data.statusText[item.status] || '待审核' }
      }) })
    } catch (error) {
      this.setData({ requests: [] })
    }
  },

  async approve(event) {
    const id = Number(event.currentTarget.dataset.id)
    try {
      await api.reviewApplication(id, true)
      wx.showToast({ title: '已同意', icon: 'success' })
      this.loadRequests()
    } catch (error) {}
  },

  async reject(event) {
    const id = Number(event.currentTarget.dataset.id)
    try {
      await api.reviewApplication(id, false, '发起人已拒绝')
      wx.showToast({ title: '已拒绝', icon: 'none' })
      this.loadRequests()
    } catch (error) {}
  },

  removeMember(event) {
    const userId = Number(event.currentTarget.dataset.userId || 0)
    if (!userId || this.data.removingId) return
    wx.showModal({
      title: '移出该成员？',
      content: '移出后该成员会立即失去群聊权限，且不能再次申请该行程。',
      confirmText: '确认移出',
      confirmColor: '#EF4444',
      success: async (result) => {
        if (!result.confirm) return
        this.setData({ removingId: userId })
        try {
          await api.removeEventMember(this.data.eventId, userId)
          wx.showToast({ title: '成员已移出', icon: 'success' })
          await this.loadRequests()
        } catch (error) {
        } finally {
          this.setData({ removingId: 0 })
        }
      }
    })
  },

  goChat() { wx.navigateTo({ url: `/pages/chat/room/room?eventId=${this.data.eventId}` }) },
  goUser(event) {
    const id = event.currentTarget.dataset.id || 1
    const request = (this.data.requests || []).find((item) => Number(item.applicantId || item.id) === Number(id))
    if (request) wx.setStorageSync(`xueju_user_detail_${id}`, request)
    wx.navigateTo({ url: `/pages/user/detail/detail?id=${id}` })
  }
})
