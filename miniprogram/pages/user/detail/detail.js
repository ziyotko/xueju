const api = require('../../../utils/api')
const { members } = require('../../../data/mock')

const skiTypeText = { snowboard: '单板', ski: '双板', both: '单双板' }
const levelText = { beginner: '新手', primary: '初级', intermediate: '中级', advanced: '高级' }

function displayName(user) {
  return user.nickname || user.name || '雪友'
}

function displayLevel(user) {
  if (user.level) return user.level
  return [skiTypeText[user.skiType] || user.skiType, levelText[user.skiLevel] || user.skiLevel].filter(Boolean).join(' · ') || '滑雪资料待完善'
}

Page({
  data: {
    userId: 0,
    name: '雪友',
    initial: '雪',
    level: '滑雪资料待完善',
    credit: '信用 5.0',
    tags: ['准时', '友好', '水平真实'],
    reviews: [],
    reviewText: '暂无同滑评价'
  },

  async onLoad(options) {
    const userId = Number(options.id || 1)
    const cached = wx.getStorageSync(`xueju_user_detail_${userId}`) || {}
    const mock = members.find((item) => Number(item.id) === userId) || {}
    const user = { ...mock, ...cached }
    const name = displayName(user)

    this.setData({
      userId,
      name,
      initial: user.initial || name.slice(0, 1),
      level: displayLevel(user),
      credit: user.credit || `信用 ${Number(user.creditScore || 5).toFixed(1)}`,
      tags: user.tags || ['准时', '友好', '水平真实']
    })

    try {
      const reviews = await api.userReviews(userId)
      this.setData({
        reviews: reviews || [],
        reviewText: reviews && reviews.length ? reviews[0].content : '暂无同滑评价'
      })
    } catch (error) {}
  },

  follow() {
    wx.showToast({ title: '已关注', icon: 'success' })
  },

  report() {
    wx.navigateTo({ url: `/pages/report/report?targetType=user&targetId=${this.data.userId || 1}` })
  }
})
