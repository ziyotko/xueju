const { getEvents } = require('../../utils/store')

function findEventById(events, id) {
  return (events || []).find((item) => Number(item.id) === Number(id))
}

Page({
  data: { keyword: '', events: [], allEvents: [], hotWords: ['万龙', '南山', '周六', '自驾', '新手友好', '单板'] },
  onLoad() {
    const allEvents = getEvents()
    this.setData({ allEvents, events: allEvents })
  },
  onInput(e) {
    this.setData({ keyword: e.detail.value }, () => this.doSearch())
  },
  useHot(e) {
    this.setData({ keyword: e.currentTarget.dataset.keyword }, () => this.doSearch())
  },
  doSearch() {
    const keyword = this.data.keyword.trim()
    if (!keyword) { this.setData({ events: this.data.allEvents }); return }
    const events = this.data.allEvents.filter((item) => JSON.stringify(item).includes(keyword))
    this.setData({ events })
  },
  goDetail(e) { wx.navigateTo({ url: `/pages/event/detail/detail?id=${e.detail.id}` }) },
  goApply(e) {
    const id = e.detail.id
    const target = findEventById(this.data.allEvents, id) || findEventById(this.data.events, id)
    if (target && target.isCreatedByMe) {
      wx.showToast({ title: '不能申请自己发布的行程', icon: 'none' })
      return
    }
    wx.navigateTo({ url: `/pages/event/apply/apply?eventId=${id}` })
  }
})
