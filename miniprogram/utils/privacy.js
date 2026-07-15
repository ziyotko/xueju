function requirePrivacyConsent() {
  if (typeof wx.requirePrivacyAuthorize !== "function") {
    return Promise.resolve()
  }

	return new Promise((resolve, reject) => {
    wx.requirePrivacyAuthorize({
      success: resolve,
	  fail(error) {
		wx.showModal({
		  title: "需要隐私授权",
		  content: "编辑资料或选择图片前，需要先阅读并同意隐私保护指引。你可以查看指引后重新操作。",
		  confirmText: "查看指引",
		  success(result) {
			if (result.confirm && typeof wx.openPrivacyContract === "function") {
			  wx.openPrivacyContract({})
			}
		  },
		  complete() { reject(error) }
		})
	  }
    })
  })
}

function openPrivacyContract() {
  if (typeof wx.openPrivacyContract === "function") {
    return new Promise((resolve, reject) => {
      wx.openPrivacyContract({ success: resolve, fail: reject })
    })
  }
  wx.showModal({
    title: "隐私保护指引",
    content: "雪局仅收集提供行程组局所需的昵称、头像、滑雪资料及用户主动发布的内容；不收集或公开手机号、微信号和二维码。",
    showCancel: false
  })
  return Promise.resolve()
}

module.exports = {
  requirePrivacyConsent,
  openPrivacyContract
}
