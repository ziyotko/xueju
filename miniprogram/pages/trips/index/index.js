const api = require('../../../utils/api')

const tabs = [
  { label: '我发起的', kind: 'created', title: '我发起的行程', empty: '还没有发布过雪局' },
  { label: '我加入的', kind: 'joined', title: '我加入的行程', empty: '还没有加入的行程' },
  { label: '待确认', kind: 'pending', title: '待确认申请', empty: '暂无待确认申请' },
  { label: '已结束', kind: 'finished', title: '已结束行程', empty: '暂无已结束行程' }
]

function statusBadge(item, active) {
  if (item.status === 'cancelled') return '已取消'
  if (active === 0) return item.status === 'finished' ? '已结束' : '我发起的'
  if (active === 1) return '已加入'
  if (active === 2) return '待确认'
  if (active === 3) return item.status === 'cancelled' ? '已取消' : '已结束'
  return item.badge || ''
}

function tripView(item, active) {
  const initials = item.memberInitials && item.memberInitials.length
    ? item.memberInitials
    : [item.hostInitial || (item.host || '雪').slice(0, 1)]
  return {
    ...item,
    badge: statusBadge(item, active),
    memberInitials: initials.slice(0, 4),
    displayAvatars: item.memberAvatars && item.memberAvatars.length
      ? item.memberAvatars.slice(0, 4)
      : initials.slice(0, 4).map((initial, index) => ({ id: `fallback-${index}`, initial, avatarUrl: '' })),
    summaryText: active === 2 ? '有待确认申请，请进入详情处理' : `${item.joinedText || ''}，${item.spotsText || ''}`,
    canReview: active === 3 && item.status === 'finished'
  }
}

Page({
  data: {
    tabs,
    active: 1,
    currentTitle: tabs[1].title,
    emptyText: tabs[1].empty,
    trips: [],
    loading: false,
    loadError: ''
  },

  onShow() {
    const tabBar = this.getTabBar && this.getTabBar()
    if (tabBar) tabBar.setData({ active: 1 })
    this.loadTrips()
  },

  async loadTrips() {
    const tab = tabs[this.data.active]
    this.setData({ loading: true, loadError: '', currentTitle: tab.title, emptyText: tab.empty })
    try {
      const list = await api.trips(tab.kind)
      this.setData({ trips: list.map((item) => tripView(item, this.data.active)) })
    } catch (error) {
      this.setData({ trips: [], loadError: '加载失败，请稍后再试' })
    } finally {
      this.setData({ loading: false })
    }
  },

  changeTab(event) {
    const active = Number(event.currentTarget.dataset.index)
    if (active === this.data.active) return
    this.setData({ active }, () => this.loadTrips())
  },

  goCreate() {
    wx.navigateTo({ url: '/pages/event/create/create' })
  },

  goDetail(event) {
    wx.navigateTo({ url: `/pages/event/detail/detail?id=${event.currentTarget.dataset.id}` })
  },

  goReview(event) {
    wx.navigateTo({ url: `/pages/review/create/create?eventId=${event.currentTarget.dataset.id}` })
  }
})
