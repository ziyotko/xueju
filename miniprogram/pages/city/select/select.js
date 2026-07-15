const api = require('../../../utils/api')

const fallbackCities = ['北京', '张家口', '崇礼', '吉林', '哈尔滨', '乌鲁木齐']

Page({
  data: {
    cities: fallbackCities.map((name) => ({ name, desc: '查看当地出发的滑雪行程' }))
  },

  async onLoad() {
    try {
      const cities = await api.cities()
      if (cities && cities.length) {
        this.setData({ cities: cities.map((name) => ({ name, desc: '查看当地出发的滑雪行程' })) })
      }
    } catch (error) {}
  },

  selectCity(event) {
    const city = event.currentTarget.dataset.name
    wx.showToast({ title: `已切换到${city}`, icon: 'success' })
    setTimeout(() => wx.reLaunch({ url: `/pages/index/index?city=${encodeURIComponent(city)}` }), 400)
  }
})
