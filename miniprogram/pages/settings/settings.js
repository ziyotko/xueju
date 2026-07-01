Page({
  goProfileEdit(){wx.navigateTo({url:'/pages/profile/edit/edit'})},
  goFavorites(){wx.navigateTo({url:'/pages/favorites/favorites'})},
  goNotifications(){wx.navigateTo({url:'/pages/notifications/notifications'})},
  goReport(){wx.navigateTo({url:'/pages/report/report?targetType=app&targetId=0'})},
  showPrivacy(){wx.showModal({title:'隐私保护指引',content:'雪局仅用于滑雪行程组局演示，涉及昵称、头像、行程、聊天和评价等信息。上线前需接入正式隐私协议。',showCancel:false})},
  deleteAccount(){wx.showModal({title:'确认注销账号？',content:'演示版将清空本地资料、聊天、申请、收藏和评价。',success:(res)=>{if(!res.confirm)return;wx.clearStorageSync();wx.showToast({title:'已清空本地数据',icon:'success'})}})}
})
