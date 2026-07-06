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

function buildSelectedTagsMap(tags) {
  const map = {}
  ;(tags || []).forEach((item) => {
    map[item] = true
  })
  return map
}

Page({
  data: {
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

  onLoad(options = {}) {
    if (options.filters) {
      try {
        this.applyInitialFilters(JSON.parse(decodeURIComponent(options.filters)))
      } catch (error) {}
    }
    const eventChannel = this.getOpenerEventChannel && this.getOpenerEventChannel()
    if (!eventChannel) return
    eventChannel.on('initFilters', (filters = {}) => this.applyInitialFilters(filters))
  },

  applyInitialFilters(filters = {}) {
    const selectedTags = filters.purposeTags || []
    this.setData({
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
