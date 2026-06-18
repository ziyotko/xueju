const { CONTENT_RISK_MESSAGE } = require("../../../constants/compliance")

Page({
  data: {
    form: {
      resort: "",
      date: "",
      depart: "",
      people: 2,
      note: "",
      noteCount: 0
    },
    basicRows: [
      { label: "目的地", value: "选择雪场" },
      { label: "出发日期", value: "选择日期" },
      { label: "出发地", value: "选择出发地" }
    ],
    boards: ["单板", "双板", "都可以"],
    levels: ["新手", "初级", "中级", "高级"],
    styles: ["刷道", "练活", "平花", "卡宾", "公园", "拍照", "放松滑"],
    toggles: [
      { label: "接受新手", checked: false },
      { label: "仅限同性", checked: false },
      { label: "同行交通说明", checked: false },
      { label: "住宿需求备注", checked: false },
      { label: "互拍视频", checked: false }
    ],
    contentRiskMessage: CONTENT_RISK_MESSAGE
  },

  increasePeople() {
    this.setData({ "form.people": Math.min(this.data.form.people + 1, 12) })
  },

  decreasePeople() {
    this.setData({ "form.people": Math.max(this.data.form.people - 1, 1) })
  },

  onNoteInput(event) {
    const note = event.detail.value
    this.setData({
      "form.note": note,
      "form.noteCount": note.length
    })
  },

  submit() {
    wx.showToast({ title: "发布接口待接入", icon: "none" })
  }
})
