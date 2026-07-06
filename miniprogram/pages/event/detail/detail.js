const api = require('../../../utils/api')
const { mockMembers, getEventById, toggleFavorite, isFavorite, deleteEvent: deleteStoredEvent } = require('../../../utils/store')

function currentUser() {
  const app = getApp()
  return app.globalData.user || wx.getStorageSync('xueju_user') || {}
}

function buildActionState(event, requests) {
  const user = currentUser()
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
      this.setData({
        event,
        members,
        memberCount: members.length,
        isFavorite: isFavorite(id),
        actionState: buildActionState(event, [])
      })
    }
  },

  goBack() { wx.navigateBack() },

  onShareAppMessage() {
    return { title: this.data.event.resort || '雪局行程', path: `/pages/event/detail/detail?id=${this.data.event.id}` }
  },

  tapShare() {
    wx.showToast({ title: '请点右上角转发给雪友', icon: 'none' })
  },

  openMore() {
    const isCreator = this.data.actionState && this.data.actionState.isCreator
    const canFinish = isCreator && !['finished', 'cancelled', 'removed'].includes(this.data.event.status)
    const itemList = isCreator
      ? (canFinish ? ['结束行程', '删除该雪局', '复制集合信息', '查看发起人主页'] : ['删除该雪局', '复制集合信息', '查看发起人主页'])
      : ['举报该行程', '复制集合信息', '查看发起人主页']

    wx.showActionSheet({
      itemList,
      success: (res) => {
        if (isCreator && canFinish && res.tapIndex === 0) { this.confirmFinish(); return }
        if (isCreator && res.tapIndex === (canFinish ? 1 : 0)) { this.confirmDelete(); return }
        if (!isCreator && res.tapIndex === 0) { this.reportEvent(); return }

        const copyIndex = isCreator ? (canFinish ? 2 : 1) : 1
        const profileIndex = isCreator ? (canFinish ? 3 : 2) : 2
        if (res.tapIndex === copyIndex) this.copyMeetInfo()
        if (res.tapIndex === profileIndex) this.goCreatorProfile()
      }
    })
  },

  confirmFinish() {
    wx.showModal({
      title: '结束行程？',
      content: '结束后成员可以提交滑后评价，行程将进入已结束列表。',
      confirmText: '结束',
      success: async (res) => {
        if (!res.confirm) return
        try {
          await api.finishEvent(this.data.event.id)
          wx.showToast({ title: '行程已结束', icon: 'success' })
          this.loadDetail(this.data.event.id)
        } catch (error) {}
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

  copyMeetInfo() {
    const text = `${this.data.event.resort} · ${this.data.event.date} ${this.data.event.time} · ${this.data.event.meetPlace}`
    wx.setClipboardData({ data: text })
  },

  reportEvent() {
    wx.navigateTo({ url: `/pages/report/report?targetType=event&targetId=${this.data.event.id}` })
  },

  goCreatorProfile() {
    const creatorId = this.data.event.creatorId || 1
    wx.setStorageSync(`xueju_user_detail_${creatorId}`, {
      id: creatorId,
      nickname: this.data.event.creatorName || this.data.event.host,
      creditScore: 5
    })
    wx.navigateTo({ url: `/pages/user/detail/detail?id=${creatorId}` })
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
  goUser(event) {
    const id = event.currentTarget.dataset.id
    const member = (this.data.members || []).find((item) => Number(item.id || item.userId) === Number(id))
    if (member) wx.setStorageSync(`xueju_user_detail_${id}`, member)
    wx.navigateTo({ url: `/pages/user/detail/detail?id=${id}` })
  },
  followHost() { wx.showToast({ title: '已关注发起人', icon: 'success' }) }
})
