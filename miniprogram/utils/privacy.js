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
    content: "雪局会收集用户主动填写的手机号，用于手机号认证、账号安全、内容管理和依法配合调查。验证码由阿里云号码认证服务发送并核验，手机号在服务端加密保存，仅展示脱敏号码，不向其他用户公开。用户可申请更正、删除或注销账号；数据按实现认证、安全审计及法定义务所需的最短期限保存。",
    showCancel: false
  })
  return Promise.resolve()
}

module.exports = {
  requirePrivacyConsent,
  openPrivacyContract
}
