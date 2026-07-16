const api = require('../../../utils/api')
const { openPrivacyContract } = require('../../../utils/privacy')

const statusText = {
  unverified: '未完成手机号认证',
  pending: '正在认证',
  verified: '已完成手机号认证',
  reverify_required: '需要重新认证',
  revoked: '认证已撤销'
}

Page({
  data: {
    phone: '',
    code: '',
    agreed: false,
    status: 'unverified',
    statusLabel: '未完成手机号认证',
    phoneMasked: '',
    changeMode: false,
    sending: false,
    submitting: false,
    countdown: 0
  },

  onLoad() {
    this.loadStatus()
  },

  onUnload() {
    if (this.timer) clearInterval(this.timer)
  },

  async loadStatus() {
    try {
      const result = await api.verificationStatus()
      this.setData({
        status: result.verificationStatus || 'unverified',
        statusLabel: statusText[result.verificationStatus] || '未完成手机号认证',
        phoneMasked: result.phoneMasked || ''
      })
    } catch (error) {}
  },

  onPhoneInput(event) { this.setData({ phone: event.detail.value.replace(/\D/g, '').slice(0, 11) }) },
  onCodeInput(event) { this.setData({ code: event.detail.value.replace(/\D/g, '').slice(0, 8) }) },
  onAgreementChange(event) { this.setData({ agreed: (event.detail.value || []).includes('agreed') }) },

  startPhoneChange() {
    wx.showModal({
      title: '更换认证手机号',
      content: '更换成功后，原认证手机号将立即失效。是否继续？',
      confirmText: '继续更换',
      success: (result) => {
        if (result.confirm) this.setData({ changeMode: true, phone: '', code: '', agreed: false })
      }
    })
  },

  cancelPhoneChange() {
    this.setData({ changeMode: false, phone: '', code: '', agreed: false })
  },

  showUserAgreement() {
    wx.showModal({
      title: '用户协议',
      content: '用户应使用本人正常使用的手机号码完成认证，不得冒用他人号码。平台将认证结果用于账号安全、内容管理及依法配合调查。',
      showCancel: false
    })
  },

  async showPrivacy() {
    try { await openPrivacyContract() } catch (error) {}
  },

  async sendCode() {
    if (this.data.sending || this.data.countdown > 0) return
    if (this.data.status === 'verified' && !this.data.changeMode) {
      wx.showToast({ title: '手机号已完成认证', icon: 'none' })
      return
    }
    if (!this.data.agreed) {
      wx.showToast({ title: '请先阅读并同意协议与隐私政策', icon: 'none' })
      return
    }
    if (!/^1[3-9]\d{9}$/.test(this.data.phone)) {
      wx.showToast({ title: '请输入正确的手机号', icon: 'none' })
      return
    }
    this.setData({ sending: true })
    try {
      await api.sendPhoneVerificationCode(this.data.phone, this.data.changeMode)
      this.startCountdown(60)
      wx.showToast({ title: '验证码已发送', icon: 'success' })
    } catch (error) {
      wx.showToast({ title: (error && error.message) || '发送失败', icon: 'none' })
    } finally {
      this.setData({ sending: false })
    }
  },

  startCountdown(seconds) {
    if (this.timer) clearInterval(this.timer)
    this.setData({ countdown: seconds })
    this.timer = setInterval(() => {
      const next = this.data.countdown - 1
      this.setData({ countdown: Math.max(next, 0) })
      if (next <= 0) {
        clearInterval(this.timer)
        this.timer = null
      }
    }, 1000)
  },

  async submit() {
    if (this.data.submitting) return
    if (this.data.status === 'verified' && !this.data.changeMode) {
      wx.showToast({ title: '手机号已完成认证', icon: 'none' })
      return
    }
    if (!this.data.agreed) {
      wx.showToast({ title: '请先阅读并同意协议与隐私政策', icon: 'none' })
      return
    }
    if (!/^1[3-9]\d{9}$/.test(this.data.phone) || !/^\d{4,8}$/.test(this.data.code)) {
      wx.showToast({ title: '请填写正确的手机号和验证码', icon: 'none' })
      return
    }
    this.setData({ submitting: true })
    try {
      await api.checkPhoneVerificationCode(this.data.phone, this.data.code, this.data.changeMode)
      const profile = await api.me()
      const app = getApp()
      app.globalData.user = profile
      wx.setStorageSync('xueju_user', profile)
      wx.showToast({ title: '手机号认证成功', icon: 'success' })
      setTimeout(() => this.returnToPreviousPage(), 500)
    } catch (error) {
      wx.showToast({ title: (error && error.message) || '认证失败', icon: 'none' })
    } finally {
      this.setData({ submitting: false })
    }
  },

  returnToPreviousPage() {
    const pages = getCurrentPages()
    if (pages.length > 1) {
      wx.navigateBack()
      return
    }
    const url = wx.getStorageSync('xueju_verification_return_url') || '/pages/index/index'
    wx.removeStorageSync('xueju_verification_return_url')
    wx.reLaunch({ url })
  }
})
