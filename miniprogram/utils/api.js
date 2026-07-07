const http = require('./request')

const skiTypeText = { snowboard: '单板', ski: '双板', both: '都可以' }
const levelText = { beginner: '新手', primary: '初级', intermediate: '中级', advanced: '高级' }
const trafficText = { self_drive: '自驾同行', high_speed_rail: '高铁同行', bus: '公共交通', other: '同行交通待定' }
const statusText = { recruiting: '招募中', full: '已满员', finished: '已结束', cancelled: '已取消', removed: '已下架' }
const defaultEventImage = 'https://images.unsplash.com/photo-1740137660688-3d3f2b5422b6?auto=format&fit=crop&w=900&q=80'

function getCurrentUserId() {
  const app = typeof getApp === 'function' ? getApp() : null
  const user = (app && app.globalData && app.globalData.user) || wx.getStorageSync('xueju_user') || {}
  return user.id
}

function mapMember(item) {
  const name = item.nickname || item.name || '雪友'
  const skiType = skiTypeText[item.skiType] || item.skiType || ''
  const level = levelText[item.skiLevel] || item.skiLevel || item.level || ''
  const role = item.role === 'creator' ? '发起人' : '成员'
  const credit = item.credit || `信用 ${Number(item.creditScore || 5).toFixed(1)}`
  return {
    ...item,
    id: item.userId || item.id,
    name,
    initial: item.initial || name.slice(0, 1),
    role,
    level: [skiType, level].filter(Boolean).join(' · ') || '未填写滑雪信息',
    credit,
    avatarUrl: item.avatarUrl || ''
  }
}

function mapEvent(item) {
  if (!item) return {}
  const joined = item.currentMembers || 1
  const max = item.maxMembers || 1
  const tags = item.purposeTags || []
  const userId = getCurrentUserId()
  const members = (item.members || []).map(mapMember)
  return {
    ...item,
    resort: item.resortName || item.title,
    date: item.eventDate || '',
    time: item.startTime ? item.startTime.slice(11, 16) : '',
    depart: `${item.departCity || ''}${item.departArea ? item.departArea + '出发' : ''}`,
    level: levelText[item.levelReq] || item.levelReq || '',
    board: skiTypeText[item.skiTypeReq] || item.skiTypeReq || '',
    traffic: trafficText[item.trafficType] || item.trafficType || '',
    people: `已 ${joined} 人，缺 ${Math.max(max - joined, 0)} 人`,
    joinedText: `已 ${joined} 人`,
    spotsText: `缺 ${Math.max(max - joined, 0)} 人`,
    maxPeople: max,
    tags,
    badge: statusText[item.status] || item.status,
    tagA: tags[0] || (levelText[item.levelReq] || item.levelReq || ''),
    tagB: tags[1] || (item.allowCarPool ? '可拼车' : item.allowRoomShare ? '可拼房' : ''),
    displayTags: (tags.length ? tags : [levelText[item.levelReq] || item.levelReq || '', skiTypeText[item.skiTypeReq] || item.skiTypeReq || '']).filter(Boolean),
    host: item.creatorName || '雪友',
    hostInitial: (item.creatorName || '雪').slice(0, 1),
    hostAvatarUrl: item.creatorAvatarUrl || '',
    credit: `信用 ${Number(item.creatorCreditScore || 5).toFixed(1)}`,
    hostSub: item.creatorEventCount ? `发起局数 ${item.creatorEventCount}` : `发起人 · ${statusText[item.status] || item.status}`,
    memberAvatars: members.slice(0, 6).map((member) => ({
      id: member.id,
      avatarUrl: member.avatarUrl,
      initial: member.initial
    })),
    memberInitials: members.slice(0, 6).map((member) => member.initial),
    note: item.remark || '',
    image: item.imageUrl || item.image || defaultEventImage,
    imageUrl: item.imageUrl || item.image || '',
    isCreatedByMe: !!item.isCreatedByMe || (userId ? Number(item.creatorId) === Number(userId) : false),
    status: item.status,
    members
  }
}

