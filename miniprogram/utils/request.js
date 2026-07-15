const { CONTENT_RISK_CODE, CONTENT_RISK_MESSAGE } = require("../constants/compliance")

const PUBLIC_PATHS = ["/health", "/auth/wechat-login", "/events", "/dict/resorts", "/dict/tags", "/dict/cities"]
let loginTask = null

function isPublicRequest(url = "") {
  if (url === "/events" || url.startsWith("/events?") || /^\/events\/\d+$/.test(url)) return true
  return PUBLIC_PATHS.some((path) => url === path || url.startsWith(`${path}?`))
}

function login() {
  const app = getApp()
  if (loginTask) return loginTask
  loginTask = new Promise((resolve, reject) => {
    wx.login({
      success(res) {
        request({
          url: "/auth/wechat-login",
          method: "POST",
          data: { code: res.code || "dev-local" },
          skipAuth: true
        }).then((data) => {
          app.globalData.token = data.token || ""
          app.globalData.user = data.user || null
          wx.setStorageSync("xueju_token", data.token || "")
          wx.setStorageSync("xueju_user", data.user || null)
          resolve(data)
        }).catch(reject)
      },
      fail: reject
    })
  }).finally(() => {
    loginTask = null
  })
  return loginTask
}

function ensureLogin(options = {}) {
  const app = getApp()
  if (options.skipAuth || isPublicRequest(options.url)) return Promise.resolve()
  if (app.globalData.token) return Promise.resolve()

  const token = wx.getStorageSync("xueju_token")
  const user = wx.getStorageSync("xueju_user")
  if (token) {
    app.globalData.token = token
    if (user) app.globalData.user = user
    return Promise.resolve()
  }

  return login()
}

function clearAuth() {
  const app = getApp()
  app.globalData.token = ""
  app.globalData.user = null
  wx.removeStorageSync("xueju_token")
  wx.removeStorageSync("xueju_user")
}

function buildHeader(options = {}) {
  const app = getApp()
  const header = {
    "content-type": "application/json",
    ...(options.header || {})
  }
  if (app.globalData.token) {
    header.Authorization = `Bearer ${app.globalData.token}`
  }
  return header
}

function request(options) {
  const app = getApp()
	if (!app.globalData.apiBaseUrl) {
	  const error = new Error("当前版本未配置 HTTPS 服务地址")
	  wx.showToast({ title: error.message, icon: "none" })
	  return Promise.reject(error)
	}
  return ensureLogin(options).then(() => new Promise((resolve, reject) => {
    wx.request({
      url: `${app.globalData.apiBaseUrl}${options.url}`,
      method: options.method || "GET",
      data: options.data || {},
      header: buildHeader(options),
      timeout: options.timeout || 20000,
      success(res) {
        const body = res.data || {}
        if (res.statusCode >= 200 && res.statusCode < 300 && body.code === 0) {
          resolve(body.data)
          return
        }

        if (res.statusCode === 401 && !options.skipAuth && !options._retried) {
          clearAuth()
          login()
            .then(() => request({ ...options, _retried: true }))
            .then(resolve)
            .catch(reject)
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
  }))
}

function upload(url, filePath, name = "file", formData = {}, retried = false) {
  const app = getApp()
  return ensureLogin({ url }).then(() => new Promise((resolve, reject) => {
    wx.uploadFile({
      url: `${app.globalData.apiBaseUrl}${url}`,
      filePath,
      name,
      formData,
      header: buildHeader({}),
      timeout: 30000,
      success(res) {
        let body = {}
        try {
          body = JSON.parse(res.data || "{}")
        } catch (error) {}
        if (res.statusCode >= 200 && res.statusCode < 300 && body.code === 0) {
          resolve(body.data)
          return
        }

        if (res.statusCode === 401 && !retried) {
          clearAuth()
          login()
            .then(() => upload(url, filePath, name, formData, true))
            .then(resolve)
            .catch(reject)
          return
        }

        const message = body.message || "图片上传失败，请稍后再试"
        wx.showToast({ title: message, icon: "none" })
        reject(new Error(message))
      },
      fail(err) {
        wx.showToast({ title: "图片上传失败，请检查网络", icon: "none" })
        reject(err)
      }
    })
  }))
}

module.exports = {
  request,
  get: (url, data) => request({ url, data, method: "GET" }),
  post: (url, data) => request({ url, data, method: "POST" }),
  put: (url, data) => request({ url, data, method: "PUT" }),
  delete: (url, data) => request({ url, data, method: "DELETE" }),
  upload,
  login,
  ensureLogin
}
