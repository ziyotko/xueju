Page({
  data:{levels:['不限','新手','初级','中级','高级'],level:'不限',tags:['刷道','练活','平花','刻滑','公园','拍照'],selectedTags:[],selectedTagsMap:{},carpool:false,beginner:false,sameGender:false},
  setLevel(e){this.setData({level:e.currentTarget.dataset.value})},
  toggleTag(e){const v=e.currentTarget.dataset.value;const selected=this.data.selectedTags.includes(v)?this.data.selectedTags.filter(i=>i!==v):this.data.selectedTags.concat(v);const map={};selected.forEach(i=>map[i]=true);this.setData({selectedTags:selected,selectedTagsMap:map})},
  onSwitch(e){this.setData({[e.currentTarget.dataset.key]:e.detail.value})},
  reset(){this.setData({level:'不限',selectedTags:[],selectedTagsMap:{},carpool:false,beginner:false,sameGender:false})},
  confirm(){wx.showToast({title:'筛选已应用',icon:'success'});setTimeout(()=>wx.navigateBack(),500)}
})
