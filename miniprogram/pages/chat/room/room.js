const { CONTENT_RISK_MESSAGE } = require('../../../constants/compliance')
const api = require('../../../utils/api')

function nowText() {
  const date = new Date()
  return `${date.getHours()}`.padStart(2, '0') + ':' + `${date.getMinutes()}`.padStart(2, '0')
}

function mapMessage(item) {
  const app = getApp()
  const user = app.globalData.user || wx.getStorageSync('xueju_user') || {}
  const mine = Number(item.senderId) === Number(user.id)
  const name = item.nickname || item.displayName || item.name || (mine ? '我' : '雪友')
  return {
    ...item,
    name,
    initial: name.slice(0, 1),
    displayName: name,
    side: mine ? 'right' : 'left',
    system: item.messageType === 'system' || item.system,
    time: item.time || nowText()
  }
}

Page({
  pollTimer: null,
  loadingMessages: false,

  data: {
    eventId: 1,
    event: {},
    messages: [],
    inputValue: '',
    toView: '',
    actionPanelVisible: false,
    quickActions: [
      { iconName: 'location', text: '集合信息', message: '集合信息：请按活动详情里的时间和地点准时集合。' },
      { iconName: 'wallet', text: '费用说明', message: '费用说明：油费、高速费和停车费按实际 AA。' },
      { iconName: 'user-safety', text: '雪场须知', message: '雪场须知：请佩戴护具，按自身水平选择雪道，安全第一。' },
      { iconName: 'edit', text: '行程备注', message: '行程备注：如临时变更集合时间，请提前在群里同步。' },
      { iconName: 'image', text: '相册', message: '我准备滑完后把照片统一发到群里。' },
      { iconName: 'map-location', text: '位置', message: '位置：集合点在停车场入口附近。' },
      { iconName: 'chart', text: '投票', message: '投票：大家想先刷道还是先热身练习？' },
      { iconName: 'more', text: '更多', message: '更多功能后续会接入。' }
    ],
    contentRiskMessage: CONTENT_RISK_MESSAGE
  },

  async onLoad(options) {
    const eventId = Number(options.eventId || 1)
    let event = {}
    try { event = await api.event(eventId) } catch (error) {}
    this.setData({ eventId, event })
    wx.setNavigationBarTitle({ title: event.resort ? `${event.resort}群聊` : '局内群聊' })
    this.loadMessages()
    this.startPolling()
  },

  onShow() {
    this.scrollToBottom()
    this.startPolling()
  },

  onHide() {
    this.stopPolling()
    this.markRead()
  },

  onUnload() {
    this.stopPolling()
    this.markRead()
  },

  async loadMessages() {
    if (this.loadingMessages) return
    this.loadingMessages = true
    try {
      const messages = await api.messages(this.data.eventId)
      this.setData({ messages: messages.map(mapMessage) }, () => {
        this.scrollToBottom()
        this.markRead()
      })
    } catch (error) {
      this.setData({ messages: [] }, () => this.scrollToBottom())
    } finally {
      this.loadingMessages = false
    }
  },

  onInput(event) { this.setData({ inputValue: event.detail.value }) },
  onConfirm() { this.send() },
  toggleActionPanel() { this.setData({ actionPanelVisible: !this.data.actionPanelVisible }, () => this.scrollToBottom()) },
  hideActionPanel() { if (this.data.actionPanelVisible) this.setData({ actionPanelVisible: false }) },

  async send() {
    const content = this.data.inputValue.trim()
    if (!content) {
      wx.showToast({ title: '请输入消息', icon: 'none' })
      return
    }
    try {
      const saved = await api.sendMessage(this.data.eventId, content)
      const next = mapMessage({ ...saved, senderId: (getApp().globalData.user || {}).id, nickname: '我', time: nowText() })
      this.setData({ messages: this.data.messages.concat(next), inputValue: '', actionPanelVisible: false }, () => {
        this.scrollToBottom()
        this.markRead()
      })
    } catch (error) {
      this.setData({ inputValue: content })
    }
  },

  sendQuickAction(event) {
    const index = Number(event.currentTarget.dataset.index)
    const action = this.data.quickActions[index]
    if (!action) return
    this.setData({ inputValue: action.message }, () => this.send())
  },

  startPolling() {
    if (this.pollTimer) return
    this.pollTimer = setInterval(() => this.loadMessages(), 5000)
  },

  stopPolling() {
    if (!this.pollTimer) return
    clearInterval(this.pollTimer)
    this.pollTimer = null
  },

  async markRead() {
    if (!this.data.eventId) return
    try { await api.markMessagesRead(this.data.eventId) } catch (error) {}
  },

  buildMessage(content, side = 'right') {
    const name = side === 'right' ? '我' : '雪友'
    return { id: Date.now(), name, initial: name.slice(0, 1), displayName: name, side, content, time: nowText() }
  },

  clearMessages() {
    wx.showToast({ title: '聊天记录由后端保存', icon: 'none' })
  },

  scrollToBottom() {
    const messages = this.data.messages
    const toView = messages.length ? `msg-${messages[messages.length - 1].id}` : ''
    setTimeout(() => this.setData({ toView }), 80)
  },

  goBack() { wx.navigateBack() }
})
