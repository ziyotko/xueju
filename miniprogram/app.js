const { login } = require('./utils/request')
const { getApiBaseUrl } = require('./config/env')

App({
  globalData: {
	apiBaseUrl: getApiBaseUrl(),
    token: "",
    user: null
  },

  onLaunch() {
    const token = wx.getStorageSync('xueju_token')
    const user = wx.getStorageSync('xueju_user')
    if (token) this.globalData.token = token
    if (user) this.globalData.user = user
    if (!token) this.loginSilently()
  },

  loginSilently() {
	login().then((data) => {
	  const user = data.user || {}
	  if (wx.getStorageSync('xueju_onboarding_prompted') || (user.nickname && user.nickname !== '雪友' && user.skiLevel)) return
	  wx.setStorageSync('xueju_onboarding_prompted', true)
	  setTimeout(() => {
		wx.showModal({
		  title: '完善滑雪资料',
		  content: '填写昵称、城市和滑雪水平后，其他雪友能更准确地审核你的申请。',
		  confirmText: '去完善',
		  success: (result) => { if (result.confirm) wx.navigateTo({ url: '/pages/profile/edit/edit' }) }
		})
	  }, 800)
	}).catch((error) => {
      console.warn('silent login failed, browse as guest', error)
    })
  }
})
