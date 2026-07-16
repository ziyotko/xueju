const api = require('../../../utils/api')
const { requirePrivacyConsent } = require('../../../utils/privacy')

const styleOptions = ["刷道", "刻滑", "节奏稳", "爱拍照", "不赶时间", "公园", "平花"]

function buildTags(list, selected) {
  return list.map((text) => ({ text, selected: selected.includes(text) }))
}

function isUploadedImageUrl(url = "") {
  return /^https?:\/\//.test(url) && !/^https?:\/\/tmp\//.test(url)
}

Page({
  data: {
    nickname: "大力",
    avatarInitial: "大",
    avatarUrl: "",
    savedAvatarUrl: "",
    avatarLocalPath: "",
    avatarChanged: false,
    city: "北京",
    bio: "热爱滑雪，周末不是在雪场就是在去雪场的路上。",
    levels: ["新手", "初级", "中级", "高级"],
    levelIndex: 2,
    skiTypes: ["单板", "双板", "都可以"],
    skiTypeIndex: 0,
    tags: buildTags(styleOptions, ["刷道", "节奏稳", "爱拍照"])
  },

  async onLoad() {
    try {
      const profile = await api.me()
      const levelMap = { beginner: "新手", primary: "初级", intermediate: "中级", advanced: "高级" }
      const typeMap = { snowboard: "单板", ski: "双板", both: "都可以" }
      const level = levelMap[profile.skiLevel] || "中级"
      const skiType = typeMap[profile.skiType] || "单板"
      this.setData({
        nickname: profile.nickname || "雪友",
        avatarInitial: (profile.nickname || "雪").slice(0, 1),
        avatarUrl: profile.avatarUrl || "",
        savedAvatarUrl: profile.avatarUrl || "",
        avatarLocalPath: "",
        avatarChanged: false,
        city: profile.city || "北京",
		bio: profile.bio || "",
        levelIndex: Math.max(this.data.levels.indexOf(level), 0),
        skiTypeIndex: Math.max(this.data.skiTypes.indexOf(skiType), 0),
        tags: buildTags(styleOptions, profile.styleTags || [])
      })
    } catch (error) {
      wx.showToast({ title: "资料加载失败，请稍后重试", icon: "none" })
    }
	const pendingUploadId = Number(wx.getStorageSync("xueju_pending_avatar_upload") || 0)
	if (pendingUploadId) {
	  try {
		const upload = await api.uploadStatus(pendingUploadId)
		if (upload.status === "approved" && upload.url) {
		  this.setData({ avatarUrl: upload.url, avatarChanged: true })
		  wx.removeStorageSync("xueju_pending_avatar_upload")
		} else if (upload.status === "rejected") {
		  wx.removeStorageSync("xueju_pending_avatar_upload")
		  wx.showToast({ title: upload.result || "头像未通过审核", icon: "none" })
		}
	  } catch (error) {}
	}
  },

  onInput(event) {
    const key = event.currentTarget.dataset.key
    const next = { [key]: event.detail.value }
    if (key === "nickname") next.avatarInitial = (event.detail.value || "雪").slice(0, 1)
    this.setData(next)
  },
  onLevelChange(event) { this.setData({ levelIndex: Number(event.detail.value) }) },
  onSkiTypeChange(event) { this.setData({ skiTypeIndex: Number(event.detail.value) }) },
  toggleTag(event) {
    const index = Number(event.currentTarget.dataset.index)
    this.setData({ tags: this.data.tags.map((item, currentIndex) => currentIndex === index ? { ...item, selected: !item.selected } : item) })
  },
	async chooseAvatar() {
	try { await requirePrivacyConsent() } catch (error) { return }
    const choose = wx.chooseMedia
      ? new Promise((resolve, reject) => {
        wx.chooseMedia({ count: 1, mediaType: ["image"], sourceType: ["album", "camera"], sizeType: ["compressed"], success: resolve, fail: reject })
      })
      : new Promise((resolve, reject) => {
        wx.chooseImage({ count: 1, sizeType: ["compressed"], sourceType: ["album", "camera"], success: (res) => resolve({ tempFiles: [{ tempFilePath: res.tempFilePaths[0], size: 0 }] }), fail: reject })
      })

    choose.then((res) => {
      const file = res.tempFiles && res.tempFiles[0]
      if (!file || !file.tempFilePath) return
      if (file.size && file.size > 5 * 1024 * 1024) {
        wx.showToast({ title: "头像不能超过 5MB", icon: "none" })
        return
      }
      this.setData({ avatarUrl: file.tempFilePath, avatarLocalPath: file.tempFilePath, avatarChanged: true })
    }).catch(() => {})
  },
  onAvatarError() {
    if (!this.data.avatarChanged) this.setData({ avatarUrl: "" })
  },
  async saveProfile() {
	try { await requirePrivacyConsent() } catch (error) { return }
    const styleTags = this.data.tags.filter((item) => item.selected).map((item) => item.text)
    const skiLevel = ["beginner", "primary", "intermediate", "advanced"][this.data.levelIndex]
    const skiType = ["snowboard", "ski", "both"][this.data.skiTypeIndex]
    try {
      let avatarUrl = this.data.avatarChanged ? this.data.avatarUrl : this.data.savedAvatarUrl
      if (this.data.avatarChanged && avatarUrl && !isUploadedImageUrl(avatarUrl)) {
		let uploaded = await api.uploadAvatar(avatarUrl)
		if (!uploaded.url && uploaded.id) uploaded = await api.waitForUpload(uploaded)
		if (!uploaded.url) {
		  if (uploaded.id) wx.setStorageSync("xueju_pending_avatar_upload", uploaded.id)
		  wx.showToast({ title: "头像审核中，请审核通过后重试", icon: "none" })
		  return
		}
        avatarUrl = uploaded.url
      }
	  await api.updateMe({ nickname: this.data.nickname, avatarUrl, bio: this.data.bio.trim(), city: this.data.city, skiLevel, skiType, styleTags })
      wx.showToast({ title: "资料已保存", icon: "success" })
      setTimeout(() => wx.navigateBack(), 600)
    } catch (error) {
    }
  }
})
