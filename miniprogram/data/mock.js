const mountainImages = [
  "https://images.unsplash.com/photo-1488590528505-98d2b5aba04b?auto=format&fit=crop&w=800&q=80",
  "https://images.unsplash.com/photo-1551524559-8af4e6624178?auto=format&fit=crop&w=800&q=80",
  "https://images.unsplash.com/photo-1605540436563-5bca919ae766?auto=format&fit=crop&w=800&q=80"
]

const events = [
  {
    id: 1,
    resort: "崇礼 · 万龙滑雪场",
    date: "12月16日（周六）",
    time: "06:30",
    depart: "北京朝阳出发",
    level: "中级",
    board: "单板",
    traffic: "自驾同行",
    people: "已2人，缺2人",
    maxPeople: 4,
    image: mountainImages[0],
    tags: ["周六出发", "单板中级", "刷道", "可同行"],
    badge: "周六出发",
    tagA: "刷道",
    tagB: "可同行",
    host: "阿飞",
    hostInitial: "阿",
    credit: "信用 4.8",
    note: "周末刷道，寻找节奏一致的同行伙伴。",
    status: "open"
  },
  {
    id: 2,
    resort: "南山滑雪场",
    date: "12月17日（周日）",
    time: "07:00",
    depart: "北京海淀出发",
    level: "初中级",
    board: "双板",
    traffic: "公共交通",
    people: "已2人，缺2人",
    maxPeople: 4,
    image: mountainImages[1],
    tags: ["周日出发", "双板初中级", "练习", "放松滑"],
    badge: "周日出发",
    tagA: "练习",
    tagB: "放松滑",
    host: "小鹿",
    hostInitial: "鹿",
    credit: "信用 4.9",
    note: "轻松练习，适合想稳定节奏的同水平雪友。",
    status: "open"
  },
  {
    id: 3,
    resort: "云顶滑雪公园",
    date: "12月16日（周六）",
    time: "07:20",
    depart: "北京东城出发",
    level: "高级",
    board: "单板",
    traffic: "同行交通待定",
    people: "已3人，缺1人",
    maxPeople: 4,
    image: mountainImages[2],
    tags: ["周六出发", "单板高级", "公园", "平花"],
    badge: "周六出发",
    tagA: "公园",
    tagB: "平花",
    host: "大力",
    hostInitial: "大",
    credit: "信用 4.7",
    note: "以公园和刻滑为主，住宿需求可在备注里说明。",
    status: "open"
  }
]

const members = [
  { id: 1, name: "阿飞", initial: "阿", role: "发起人", level: "单板 · 中级", credit: "信用 4.8" },
  { id: 2, name: "大力", initial: "大", role: "成员", level: "单板 · 中级", credit: "信用 4.6" }
]

const messages = [
  { id: 1, name: "阿飞", initial: "阿", displayName: "阿飞（发起人）", side: "left", content: "大家好，周六见！", time: "10:30" },
  { id: 2, name: "我", initial: "我", displayName: "我", side: "right", content: "Hi，明天见！", time: "10:31" },
  { id: 3, name: "大力", initial: "大", displayName: "大力", side: "left", content: "我到时候开车，朝阳出发。", time: "10:31" },
  { id: 4, name: "小鹿", initial: "鹿", displayName: "小鹿", side: "left", content: "我带运动相机，可以互拍。", time: "10:32" },
  {
    id: 5,
    name: "阿飞",
    initial: "阿",
    displayName: "阿飞",
    side: "left",
    content: "集合信息：12月16日 06:30，朝阳大悦城停车场。记得带好装备，注意保暖。",
    time: "10:33",
    system: true
  }
]

module.exports = {
  events,
  members,
  messages,
  heroImage: mountainImages[0]
}
