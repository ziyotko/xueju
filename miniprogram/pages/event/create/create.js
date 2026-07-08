const { CONTENT_RISK_MESSAGE } = require('../../../constants/compliance')
const { unsplashImages } = require('../../../data/mock')
const api = require('../../../utils/api')
const { isValidPhone, maskPhone } = require('../../../utils/phone')

function formatDate(date) {
  const year = date.getFullYear()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  return `${year}-${month}-${day}`
}

function splitDepart(value) {
  const text = (value || '').trim()
  const knownCities = ['北京', '上海', '广州', '深圳', '杭州', '成都', '重庆', '天津', '南京', '武汉', '西安', '张家口', '吉林', '乌鲁木齐']
  const city = knownCities.find((item) => text.startsWith(item)) || text.slice(0, 2) || ''
  return {
    city,
    area: city ? text.replace(city, '').trim() : text
  }
}

function isUploadedImageUrl(url = '') {
  return /^https?:\/\//.test(url) && !/^https?:\/\/tmp\//.test(url)
}

Page({
  data: {
    form: {
      resort: '崇礼 · 万龙滑雪场',
      resortId: 1,
      date: '',
      depart: '北京朝阳',
      departCity: '北京',
      departArea: '朝阳',
      time: '06:30',
      meetPlace: '朝阳大悦城停车场',
      people: 4,
      image: '',
      imageUrl: '',
      note: '',
      noteCount: 0,
      skiTypeReq: 'snowboard',
      levelReq: 'intermediate',
      board: '单板',
      level: '中级',
      style: '刷道',
      trafficType: 'self_drive',
      traffic: '自驾同行',
      canCarpool: true,
      contactType: '站内消息',
      contactContent: '',
      contactPublic: false
    },
    resorts: ['崇礼 · 万龙滑雪场', '南山滑雪场', '云顶滑雪公园', '太舞滑雪小镇'],
    dateStart: '',
    dateEnd: '',
    times: ['06:30', '07:00', '07:20', '08:00'],
    boards: ['单板', '双板', '都可以'],
    levels: ['新手', '初级', '中级', '高级'],
    styles: ['刷道', '练习', '平花', '刻滑', '公园', '拍照', '休闲滑'],
    trafficTypes: ['自驾同行', '公共交通', '高铁同行', '同行交通待定'],
    contactTypes: ['站内消息', '微信号', '手机号'],
    contactTypeIndex: 0,
    toggles: [
      { key: 'allowBeginner', label: '接受新手', checked: false },
      { key: 'sameGenderOnly', label: '仅限同性', checked: false },
      { key: 'canCarpool', label: '同行交通说明', checked: true },
      { key: 'allowRoomShare', label: '住宿需求备注', checked: false },
      { key: 'photo', label: '互拍视频', checked: false }
    ],
    contentRiskMessage: CONTENT_RISK_MESSAGE
  },

  async onLoad() {
    const today = new Date()
    const end = new Date(today)
    end.setFullYear(end.getFullYear() + 2)
    this.setData({
      dateStart: formatDate(today),
      dateEnd: formatDate(end)
    })
    try {
      const resorts = await api.resorts()
      if (resorts.length) {
        this.setData({
          resorts: resorts.map((item) => item.name),
          'form.resort': resorts[0].name,
          'form.resortId': resorts[0].id
        })
      }
    } catch (error) {}
  },

  setPicker(event) {
    const key = event.currentTarget.dataset.key
    const listName = event.currentTarget.dataset.list
    const list = this.data[listName]
    const index = Number(event.detail.value)
    const value = list[index]
    const update = { [`form.${key}`]: value }
    if (key === 'resort') update['form.resortId'] = index + 1
    this.setData(update)
  },
  onDateChange(event) {
    this.setData({ 'form.date': event.detail.value })
  },
  onDepartInput(event) {
    const depart = event.detail.value
    const parsed = splitDepart(depart)
    this.setData({
      'form.depart': depart,
      'form.departCity': parsed.city,
      'form.departArea': parsed.area
    })
  },
  selectPill(event) {
    const key = event.currentTarget.dataset.key
    const value = event.currentTarget.dataset.value
    const update = { [`form.${key}`]: value }
    if (key === 'board') update['form.skiTypeReq'] = value === '双板' ? 'ski' : value === '都可以' ? 'both' : 'snowboard'
    if (key === 'level') update['form.levelReq'] = value === '新手' ? 'beginner' : value === '初级' ? 'primary' : value === '高级' ? 'advanced' : 'intermediate'
    if (key === 'traffic') update['form.trafficType'] = value === '高铁同行' ? 'high_speed_rail' : value === '公共交通' ? 'bus' : value === '同行交通待定' ? 'other' : 'self_drive'
    this.setData(update)
  },
  increasePeople() { this.setData({ 'form.people': Math.min(this.data.form.people + 1, 12) }) },
  decreasePeople() { this.setData({ 'form.people': Math.max(this.data.form.people - 1, 1) }) },
  onPeopleChange(event) {
    this.setData({ 'form.people': event.detail.value })
  },
  onFormInput(event) {
    this.setData({ [`form.${event.currentTarget.dataset.key}`]: event.detail.value })
  },
  chooseTripImage() {
    const choose = wx.chooseMedia
      ? new Promise((resolve, reject) => {
        wx.chooseMedia({ count: 1, mediaType: ['image'], sourceType: ['album', 'camera'], sizeType: ['compressed'], success: resolve, fail: reject })
      })
      : new Promise((resolve, reject) => {
        wx.chooseImage({ count: 1, sizeType: ['compressed'], sourceType: ['album', 'camera'], success: (res) => resolve({ tempFiles: [{ tempFilePath: res.tempFilePaths[0], size: 0 }] }), fail: reject })
      })

    choose.then((res) => {
      const file = res.tempFiles && res.tempFiles[0]
      if (!file || !file.tempFilePath) return
      if (file.size && file.size > 5 * 1024 * 1024) {
        wx.showToast({ title: '图片不能超过 5MB', icon: 'none' })
        return
      }
      this.setData({ 'form.image': file.tempFilePath, 'form.imageUrl': '' })
    }).catch(() => {})
  },
  previewTripImage() {
    if (!this.data.form.image) return
    wx.previewImage({ urls: [this.data.form.image], current: this.data.form.image })
  },
  removeTripImage() {
    this.setData({ 'form.image': '', 'form.imageUrl': '' })
  },
  onToggle(event) {
    const index = Number(event.currentTarget.dataset.index)
    const checked = event.detail.value
    const toggles = this.data.toggles.map((item, current) => current === index ? { ...item, checked } : item)
    const key = toggles[index].key
    this.setData({ toggles, [`form.${key}`]: checked })
  },
  onContactTypeChange(event) {
    const index = Number(event.detail.value)
    this.setData({ contactTypeIndex: index, 'form.contactType': this.data.contactTypes[index] })
  },
  onContactInput(event) {
    this.setData({ 'form.contactContent': event.detail.value })
  },
  onContactPublic(event) {
    this.setData({ 'form.contactPublic': event.detail.value })
  },
  onNoteInput(event) {
    const note = event.detail.value
    this.setData({ 'form.note': note, 'form.noteCount': note.length })
  },
  async submit() {
    const f = this.data.form
    if (!f.resort || !f.date || !f.depart || !f.meetPlace) {
      wx.showToast({ title: '请完善基本信息', icon: 'none' })
      return
    }
    if (f.contactPublic && f.contactType !== '站内消息') {
      if (!f.contactContent.trim()) {
        wx.showToast({ title: '请填写公开联系方式内容', icon: 'none' })
        return
      }
      if (f.contactType === '手机号' && !isValidPhone(f.contactContent)) {
        wx.showToast({ title: '请输入正确的手机号', icon: 'none' })
        return
      }
      const confirmed = await new Promise((resolve) => {
        wx.showModal({
          title: '确认公开联系方式',
          content: '你选择公开展示联系方式，其他用户可以看到该信息。请确认是否继续公开。',
          cancelText: '取消',
          confirmText: '确认公开',
          success: (res) => resolve(res.confirm)
        })
      })
      if (!confirmed) return
    }
    const imageMap = {
      '崇礼 · 万龙滑雪场': unsplashImages.wanlong,
      '南山滑雪场': unsplashImages.nanshan,
      '云顶滑雪公园': unsplashImages.yunding,
      '太舞滑雪小镇': unsplashImages.taiwu
    }
    const contactContent = f.contactType === '手机号' ? maskPhone(f.contactContent) : f.contactContent
    const publicContact = f.contactPublic && f.contactType !== '站内消息'
      ? `\n公开联系方式：${f.contactType} ${contactContent}`
      : ''
    try {
      let imageUrl = f.imageUrl || ''
      if (f.image && isUploadedImageUrl(f.image)) {
        imageUrl = f.image
      } else if (f.image) {
        const uploaded = await api.uploadEventImage(f.image)
        imageUrl = uploaded.url
      }
      const fallbackImage = imageMap[f.resort] || unsplashImages.wanlong
      const submitForm = { ...f, note: `${f.note || ''}${publicContact}`, purposeTags: [f.style], image: imageUrl || fallbackImage, imageUrl: imageUrl || fallbackImage }
      const event = await api.createEvent(submitForm)
      wx.showToast({ title: '发布成功', icon: 'success' })
      setTimeout(() => wx.navigateTo({ url: `/pages/event/detail/detail?id=${event.id}` }), 500)
    } catch (error) {
      wx.showToast({ title: '发布失败，请稍后重试', icon: 'none' })
    }
  }
})
