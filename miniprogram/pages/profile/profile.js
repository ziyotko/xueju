const { requirePrivacyConsent } = require("../../utils/privacy")

Page({
  data: {
    nickname: "大力",
    level: "中级",
    bio: "热爱滑雪，周末不是在雪场就是在去雪场的路上。",
    privacyReady: false,
    stats: [
      { value: 12, label: "发起局数" },
      { value: 28, label: "加入局数" },
      { value: 156, label: "相册" },
      { value: "98%", label: "好评率" }
    ],
    certifications: ["手机认证", "实名认证", "滑雪水平认证"],
    resorts: ["崇礼万龙", "崇礼云顶", "南山滑雪场", "怀北"],
    styles: ["刷道", "节奏稳", "爱拍照", "不赶时间"],
    reviews: [
      {
        name: "阿飞",
        initial: "阿",
        date: "2023-12-10",
        credit: "信用 4.8",
        text: "节奏合适，沟通顺畅，下次还愿意一起滑。",
        tags: ["准时", "友好", "水平真实", "沟通顺畅"]
      }
    ]
  },

  onLoad() {
    requirePrivacyConsent()
      .then(() => {
        this.setData({ privacyReady: true })
      })
      .catch(() => {
        wx.showToast({ title: "请先同意隐私保护指引", icon: "none" })
      })
  }
})
