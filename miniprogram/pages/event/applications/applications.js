const api = require('../../../utils/api')
const { getJoinRequests, updateJoinRequest, getEventById } = require('../../../utils/store')

Page({
  data: {
    eventId: 1,
    event: {},
    summaryText: '',
    requests: [],
    statusText: { pending: '待审核', approved: '已通过', rejected: '已拒绝' }
  },

  async onLoad(options) {
    const eventId = Number(options.eventId || 1)
    let event = {}
    try { event = await api.event(eventId) } catch (error) { event = getEventById(eventId) }
    this.setData({ eventId, event, summaryText: `当前 ${event.joinedText || '已 1 人'}，最多 ${event.maxPeople || 4} 人 · 审核通过后可进入局内群聊` })
    this.loadRequests()
  },

  onShow() { this.loadRequests() },

  async loadRequests() {
    try {
      const requests = await api.applications(this.data.eventId)
      this.setData({ requests: requests.map((item) => ({ ...item, statusLabel: this.data.statusText[item.status] || '待审核' })) })
    } catch (error) {
      const requests = getJoinRequests()
        .filter((item) => Number(item.eventId || this.data.eventId) === Number(this.data.eventId))
        .map((item) => ({ ...item, statusLabel: this.data.statusText[item.status] || '待审核' }))
      this.setData({ requests })
    }
  },

  async approve(event) {
    const id = Number(event.currentTarget.dataset.id)
    try { await api.reviewApplication(id, true) } catch (error) { updateJoinRequest(id, 'approved') }
    wx.showToast({ title: '已同意', icon: 'success' })
    this.loadRequests()
  },

  async reject(event) {
    const id = Number(event.currentTarget.dataset.id)
    try { await api.reviewApplication(id, false, '发起人已拒绝') } catch (error) { updateJoinRequest(id, 'rejected') }
    wx.showToast({ title: '已拒绝', icon: 'none' })
    this.loadRequests()
  },

  goChat() { wx.navigateTo({ url: `/pages/chat/room/room?eventId=${this.data.eventId}` }) },
  goUser(event) {
    const id = event.currentTarget.dataset.id || 1
    const request = (this.data.requests || []).find((item) => Number(item.applicantId || item.id) === Number(id))
    if (request) wx.setStorageSync(`xueju_user_detail_${id}`, request)
    wx.navigateTo({ url: `/pages/user/detail/detail?id=${id}` })
  }
})
