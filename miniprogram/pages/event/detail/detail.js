const api = require('../../../utils/api')
const { mockMembers, getEventById, toggleFavorite, isFavorite, deleteEvent: deleteStoredEvent } = require('../../../utils/store')

function buildActionState(event, requests) {
  const app = getApp()
  const user = app.globalData.user || wx.getStorageSync('xueju_user') || {}
  const myRequest = (requests || []).find((item) => Number(item.eventId) === Number(event.id))
  const isCreator = Number(event.creatorId) === Number(user.id) || !!event.isCreatedByMe
  const isApproved = myRequest && myRequest.status === 'approved'
  const isPending = myRequest && myRequest.status === 'pending'
  const isFinished = event.status === 'finished'
  return {
    isCreator,
    isApproved,
    isPending,
    canManage: isCreator,
    canChat: isCreator || isApproved,
    canApply: !isCreator && !isApproved && !isPending && !isFinished,
    canReview: isFinished,
    primaryText: isCreator ? '进入群聊' : isApproved ? '进入群聊' : isPending ? '待审核' : isFinished ? '去评价' : '申请加入'
  }
}

Page({
  data: {
    event: {},
    members: mockMembers,
    memberCount: mockMembers.length,
    isFavorite: false,
    actionState: {},
    rules: ['遵守雪场通行和水平要求，安全第一', '不线下爽约，互相尊重', '费用 AA，按实际产生均摊', '有任何问题及时沟通']
  },

  onLoad(options) {
    const id = Number(options.id || 1)
    this.loadDetail(id)
  },

  onShow() {
    if (this.data.event && this.data.event.id) this.loadDetail(this.data.event.id)
  },

  async loadDetail(id) {
    try {
      const event = await api.event(id)
      let requests = []
      try { requests = await api.myJoinRequests() } catch (error) {}
      const members = event.members || []
      this.setData({
        event,
        members,
        memberCount: members.length,
        isFavorite: isFavorite(id),
        actionState: buildActionState(event, requests)
      })
    } catch (error) {
      const event = getEventById(id)
      const members = event.members && event.members.length ? event.members : mockMembers
      this.setData({ event, members, memberCount: members.length, isFavorite: isFavorite(id), actionState: buildActionState(event, []) })
    }
  },

  goBack() { wx.navigateBack() },
  onShareAppMessage() {
    return { title: this.data.event.resort || '雪局行程', path: `/pages/event/detail/detail?id=${this.data.event.id}` }
  },
  tapShare() { wx.showToast({ title: '请点右上角转发给雪友', icon: 'none' }) },
  openMore() {
    const isCreator = this.data.actionState && this.data.actionState.isCreator
    const itemList = isCreator ? ['删除该雪局', '复制集合信息', '查看发起人主页'] : ['举报该行程', '复制集合信息', '查看发起人主页']
    wx.showActionSheet({
      itemList,
      success: (res) => {
        if (res.tapIndex === 0 && isCreator) { this.confirmDelete(); return }
        if (res.tapIndex === 0) wx.navigateTo({ url: `/pages/report/report?targetType=event&targetId=${this.data.event.id}` })
        if (res.tapIndex === 1) {
          const text = `${this.data.event.resort} · ${this.data.event.date} ${this.data.event.time} · ${this.data.event.meetPlace}`
          wx.setClipboardData({ data: text })
        }
        if (res.tapIndex === 2) wx.navigateTo({ url: `/pages/user/detail/detail?id=${this.data.event.creatorId || 1}` })
      }
    })
  },
  confirmDelete() {
    wx.showModal({
      title: '删除雪局？',
      content: '删除后将不再展示在发现页和我的行程中，其他雪友也无法继续查看。',
      confirmText: '删除',
      confirmColor: '#EF4444',
      success: async (res) => {
        if (!res.confirm) return
        try {
          await api.deleteEvent(this.data.event.id)
          wx.showToast({ title: '已删除', icon: 'success' })
          setTimeout(() => wx.navigateBack(), 500)
        } catch (error) {
          if (!this.data.event.isCreatedByMe) return
          deleteStoredEvent(this.data.event.id)
          wx.showToast({ title: '已删除本地演示', icon: 'success' })
          setTimeout(() => wx.navigateBack(), 500)
        }
      }
    })
  },
  toggleCollect() {
    const collected = toggleFavorite(this.data.event.id)
    this.setData({ isFavorite: collected })
    wx.showToast({ title: collected ? '已收藏' : '已取消收藏', icon: 'success' })
  },
  handlePrimaryAction() {
    const state = this.data.actionState
    if (state.isCreator) { this.goApplications(); return }
    if (state.canChat) { this.goChat(); return }
    if (state.isPending) { wx.showToast({ title: '申请正在等待发起人审核', icon: 'none' }); return }
    if (state.canReview) { wx.navigateTo({ url: `/pages/review/create/create?eventId=${this.data.event.id}` }); return }
    this.goApply()
  },
  goApply() {
    if (this.data.actionState && this.data.actionState.isCreator) {
      wx.showToast({ title: '不能申请自己发布的行程', icon: 'none' })
      return
    }
    wx.navigateTo({ url: `/pages/event/apply/apply?eventId=${this.data.event.id}` })
  },
  goApplications() { wx.navigateTo({ url: `/pages/event/applications/applications?eventId=${this.data.event.id}` }) },
  goChat() { wx.navigateTo({ url: `/pages/chat/room/room?eventId=${this.data.event.id}` }) },
  goUser(event) { wx.navigateTo({ url: `/pages/user/detail/detail?id=${event.currentTarget.dataset.id}` }) },
  followHost() { wx.showToast({ title: '已关注发起人', icon: 'success' }) }
})
