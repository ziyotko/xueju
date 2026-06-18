const app = getApp()
const { CONTENT_RISK_CODE, CONTENT_RISK_MESSAGE } = require("../constants/compliance")

function request(options) {
  const token = app.globalData.token
  const header = {
    "content-type": "application/json",
    ...(options.header || {})
  }

  if (token) {
    header.Authorization = `Bearer ${token}`
  }

  return new Promise((resolve, reject) => {
    wx.request({
      url: `${app.globalData.apiBaseUrl}${options.url}`,
      method: options.method || "GET",
      data: options.data || {},
      header,
      success(res) {
        const body = res.data || {}
        if (res.statusCode >= 200 && res.statusCode < 300 && body.code === 0) {
          resolve(body.data)
          return
        }

        const message = body.code === CONTENT_RISK_CODE
          ? CONTENT_RISK_MESSAGE
          : body.message || "请求失败，请稍后再试"
        wx.showToast({ title: message, icon: "none" })
        reject(new Error(message))
      },
      fail(err) {
        wx.showToast({ title: "网络连接失败", icon: "none" })
        reject(err)
      }
    })
  })
}

module.exports = {
  request,
  get: (url, data) => request({ url, data, method: "GET" }),
  post: (url, data) => request({ url, data, method: "POST" })
}
