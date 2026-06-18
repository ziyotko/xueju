function requirePrivacyConsent() {
  if (typeof wx.requirePrivacyAuthorize !== "function") {
    return Promise.resolve()
  }

  return new Promise((resolve, reject) => {
    wx.requirePrivacyAuthorize({
      success: resolve,
      fail: reject
    })
  })
}

module.exports = {
  requirePrivacyConsent
}

