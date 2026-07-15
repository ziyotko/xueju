const { profileCover } = require('../../data/mock')
const api = require('../../utils/api')

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
	level: '未填写水平',
    skiTypeText: '',
    city: '',
	bio: '',
    profileCover,
    reviewCount: 0,
    stats: [
      { value: 0, label: '发起局数' },
      { value: 0, label: '加入局数' },
      { value: '5.0', label: '信用分' },
      { value: '100%', label: '好评率' }
    ],
	certifications: ['微信登录', '站内沟通'],
	resorts: [],
	styles: [],
    reviews: []
  },

  onShow() {
    const tabBar = this.getTabBar && this.getTabBar()
    if (tabBar) tabBar.setData({ active: 4 })
    this.loadProfile()
  },

  async loadProfile() {
    try {
      const profile = await api.me()
	  const levelText = { beginner: '新手', primary: '初级', intermediate: '中级', advanced: '高级' }
      this.setData({
        nickname: profile.nickname || '雪友',
        avatarUrl: profile.avatarUrl || '',
        avatarInitial: (profile.nickname || '雪').slice(0, 1),
		level: levelText[profile.skiLevel] || '未填写水平',
        skiTypeText: profile.skiType === 'ski' ? '双板' : profile.skiType === 'both' ? '单双板' : profile.skiType === 'snowboard' ? '单板' : '',
        city: profile.city || '',
		bio: profile.bio || '',
        styles: profile.styleTags && profile.styleTags.length ? profile.styleTags : this.data.styles,
        resorts: profile.favoriteResorts && profile.favoriteResorts.length ? profile.favoriteResorts : this.data.resorts,
        stats: [
          { value: profile.eventCount || 0, label: '发起局数' },
          { value: profile.joinCount || 0, label: '加入局数' },
          { value: Number(profile.creditScore || 5).toFixed(1), label: '信用分' },
          { value: `${Math.round(profile.goodRate || 100)}%`, label: '好评率' }
        ]
      })
      const reviews = (await api.userReviews(profile.id)).map(mapReview)
      this.setData({ reviews, reviewCount: reviews.length })
    } catch (error) {
      this.setData({
        reviews: [],
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
