const levelMap = {
  '新手': 'beginner',
  '初级': 'primary',
  '中级': 'intermediate',
  '高级': 'advanced'
}

const levelTextMap = {
  beginner: '新手',
  primary: '初级',
  intermediate: '中级',
  advanced: '高级'
}

const skiTypeMap = { '单板': 'snowboard', '双板': 'ski', '都可以': 'both' }
const skiTypeTextMap = { snowboard: '单板', ski: '双板', both: '都可以' }
const trafficMap = { '自驾同行': 'self_drive', '高铁同行': 'high_speed_rail', '公共交通': 'bus', '同行交通待定': 'other' }
const trafficTextMap = { self_drive: '自驾同行', high_speed_rail: '高铁同行', bus: '公共交通', other: '同行交通待定' }

function buildSelectedTagsMap(tags) {
  const map = {}
  ;(tags || []).forEach((item) => {
    map[item] = true
  })
  return map
}

Page({
  data: {
	date: '',
	dateStart: '',
	resorts: ['不限'],
	resortOptions: [{ id: 0, name: '不限' }],
	resortIndex: 0,
	skiTypes: ['不限', '单板', '双板', '都可以'],
	skiType: '不限',
	trafficTypes: ['不限', '自驾同行', '高铁同行', '公共交通', '同行交通待定'],
	trafficType: '不限',
    levels: ['不限', '新手', '初级', '中级', '高级'],
    level: '不限',
    tags: ['刷道', '练习', '平花', '刻滑', '公园', '拍照'],
    selectedTags: [],
    selectedTagsMap: {},
    carpool: false,
    roomShare: false,
    beginner: false,
    sameGender: false
  },

	async onLoad(options = {}) {
	const today = new Date()
	const dateStart = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`
	this.setData({ dateStart })
	let initialFilters = {}
    if (options.filters) {
      try {
		initialFilters = JSON.parse(decodeURIComponent(options.filters))
      } catch (error) {}
    }
	try {
	  const resorts = await api.resorts()
	  const resortOptions = [{ id: 0, name: '不限' }].concat(resorts || [])
	  this.setData({ resortOptions, resorts: resortOptions.map((item) => item.name) })
	} catch (error) {}
	this.applyInitialFilters(initialFilters)
    const eventChannel = this.getOpenerEventChannel && this.getOpenerEventChannel()
    if (!eventChannel) return
    eventChannel.on('initFilters', (filters = {}) => this.applyInitialFilters(filters))
  },

  applyInitialFilters(filters = {}) {
    const selectedTags = filters.purposeTags || []
	const resortIndex = Math.max(this.data.resortOptions.findIndex((item) => Number(item.id) === Number(filters.resortId || 0)), 0)
    this.setData({
	  date: filters.date || '',
	  resortIndex,
	  skiType: skiTypeTextMap[filters.skiType] || '不限',
	  trafficType: trafficTextMap[filters.trafficType] || '不限',
      level: levelTextMap[filters.level] || '不限',
      selectedTags,
      selectedTagsMap: buildSelectedTagsMap(selectedTags),
      carpool: !!filters.allowCarPool,
      roomShare: !!filters.allowRoomShare,
      beginner: !!filters.allowBeginner,
      sameGender: !!filters.sameGenderOnly
    })
  },

  setLevel(event) {
    this.setData({ level: event.currentTarget.dataset.value })
  },
	onDateChange(event) { this.setData({ date: event.detail.value }) },
	onResortChange(event) { this.setData({ resortIndex: Number(event.detail.value) }) },
	setSkiType(event) { this.setData({ skiType: event.currentTarget.dataset.value }) },
	setTrafficType(event) { this.setData({ trafficType: event.currentTarget.dataset.value }) },

  toggleTag(event) {
    const value = event.currentTarget.dataset.value
    const selected = this.data.selectedTags.includes(value)
      ? this.data.selectedTags.filter((item) => item !== value)
      : this.data.selectedTags.concat(value)
    this.setData({ selectedTags: selected, selectedTagsMap: buildSelectedTagsMap(selected) })
  },

  onSwitch(event) {
    this.setData({ [event.currentTarget.dataset.key]: event.detail.value })
  },

  reset() {
    this.setData({
	  date: '',
	  resortIndex: 0,
	  skiType: '不限',
	  trafficType: '不限',
      level: '不限',
      selectedTags: [],
      selectedTagsMap: {},
      carpool: false,
      roomShare: false,
      beginner: false,
      sameGender: false
    })
  },

  confirm() {
    const filters = {}
	if (this.data.date) filters.date = this.data.date
	const resort = this.data.resortOptions[this.data.resortIndex]
	if (resort && resort.id) filters.resortId = Number(resort.id)
	if (skiTypeMap[this.data.skiType]) filters.skiType = skiTypeMap[this.data.skiType]
	if (trafficMap[this.data.trafficType]) filters.trafficType = trafficMap[this.data.trafficType]
    if (levelMap[this.data.level]) filters.level = levelMap[this.data.level]
    if (this.data.selectedTags.length) filters.purposeTags = this.data.selectedTags
    if (this.data.carpool) filters.allowCarPool = true
    if (this.data.roomShare) filters.allowRoomShare = true
    if (this.data.beginner) filters.allowBeginner = true
    if (this.data.sameGender) filters.sameGenderOnly = true

    const eventChannel = this.getOpenerEventChannel && this.getOpenerEventChannel()
    if (eventChannel) eventChannel.emit('applyFilters', filters)
    wx.showToast({ title: '筛选已应用', icon: 'success' })
    setTimeout(() => wx.navigateBack(), 300)
  }
})
const api = require('../../utils/api')
