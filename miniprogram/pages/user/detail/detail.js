const api = require('../../../utils/api')

const skiTypeText = { snowboard: '单板', ski: '双板', both: '单双板' }
const levelText = { beginner: '新手', primary: '初级', intermediate: '中级', advanced: '高级' }

function displayName(user) {
  return user.nickname || user.name || '雪友'
}

function displayLevel(user) {
  if (user.level) return user.level
  return [skiTypeText[user.skiType] || user.skiType, levelText[user.skiLevel] || user.skiLevel].filter(Boolean).join(' · ') || '滑雪资料待完善'
}

function currentUserId() {
  const app = getApp()
  const user = (app.globalData && app.globalData.user) || wx.getStorageSync('xueju_user') || {}
  return Number(user.id || 0)
}

Page({
  data: {
    userId: 0,
    isSelf: false,
    name: '雪友',
    initial: '雪',
    level: '滑雪资料待完善',
    credit: '',
    tags: [],
    reviews: [],
    reviewText: '暂无同滑评价'
  },

  async onLoad(options) {
    const userId = Number(options.id || 1)
    let user = {}
    try { user = await api.publicUser(userId) } catch (error) {}
    let myId = currentUserId()
    if (!myId) {
      try {
        const me = await api.me()
        myId = Number(me.id || 0)
      } catch (error) {}
    }
    const name = displayName(user)

    this.setData({
      userId,
      isSelf: !!myId && Number(userId) === myId,
      name,
      initial: user.initial || name.slice(0, 1),
      level: displayLevel(user),
      credit: user.credit || (user.creditScore ? `信用 ${Number(user.creditScore).toFixed(1)}` : ''),
      tags: user.styleTags || []
    })

    try {
      const reviews = await api.userReviews(userId)
      this.setData({
        reviews: reviews || [],
        reviewText: reviews && reviews.length ? reviews[0].content : '暂无同滑评价'
      })
    } catch (error) {}
  },

  async follow() {
    if (this.data.isSelf) {
      wx.showToast({ title: '不能关注自己', icon: 'none' })
      return
    }
    try {
      await api.followUser(this.data.userId, true)
      wx.showToast({ title: '已关注', icon: 'success' })
    } catch (error) {}
  },

  report() {
    wx.navigateTo({ url: `/pages/report/report?targetType=user&targetId=${this.data.userId || 1}` })
  }
})
