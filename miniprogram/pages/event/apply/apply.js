const api = require('../../../utils/api')
const { getEventById, addJoinRequest } = require('../../../utils/store')
const { isValidPhone, maskPhone } = require('../../../utils/phone')

function isOwnEvent(event, profile) {
  if (!event) return false
  if (event.isCreatedByMe) return true
  return profile && profile.id && Number(event.creatorId) === Number(profile.id)
}

function leaveApplyPage(eventId) {
  const pages = getCurrentPages()
  setTimeout(() => {
    if (pages.length > 1) {
      wx.navigateBack()
      return
    }
    wx.redirectTo({ url: `/pages/event/detail/detail?id=${eventId}` })
  }, 500)
}

Page({
  data: {
    event: {},
    levels: ['新手', '初级', '中级', '高级'],
    skiTypes: ['单板', '双板', '都可以'],
    levelIndex: 2,
    skiTypeIndex: 0,
    hasCar: false,
    canCarryPeople: false,
    contactPhone: '',
    savePhoneToProfile: false,
    departArea: '北京朝阳',
    message: '单板中级，能连续换刃，想一起刷道互拍。'
  },

  async onLoad(options) {
    const id = Number(options.eventId || 1)
    try {
      const event = await api.event(id)
      let profile = {}
      try { profile = await api.me() } catch (error) {}
      if (isOwnEvent(event, profile)) {
        wx.showToast({ title: '不能申请自己发布的行程', icon: 'none' })
        leaveApplyPage(id)
        return
      }
      this.setData({ event, contactPhone: profile.phone || '' })
    } catch (error) {
      const event = getEventById(id)
      if (isOwnEvent(event, {})) {
        wx.showToast({ title: '不能申请自己发布的行程', icon: 'none' })
        leaveApplyPage(id)
        return
      }
      this.setData({ event })
    }
  },

  onLevelChange(event) { this.setData({ levelIndex: Number(event.detail.value) }) },
  onSkiTypeChange(event) { this.setData({ skiTypeIndex: Number(event.detail.value) }) },
  onInput(event) { this.setData({ [event.currentTarget.dataset.key]: event.detail.value }) },
  onSwitch(event) { this.setData({ [event.currentTarget.dataset.key]: event.detail.value }) },
  toggleSavePhone(event) { this.setData({ savePhoneToProfile: event.detail.checked }) },

  async submitApply() {
    if (isOwnEvent(this.data.event, {})) {
      wx.showToast({ title: '不能申请自己发布的行程', icon: 'none' })
      return
    }
    if (!this.data.departArea.trim()) { wx.showToast({ title: '请填写出发区域', icon: 'none' }); return }
    if (!this.data.contactPhone.trim()) { wx.showToast({ title: '请填写联系手机号', icon: 'none' }); return }
    if (!isValidPhone(this.data.contactPhone)) { wx.showToast({ title: '请输入正确的手机号', icon: 'none' }); return }
    if (!this.data.message.trim()) { wx.showToast({ title: '请填写申请说明', icon: 'none' }); return }
    const payload = {
      skiLevel: ['beginner', 'primary', 'intermediate', 'advanced'][this.data.levelIndex],
      skiType: ['snowboard', 'ski', 'both'][this.data.skiTypeIndex],
      hasCar: this.data.hasCar,
      canCarryPeople: this.data.canCarryPeople,
      departArea: this.data.departArea,
      message: `${this.data.message}\n联系手机号：${maskPhone(this.data.contactPhone)}`
    }
    try {
      await api.applyEvent(this.data.event.id, payload)
      if (this.data.savePhoneToProfile) {
        try {
          const profile = await api.me()
          await api.updateMe({ ...profile, phone: this.data.contactPhone.trim(), genderVisible: true })
        } catch (error) {}
      }
      wx.showToast({ title: '申请已提交', icon: 'success' })
    } catch (error) {
      const result = addJoinRequest({ eventId: this.data.event.id, eventTitle: this.data.event.resort, ...payload })
      if (result.duplicated) {
        wx.showToast({ title: '你已经提交过申请', icon: 'none' })
        return
      }
      wx.showToast({ title: '申请已保存本地', icon: 'none' })
    }
    setTimeout(() => wx.switchTab({ url: '/pages/trips/index/index' }), 600)
  }
})
