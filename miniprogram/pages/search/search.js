const api = require('../../utils/api')

function findEventById(events, id) {
  return (events || []).find((item) => Number(item.id) === Number(id))
}

Page({
  data: {
    keyword: '',
    events: [],
    allEvents: [],
    loading: false,
    hotWords: ['万龙', '南山', '周六', '自驾', '新手友好', '单板']
  },

  onLoad() {
    this.loadEvents()
  },

  onInput(event) {
    this.setData({ keyword: event.detail.value })
  },

  useHot(event) {
    this.setData({ keyword: event.currentTarget.dataset.keyword }, () => this.doSearch())
  },

  async loadEvents(keyword = '') {
    this.setData({ loading: true })
    try {
      const page = await api.events({ page: 1, pageSize: 50, keyword })
      this.setData({ allEvents: page.list || [], events: page.list || [] })
    } catch (error) {
      this.setData({ allEvents: [], events: [] })
    } finally {
      this.setData({ loading: false })
    }
  },

  doSearch() {
    this.loadEvents(this.data.keyword.trim())
  },

  goDetail(event) {
    wx.navigateTo({ url: `/pages/event/detail/detail?id=${event.detail.id}` })
  },

  goApply(event) {
    const id = event.detail.id
    const target = findEventById(this.data.allEvents, id) || findEventById(this.data.events, id)
    if (target && target.isCreatedByMe) {
      wx.showToast({ title: '不能申请自己发布的行程', icon: 'none' })
      return
    }
    wx.navigateTo({ url: `/pages/event/apply/apply?eventId=${id}` })
  }
})
