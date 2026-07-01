const { events: mockEvents, messages: mockMessages, members: mockMembers } = require('../data/mock')

const KEYS = {
  events: 'xueju_events_v3',
  requests: 'xueju_join_requests_v3',
  reviews: 'xueju_reviews_v3',
  favorites: 'xueju_favorites_v3',
  profile: 'xueju_profile_v3',
  notifications: 'xueju_notifications_v3'
}

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

function read(key, fallback) {
  const value = wx.getStorageSync(key)
  if (value === '' || value === null || typeof value === 'undefined') return clone(fallback)
  return value
}

function write(key, value) {
  wx.setStorageSync(key, value)
  return value
}

function getEvents() {
  const created = read(KEYS.events, [])
  return created.concat(clone(mockEvents))
}

function getEventById(id) {
  return getEvents().find((item) => Number(item.id) === Number(id)) || clone(mockEvents[0])
}

function addEvent(form) {
  const event = {
    id: Date.now(),
    resort: form.resort || '崇礼 · 万龙滑雪场',
    date: form.date || '12月23日（周六）',
    time: form.time || '06:30',
    depart: form.depart || '北京朝阳出发',
    meetPlace: form.meetPlace || '朝阳大悦城停车场',
    level: form.level || '中级',
    board: form.board || '单板',
    traffic: form.traffic || '自驾同行',
    people: `已1人，缺${form.people || 2}人`,
    joinedText: '已 1 人',
    spotsText: `缺 ${form.people || 2} 人`,
    maxPeople: Number(form.people || 2) + 1,
    image: form.image || mockEvents[0].image,
    tags: ['我发起的', `${form.board || '单板'}${form.level || '中级'}`, form.style || '刷道', form.canCarpool ? '可拼车' : '可同行'],
    badge: '我发起的',
    tagA: form.style || '刷道',
    tagB: form.canCarpool ? '可拼车' : '可同行',
    displayTags: [form.style || '刷道', form.canCarpool ? '可拼车' : '可同行'],
    memberInitials: ['我'],
    host: '我',
    hostInitial: '我',
    credit: '信用 4.8',
    hostSub: '刚刚发布 · 待成局',
    note: form.note || '希望找水平相近的雪友一起滑。',
    status: 'open',
    isCreatedByMe: true
  }
  const list = read(KEYS.events, [])
  write(KEYS.events, [event].concat(list))
  addNotification('滑雪局发布成功', `${event.resort} 已进入招募中`)
  return event
}

function deleteEvent(id) {
  const eventId = Number(id)
  const list = read(KEYS.events, [])
  const next = list.filter((item) => Number(item.id) !== eventId)
  if (list.length === next.length) return false
  write(KEYS.events, next)
  write(KEYS.favorites, getFavorites().filter((item) => Number(item) !== eventId))
  addNotification('雪局已删除', '你发布的雪局已从本地演示数据中移除')
  return true
}

function getJoinRequests() {
  const stored = read(KEYS.requests, [])
  if (stored.length) return stored
  return [
    {
      id: 10001,
      eventId: 1,
      eventTitle: '崇礼 · 万龙滑雪场',
      name: '大力',
      initial: '大',
      level: '中级',
      skiType: '单板',
      departArea: '北京朝阳',
      message: '单板中级，能连续换刃，想一起刷道互拍。',
      credit: '信用 4.6',
      status: 'pending',
      createdAt: new Date().toISOString()
    },
    {
      id: 10002,
      eventId: 1,
      eventTitle: '崇礼 · 万龙滑雪场',
      name: '小鹿',
      initial: '鹿',
      level: '初中级',
      skiType: '双板',
      departArea: '北京海淀',
      message: '想轻松刷道，可以互拍，准时集合。',
      credit: '信用 4.9',
      status: 'pending',
      createdAt: new Date().toISOString()
    }
  ]
}

function addJoinRequest(request) {
  const list = getJoinRequests()
  const exists = list.find((item) => Number(item.eventId) === Number(request.eventId) && item.name === '我' && item.status !== 'rejected')
  if (exists) return { duplicated: true, request: exists }
  const next = { ...request, id: Date.now(), status: 'pending', name: '我', initial: '我', credit: '信用 4.8', createdAt: new Date().toISOString() }
  write(KEYS.requests, [next].concat(list))
  addNotification('申请已提交', `你已申请加入 ${request.eventTitle}`)
  return { duplicated: false, request: next }
}

