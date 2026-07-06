const { login } = require('./utils/request')

App({
  globalData: {
    apiBaseUrl: "http://10.1.100.82:8080/api",
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
    login().catch((error) => {
      console.warn('silent login failed, browse as guest', error)
    })
  }
})
