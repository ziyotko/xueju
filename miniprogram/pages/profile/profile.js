const { requirePrivacyConsent } = require('../../utils/privacy')
const { profileCover } = require('../../data/mock')
const { getProfile, getReviews } = require('../../utils/store')
const api = require('../../utils/api')
const { maskPhone } = require('../../utils/phone')

Page({
  data: {
    nickname: '大力',
    avatarUrl: '',
    avatarInitial: '大',
    level: '中级',
    bio: '热爱滑雪，周末不是在雪场就是在去雪场的路上。',
    phoneText: '未填写',
    phoneHint: '用于活动报名、预约确认和必要联系',
    profileCover,
    privacyReady: false,
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
      const localProfile = getProfile()
      this.setData({
        nickname: profile.nickname || '雪友',
        avatarUrl: localProfile.avatarLocalPath || profile.avatarUrl || localProfile.avatarUrl || '',
        avatarInitial: (profile.nickname || '雪').slice(0, 1),
        level: profile.skiLevel || '中级',
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
      const reviews = await api.userReviews(profile.id)
      this.setData({ reviews: reviews.map((item) => ({ name: item.reviewerName, initial: item.reviewerName.slice(0, 1), date: (item.createdAt || '').slice(0, 10), credit: `评分 ${item.score}.0`, text: item.content, tags: item.positiveTags || [] })) })
    } catch (error) {
      const profile = getProfile()
      const storedReviews = getReviews().map((item) => ({
        name: item.anonymous ? '匿名雪友' : '我',
        initial: item.anonymous ? '匿' : '我',
        date: (item.createdAt || '').slice(0, 10) || '刚刚',
        credit: `评分 ${item.score}.0`,
        text: item.content,
        tags: item.positive && item.positive.length ? item.positive : ['沟通顺畅']
      }))
      this.setData({
        nickname: profile.nickname,
        avatarUrl: profile.avatarLocalPath || profile.avatarUrl || '',
        avatarInitial: (profile.nickname || '雪').slice(0, 1),
        level: profile.level,
        bio: profile.bio,
        phoneText: maskPhone(profile.phone) || '未填写',
        phoneHint: profile.phone ? '手机号仅自己可见' : '用于活动报名、预约确认和必要联系',
        styles: profile.styles || this.data.styles,
        reviews: storedReviews
      })
    }
  },

  onAvatarError() {
    const localProfile = getProfile()
    if (localProfile.avatarLocalPath && this.data.avatarUrl !== localProfile.avatarLocalPath) {
      this.setData({ avatarUrl: localProfile.avatarLocalPath })
      return
    }
    this.setData({ avatarUrl: '' })
    wx.showToast({ title: '头像加载失败，请检查网络', icon: 'none' })
  },

  goEditProfile() { wx.navigateTo({ url: '/pages/profile/edit/edit' }) },
  goSettings() { wx.navigateTo({ url: '/pages/settings/settings' }) },
  goFavorites() { wx.navigateTo({ url: '/pages/favorites/favorites' }) },
  goNotifications() { wx.navigateTo({ url: '/pages/notifications/notifications' }) },
  showAllReviews() { wx.showToast({ title: '已展示全部评价', icon: 'none' }) }
})
