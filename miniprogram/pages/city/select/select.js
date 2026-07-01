Page({
  data:{cities:[{name:'北京',desc:'南山 / 怀北 / 渔阳 / 崇礼出发'},{name:'张家口',desc:'万龙 / 云顶 / 太舞'},{name:'吉林',desc:'北大湖 / 松花湖'},{name:'乌鲁木齐',desc:'丝绸之路 / 将军山'}]},
  selectCity(e){const city=e.currentTarget.dataset.name;wx.showToast({title:`已切换到${city}`,icon:'success'});setTimeout(()=>wx.reLaunch({url:`/pages/index/index?city=${encodeURIComponent(city)}`}),400)}
})