function updateJoinRequest(id, status) {
  const list = getJoinRequests().map((item) => (Number(item.id) === Number(id) ? { ...item, status } : item))
  write(KEYS.requests, list)
  const request = list.find((item) => Number(item.id) === Number(id))
  if (request) {
    addNotification(status === 'approved' ? '申请已通过' : '申请已拒绝', `${request.eventTitle || '滑雪局'} · ${request.name || '成员'}`)
  }
  return list
}

function getMessages(eventId = 1) {
  return read(`xueju_chat_messages_${eventId}`, clone(mockMessages))
}

function saveMessages(eventId = 1, messages) {
  return write(`xueju_chat_messages_${eventId}`, messages)
}

function getReviews() {
  return read(KEYS.reviews, [])
}

function addReview(review) {
  const list = getReviews()
  const next = { ...review, id: Date.now(), createdAt: new Date().toISOString() }
  write(KEYS.reviews, [next].concat(list))
  addNotification('评价已提交', '你的滑后评价已保存')
  return next
}

function getFavorites() {
  return read(KEYS.favorites, [])
}

function toggleFavorite(eventId) {
  const list = getFavorites()
  const id = Number(eventId)
  const exists = list.includes(id)
  const next = exists ? list.filter((item) => item !== id) : [id].concat(list)
  write(KEYS.favorites, next)
  return !exists
}

function isFavorite(eventId) {
  return getFavorites().includes(Number(eventId))
}

function getProfile() {
  return read(KEYS.profile, {
    nickname: '大力',
    avatarUrl: '',
    avatarLocalPath: '',
    phone: '',
    city: '北京',
    bio: '热爱滑雪，周末不是在雪场就是在去雪场的路上。',
    level: '中级',
    skiType: '单板',
    styles: ['刷道', '节奏稳', '爱拍照']
  })
}

function saveLocalFile(tempFilePath) {
  return new Promise((resolve) => {
    if (!tempFilePath || /^https?:\/\//.test(tempFilePath)) {
      resolve(tempFilePath || '')
      return
    }
    const fs = wx.getFileSystemManager && wx.getFileSystemManager()
    if (!fs || !wx.env || !wx.env.USER_DATA_PATH) {
      resolve(tempFilePath)
      return
    }
    const extMatch = tempFilePath.match(/\.[a-z0-9]+($|\?)/i)
    const ext = extMatch ? extMatch[0].replace('?', '') : '.jpg'
    const savedPath = `${wx.env.USER_DATA_PATH}/avatar-${Date.now()}${ext}`
    fs.copyFile({
      srcPath: tempFilePath,
      destPath: savedPath,
      success: () => resolve(savedPath),
      fail: () => resolve(tempFilePath)
    })
  })
}

function saveProfile(profile) {
  write(KEYS.profile, profile)
  addNotification('资料已更新', '你的个人主页资料已保存')
  return profile
}

function getNotifications() {
  return read(KEYS.notifications, [
    { id: 1, title: '欢迎来到雪局', content: '按雪场、日期和水平找到靠谱雪友。', time: '刚刚', read: false },
    { id: 2, title: '安全提醒', content: '线下同行前请确认集合信息，滑雪安全第一。', time: '今天', read: false }
  ])
}

function addNotification(title, content) {
  const list = getNotifications()
  write(KEYS.notifications, [{ id: Date.now(), title, content, time: '刚刚', read: false }].concat(list))
}

function markNotificationsRead() {
  const list = getNotifications().map((item) => ({ ...item, read: true }))
  write(KEYS.notifications, list)
  return list
}

module.exports = {
  getEvents,
  getEventById,
  addEvent,
  deleteEvent,
  getJoinRequests,
  addJoinRequest,
  updateJoinRequest,
  getMessages,
  saveMessages,
  getReviews,
  addReview,
  getFavorites,
  toggleFavorite,
  isFavorite,
  getProfile,
  saveProfile,
  saveLocalFile,
  getNotifications,
  addNotification,
  markNotificationsRead,
  mockMembers
}
