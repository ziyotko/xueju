const { requirePrivacyConsent } = require('../../utils/privacy')
const { profileCover } = require('../../data/mock')
const api = require('../../utils/api')
const { maskPhone } = require('../../utils/phone')

function mapReview(item) {
  const name = item.reviewerName || item.name || (item.anonymous ? '匿名雪友' : '雪友')
  return {
    id: item.id,
    name,
    initial: name.slice(0, 1),
    date: (item.createdAt || '').slice(0, 10) || item.date || '刚刚',
    credit: item.credit || `评分 ${item.score || 5}.0`,
    text: item.content || item.text || '',
    tags: item.positiveTags || item.tags || []
  }
}

Page({
  data: {
    nickname: '雪友',
    avatarUrl: '',
    avatarInitial: '雪',
    level: '中级',
    skiTypeText: '',
    city: '',
    bio: '热爱滑雪，周末不是在雪场就是在去雪场的路上。',
    phoneText: '未填写',
    phoneHint: '用于活动报名、预约确认和必要联系',
    profileCover,
    privacyReady: false,
    reviewCount: 0,
    stats: [
      { value: 0, label: '发起局数' },
      { value: 0, label: '加入局数' },
      { value: 0, label: '相册' },
      { value: '100%', label: '好评率' }
    ],
    certifications: ['手机认证', '实名认证', '滑雪水平认证'],
    resorts: ['崇礼万龙', '崇礼云顶', '南山滑雪场'],
    styles: ['刷道', '节奏稳', '爱拍照'],
    reviews: []
  },

  onLoad() {
    requirePrivacyConsent()
      .then(() => this.setData({ privacyReady: true }))
      .catch(() => wx.showToast({ title: '请先同意隐私保护指引', icon: 'none' }))
  },

  onShow() {
    const tabBar = this.getTabBar && this.getTabBar()
    if (tabBar) tabBar.setData({ active: 4 })
    this.loadProfile()
  },

  async loadProfile() {
    try {
      const profile = await api.me()
      this.setData({
        nickname: profile.nickname || '雪友',
        avatarUrl: profile.avatarUrl || '',
        avatarInitial: (profile.nickname || '雪').slice(0, 1),
        level: profile.skiLevel || '中级',
        skiTypeText: profile.skiType === 'ski' ? '双板' : profile.skiType === 'both' ? '单双板' : profile.skiType === 'snowboard' ? '单板' : '',
        city: profile.city || '',
        phoneText: maskPhone(profile.phone) || '未填写',
        phoneHint: profile.phone ? '手机号仅自己可见' : '用于活动报名、预约确认和必要联系',
        styles: profile.styleTags && profile.styleTags.length ? profile.styleTags : this.data.styles,
        resorts: profile.favoriteResorts && profile.favoriteResorts.length ? profile.favoriteResorts : this.data.resorts,
        stats: [
          { value: profile.eventCount || 0, label: '发起局数' },
          { value: profile.joinCount || 0, label: '加入局数' },
          { value: 0, label: '相册' },
          { value: `${Math.round(profile.goodRate || 100)}%`, label: '好评率' }
        ]
      })
      const reviews = (await api.userReviews(profile.id)).map(mapReview)
      this.setData({ reviews, reviewCount: reviews.length })
    } catch (error) {
      this.setData({
        reviews: [],
        phoneText: '未填写',
        reviewCount: 0
      })
    }
  },

  onAvatarError() {
    this.setData({ avatarUrl: '' })
    wx.showToast({ title: '头像加载失败，请检查网络', icon: 'none' })
  },

  goEditProfile() { wx.navigateTo({ url: '/pages/profile/edit/edit' }) },
  goSettings() { wx.navigateTo({ url: '/pages/settings/settings' }) },
  goFavorites() { wx.navigateTo({ url: '/pages/favorites/favorites' }) },
  goNotifications() { wx.navigateTo({ url: '/pages/notifications/notifications' }) },
  showAllReviews() { wx.showToast({ title: this.data.reviewCount ? '已展示全部评价' : '暂无收到的评价', icon: 'none' }) }
})
