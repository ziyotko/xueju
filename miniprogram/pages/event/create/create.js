const { CONTENT_RISK_MESSAGE } = require('../../../constants/compliance')
const { unsplashImages } = require('../../../data/mock')
const api = require('../../../utils/api')
const { requirePrivacyConsent } = require('../../../utils/privacy')

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
    eventId: 0,
    editMode: false,
    submitting: false,
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
      canCarpool: true
    },
    resorts: ['崇礼 · 万龙滑雪场', '南山滑雪场', '云顶滑雪公园', '太舞滑雪小镇'],
    resortOptions: [
      { id: 1, name: '崇礼 · 万龙滑雪场' },
      { id: 2, name: '南山滑雪场' },
      { id: 3, name: '云顶滑雪公园' },
      { id: 4, name: '太舞滑雪小镇' }
    ],
    dateStart: '',
    dateEnd: '',
    times: ['06:30', '07:00', '07:20', '08:00'],
    boards: ['单板', '双板', '都可以'],
    levels: ['新手', '初级', '中级', '高级'],
    styles: ['刷道', '练习', '平花', '刻滑', '公园', '拍照', '休闲滑'],
    trafficTypes: ['自驾同行', '公共交通', '高铁同行', '同行交通待定'],
    toggles: [
      { key: 'allowBeginner', label: '接受新手', checked: false },
      { key: 'sameGenderOnly', label: '仅限同性', checked: false },
      { key: 'canCarpool', label: '同行交通说明', checked: true },
      { key: 'allowRoomShare', label: '住宿需求备注', checked: false },
      { key: 'photo', label: '互拍视频', checked: false }
    ],
    contentRiskMessage: CONTENT_RISK_MESSAGE
  },

  async onLoad(options = {}) {
	const eventId = Number(options.id || 0)
	this.setData({ eventId, editMode: eventId > 0 })
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
          resortOptions: resorts,
          'form.resort': resorts[0].name,
          'form.resortId': resorts[0].id
        })
      }
    } catch (error) {}
	if (eventId) {
	  try {
		const event = await api.event(eventId)
		const skiTypeText = { snowboard: '单板', ski: '双板', both: '都可以' }
		const levelText = { beginner: '新手', primary: '初级', intermediate: '中级', advanced: '高级' }
		const trafficText = { self_drive: '自驾同行', high_speed_rail: '高铁同行', bus: '公共交通', other: '同行交通待定' }
		const form = {
		  ...this.data.form,
		  resort: event.resortName || event.resort,
		  resortId: Number(event.resortId || 0),
		  date: event.eventDate || event.date,
		  depart: `${event.departCity || ''}${event.departArea || ''}`,
		  departCity: event.departCity || '',
		  departArea: event.departArea || '',
		  time: event.startTime ? event.startTime.slice(11, 16) : event.time,
		  meetPlace: event.meetPlace || '',
		  people: Number(event.maxMembers || event.maxPeople || 1),
		  image: event.imageUrl || '',
		  imageUrl: event.imageUrl || '',
		  note: event.remark || event.note || '',
		  noteCount: (event.remark || event.note || '').length,
		  skiTypeReq: event.skiTypeReq || 'snowboard',
		  levelReq: event.levelReq || 'intermediate',
		  board: skiTypeText[event.skiTypeReq] || '单板',
		  level: levelText[event.levelReq] || '中级',
		  style: (event.purposeTags || [])[0] || '刷道',
		  trafficType: event.trafficType || 'self_drive',
		  traffic: trafficText[event.trafficType] || '自驾同行',
		  allowBeginner: !!event.allowBeginner,
		  sameGenderOnly: !!event.sameGenderOnly,
		  canCarpool: !!event.allowCarPool,
		  allowRoomShare: !!event.allowRoomShare,
		  photo: !!event.allowPhoto
		}
		this.setData({
		  form,
		  toggles: this.data.toggles.map((item) => ({ ...item, checked: !!form[item.key] }))
		})
	  } catch (error) {
		wx.showToast({ title: '行程加载失败', icon: 'none' })
	  }
	}
	if (!eventId) {
	  const pendingUploadId = Number(wx.getStorageSync('xueju_pending_event_upload') || 0)
	  if (pendingUploadId) {
		try {
		  const upload = await api.uploadStatus(pendingUploadId)
		  if (upload.status === 'approved' && upload.url) {
			this.setData({ 'form.image': upload.url, 'form.imageUrl': upload.url })
			wx.removeStorageSync('xueju_pending_event_upload')
		  } else if (upload.status === 'rejected') {
			wx.removeStorageSync('xueju_pending_event_upload')
			wx.showToast({ title: upload.result || '图片未通过审核', icon: 'none' })
		  }
		} catch (error) {}
	  }
	}
  },

  setPicker(event) {
    const key = event.currentTarget.dataset.key
    const listName = event.currentTarget.dataset.list
    const list = this.data[listName]
    const index = Number(event.detail.value)
    const value = list[index]
    const update = { [`form.${key}`]: value }
    if (key === 'resort') update['form.resortId'] = Number((this.data.resortOptions[index] || {}).id || 0)
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
  increasePeople() { this.setData({ 'form.people': Math.min(this.data.form.people + 1, 20) }) },
  decreasePeople() { this.setData({ 'form.people': Math.max(this.data.form.people - 1, 1) }) },
  onPeopleChange(event) {
    this.setData({ 'form.people': event.detail.value })
  },
  onFormInput(event) {
    this.setData({ [`form.${event.currentTarget.dataset.key}`]: event.detail.value })
  },
  async chooseTripImage() {
    try { await requirePrivacyConsent() } catch (error) { return }
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
  onNoteInput(event) {
    const note = event.detail.value
    this.setData({ 'form.note': note, 'form.noteCount': note.length })
  },
  async submit() {
	if (this.data.submitting) return
    const f = this.data.form
    if (!f.resort || !f.date || !f.depart || !f.meetPlace) {
      wx.showToast({ title: '请完善基本信息', icon: 'none' })
      return
    }
    const imageMap = {
      '崇礼 · 万龙滑雪场': unsplashImages.wanlong,
      '南山滑雪场': unsplashImages.nanshan,
      '云顶滑雪公园': unsplashImages.yunding,
      '太舞滑雪小镇': unsplashImages.taiwu
    }
    try {
	  this.setData({ submitting: true })
      let imageUrl = f.imageUrl || ''
      if (f.image && isUploadedImageUrl(f.image)) {
        imageUrl = f.image
      } else if (f.image) {
		let uploaded = await api.uploadEventImage(f.image)
		if (!uploaded.url && uploaded.id) uploaded = await api.waitForUpload(uploaded)
        if (!uploaded.url) {
		  if (uploaded.id) wx.setStorageSync('xueju_pending_event_upload', uploaded.id)
          wx.showToast({ title: '图片审核中，可移除图片后先发布', icon: 'none' })
          return
        }
        imageUrl = uploaded.url
      }
      const fallbackImage = imageMap[f.resort] || unsplashImages.wanlong
      const submitForm = { ...f, note: f.note || '', purposeTags: [f.style], image: imageUrl || fallbackImage, imageUrl: imageUrl || fallbackImage }
	  const event = this.data.editMode
		? await api.updateEvent(this.data.eventId, submitForm)
		: await api.createEvent(submitForm)
	  wx.showToast({ title: this.data.editMode ? '修改已保存' : '发布成功', icon: 'success' })
	  setTimeout(() => wx.redirectTo({ url: `/pages/event/detail/detail?id=${event.id}` }), 500)
    } catch (error) {
	  wx.showToast({ title: (error && error.message) || (this.data.editMode ? '保存失败，请稍后重试' : '发布失败，请稍后重试'), icon: 'none' })
	} finally {
	  this.setData({ submitting: false })
    }
  }
})