function eventPayload(form) {
  const resort = form.resort || ''
  const date = /^\d{4}-\d{2}-\d{2}$/.test(form.date || '') ? form.date : ''
  const time = form.time || '06:30'
  return {
    title: form.title || `${resort}滑雪局`,
    resortId: Number(form.resortId || 0),
    resortName: resort,
    eventDate: date,
    startTime: date ? `${date} ${time}:00` : '',
    departCity: form.departCity || '北京',
    departArea: form.departArea || form.depart || '',
    meetPlace: form.meetPlace || '',
    trafficType: form.trafficType || 'self_drive',
    maxMembers: Number(form.people || form.maxMembers || 4),
    skiTypeReq: form.skiTypeReq || 'snowboard',
    levelReq: form.levelReq || 'intermediate',
    purposeTags: form.purposeTags || [form.style || '刷道'],
    allowBeginner: !!form.allowBeginner,
    sameGenderOnly: !!form.sameGenderOnly,
    allowCarPool: !!form.canCarpool,
    allowRoomShare: !!form.allowRoomShare,
    allowPhoto: !!form.photo,
    costDesc: form.costDesc || '',
    remark: form.note || form.remark || '',
    imageUrl: form.imageUrl || form.image || ''
  }
}

module.exports = {
  mapEvent,
  async login(code) {
    return http.post('/auth/wechat-login', { code })
  },
  async me() {
    return http.get('/user/me')
  },
  async updateMe(profile) {
    return http.put('/user/me', profile)
  },
  async uploadAvatar(filePath) {
    return http.upload('/uploads/avatar', filePath, 'file')
  },
  async events(params) {
    const page = await http.get('/events', params || {})
    return { ...page, list: (page.list || []).map(mapEvent) }
  },
  async event(id) {
    return mapEvent(await http.get(`/events/${id}`))
  },
  async createEvent(form) {
    return mapEvent(await http.post('/events', eventPayload(form)))
  },
  async uploadEventImage(filePath) {
    return http.upload('/uploads/event-image', filePath, 'file')
  },
  async deleteEvent(eventId) {
    return http.delete(`/events/${eventId}`)
  },
  async finishEvent(eventId) {
    return http.post(`/events/${eventId}/finish`, {})
  },
  async cancelEvent(eventId) {
    return http.post(`/events/${eventId}/cancel`, {})
  },
  async applyEvent(eventId, form) {
    return http.post(`/events/${eventId}/apply`, form)
  },
  async myJoinRequests() {
    return http.get('/join-requests/my')
  },
  async applications(eventId) {
    return http.get(`/events/${eventId}/applications`)
  },
  async reviewApplication(id, approved, reason) {
    return http.post(`/join-requests/${id}/${approved ? 'approve' : 'reject'}`, { reason })
  },
  async trips(kind) {
    const list = await http.get(`/trips/${kind}`)
    return (list || []).map(mapEvent)
  },
  async conversations() {
    return http.get('/chat/conversations')
  },
  async markAllConversationsRead() {
    return http.post('/chat/conversations/read', {})
  },
  async messages(eventId) {
    return http.get(`/events/${eventId}/messages`)
  },
  async sendMessage(eventId, content, messageType = 'text') {
    return http.post(`/events/${eventId}/messages`, { messageType, content })
  },
  async markMessagesRead(eventId) {
    return http.post(`/events/${eventId}/messages/read`, {})
  },
  async createReview(payload) {
    return http.post('/reviews', payload)
  },
  async userReviews(userId) {
    return http.get(`/users/${userId}/reviews`)
  },
  async createReport(payload) {
    return http.post('/reports', payload)
  },
  async publicUser(userId) {
    return http.get(`/users/${userId}`)
  },
  async favorites() {
    const page = await http.get('/favorites')
    return { ...page, list: (page.list || []).map(mapEvent) }
  },
  async setFavorite(eventId, favorite) {
    return favorite ? http.post(`/events/${eventId}/favorite`, {}) : http.delete(`/events/${eventId}/favorite`)
  },
  async followUser(userId, following = true) {
    return following ? http.post(`/users/${userId}/follow`, {}) : http.delete(`/users/${userId}/follow`)
  },
  async notifications() {
    return http.get('/notifications')
  },
  async markNotificationsRead() {
    return http.post('/notifications/read', {})
  },
  async clearNotifications() {
    return http.delete('/notifications')
  },
  async resorts() {
    return http.get('/dict/resorts')
  },
  async tags() {
    return http.get('/dict/tags')
  },
  async cities() {
    return http.get('/dict/cities')
  }
}
