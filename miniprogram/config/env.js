const runtime = require('./runtime')

function getApiBaseUrl() {
  let extConfig = {}
  try {
    const raw = wx.getExtConfigSync ? wx.getExtConfigSync() : {}
    extConfig = raw.extConfig || raw || {}
  } catch (error) {}

  if (extConfig.apiBaseUrl) return String(extConfig.apiBaseUrl).replace(/\/$/, "")

  let envVersion = "develop"
  try {
    envVersion = wx.getAccountInfoSync().miniProgram.envVersion || "develop"
  } catch (error) {}

  // Trial and release builds must receive an HTTPS address through extConfig;
  // this prevents accidentally shipping a development IP or localhost URL.
	const configured = runtime[envVersion] || ""
	if (envVersion !== "develop" && configured && !/^https:\/\//.test(configured)) return ""
	return String(configured).replace(/\/$/, "")
}

module.exports = { getApiBaseUrl }
