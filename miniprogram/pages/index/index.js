const api = require('../../utils/api')
const { heroImage } = require('../../data/mock')

function findEventById(events, id) {
  return (events || []).find((item) => Number(item.id) === Number(id))
}

Page({
  data: {
    city: '北京',
    events: [],
    allEvents: [],
    heroImage,
    bannerTitleLines: ['这个周末', '找个水平差不多的人一起滑'],
    health: null,
    loading: false,
    activeFilters: {},
    quickEntries: [
      { iconName: 'app', text: '全部', filter: 'all' },
      { iconName: 'calendar', text: '最新', filter: 'latest' },
      { iconName: 'vehicle', text: '可拼车', filter: 'carpool' },
      { iconName: 'home', text: '可拼房', filter: 'room' },
      { iconName: 'usergroup', text: '同城', filter: 'city' },
      { iconName: 'user-add', text: '新手友好', filter: 'beginner' }
    ],
    tabs: ['推荐', '最新'],
    activeTab: '推荐'
  },

  onLoad(options) {
    if (options.city) this.setData({ city: decodeURIComponent(options.city) })
    this.loadEvents()
    this.loadHealth()
  },

  onShow() {
    const tabBar = this.getTabBar && this.getTabBar()
    if (tabBar) tabBar.setData({ active: 0 })
    this.loadEvents()
  },

  async loadEvents(params = {}) {
    this.setData({ loading: true })
    const filters = { ...(this.data.activeFilters || {}), ...params }

    try {
      const page = await api.events({
        page: 1,
        pageSize: 50,
        city: this.data.city,
        sort: this.data.activeTab === '最新' ? 'latest' : 'recommend',
        ...filters,
        purposeTags: (filters.purposeTags || []).join(',')
      })
      const events = page.list || []
      this.setData({ allEvents: events, events })
    } catch (error) {
      this.setData({ allEvents: [], events: [] })
    } finally {
      this.setData({ loading: false })
    }
  },

  async loadHealth() {
    try {
      const health = await require('../../utils/request').get('/health')
      this.setData({ health })
    } catch (error) {
      this.setData({ health: null })
    }
  },

  onTabChange(event) {
    const active = event.detail.active
    this.setData({ activeTab: active }, () => this.loadEvents())
  },

  openFilter() {
    const filters = encodeURIComponent(JSON.stringify(this.data.activeFilters || {}))
    wx.navigateTo({
      url: `/pages/filter/filter?filters=${filters}`,
      success: (res) => {
        res.eventChannel.emit('initFilters', this.data.activeFilters || {})
        res.eventChannel.on('applyFilters', (filters) => {
          this.setData({ activeFilters: filters || {} }, () => this.loadEvents())
        })
      }
    })
  },

  onQuickEntry(event) {
    const item = event.detail.item
    if (!item) return
    if (item.filter === 'all') {
      this.setData({ activeFilters: {} }, () => this.loadEvents())
      return
    }
    const params = {}
    if (item.filter === 'carpool') params.allowCarPool = true
    if (item.filter === 'room') params.allowRoomShare = true
    if (item.filter === 'beginner') params.allowBeginner = true
    if (item.filter === 'latest') params.sort = 'latest'
    if (item.filter === 'city') params.city = this.data.city
    this.setData({ activeFilters: params }, () => this.loadEvents())
    wx.showToast({ title: `已筛选：${item.text}`, icon: 'none' })
  },

  goCreate() { wx.navigateTo({ url: '/pages/event/create/create' }) },

  goDetail(event) {
    const id = event.detail && event.detail.id ? event.detail.id : event.currentTarget.dataset.id
    wx.navigateTo({ url: `/pages/event/detail/detail?id=${id}` })
  },

  goApply(event) {
    const id = event.detail && event.detail.id ? event.detail.id : 1
    const target = findEventById(this.data.allEvents, id) || findEventById(this.data.events, id)
    if (target && target.isCreatedByMe) {
      wx.showToast({ title: '不能申请自己发布的行程', icon: 'none' })
      return
    }
    wx.navigateTo({ url: `/pages/event/apply/apply?eventId=${id}` })
  },

  goSearch() { wx.navigateTo({ url: '/pages/search/search' }) },
  goCity() { wx.navigateTo({ url: '/pages/city/select/select' }) },
  goNotifications() { wx.navigateTo({ url: '/pages/notifications/notifications' }) }
})
