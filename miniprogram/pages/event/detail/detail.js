const api = require('../../../utils/api')

function currentUser() {
  const app = getApp()
  return app.globalData.user || wx.getStorageSync('xueju_user') || {}
}

function buildActionState(event, requests, options = {}) {
  const user = currentUser()
  const myRequest = (requests || []).find((item) => Number(item.eventId) === Number(event.id))
  const isCreator = Number(event.creatorId) === Number(user.id) || (!options.fromShare && !!event.isCreatedByMe)
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

function enrichEvent(event) {
  const baseTag = [event.board, event.level].filter(Boolean).join('')
  const tags = []
  ;[baseTag].concat(event.displayTags || [], event.tags || []).forEach((item) => {
    if (item && !tags.includes(item)) tags.push(item)
  })
  if (event.allowCarPool && !tags.includes('可拼车')) tags.push('可拼车')
  if (event.allowRoomShare && !tags.includes('可拼房')) tags.push('可拼房')
  if (event.allowBeginner && !tags.includes('新手友好')) tags.push('新手友好')
  if (event.sameGenderOnly && !tags.includes('同性局')) tags.push('同性局')

  return {
    ...event,
    detailTags: tags,
    trafficCostText: [event.traffic, event.costDesc].filter(Boolean).join('，'),
    hostCreditText: [event.credit, event.creatorEventCount ? `发起局数 ${event.creatorEventCount}` : ''].filter(Boolean).join('　')
  }
}

Page({
  data: {
    event: {},
    members: [],
    memberCount: 0,
    isFavorite: false,
    canFollowHost: false,
    fromShare: false,
    actionState: {},
    rules: ['遵守雪场通行和水平要求，安全第一', '不线下爽约，互相尊重', '有任何问题及时沟通']
  },

  onLoad(options) {
    const id = Number(options.id || 1)
    this.setData({ fromShare: options.fromShare === '1' })
    this.loadDetail(id)
  },

  onShow() {
    if (this.data.event && this.data.event.id) this.loadDetail(this.data.event.id)
  },

  async loadDetail(id) {
    try {
      const event = enrichEvent(await api.event(id))
      let requests = []
      let favoriteIds = []
      try { requests = await api.myJoinRequests() } catch (error) {}
      try {
        const favorites = await api.favorites()
        favoriteIds = (favorites.list || []).map((item) => Number(item.id))
      } catch (error) {}
      const members = event.members || []
      const actionState = buildActionState(event, requests, { fromShare: this.data.fromShare })
      this.setData({
        event,
        members,
        memberCount: members.length,
        isFavorite: favoriteIds.includes(Number(id)),
        canFollowHost: !actionState.isCreator,
        actionState
      })
    } catch (error) {
      this.setData({
        event: {},
        members: [],
        memberCount: 0,
        isFavorite: false,
        canFollowHost: false,
        actionState: {}
      })
    }
  },

  onShareAppMessage() {
    return { title: this.data.event.resort || '雪局行程', path: `/pages/event/detail/detail?id=${this.data.event.id}&fromShare=1` }
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
        } catch (error) {}
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

  async toggleCollect() {
    const collected = !this.data.isFavorite
    try {
      await api.setFavorite(this.data.event.id, collected)
      this.setData({ isFavorite: collected })
      wx.showToast({ title: collected ? '已收藏' : '已取消收藏', icon: 'success' })
    } catch (error) {}
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
  async followHost() {
    const creatorId = this.data.event.creatorId
    if (!creatorId || !this.data.canFollowHost) return
    const user = currentUser()
    if (Number(creatorId) === Number(user.id)) return
    try {
      await api.followUser(creatorId, true)
      wx.showToast({ title: '已关注发起人', icon: 'success' })
    } catch (error) {}
  }
})
