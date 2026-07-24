import type { AdminResource } from "../api/http"

const fieldLabels: Record<string, string> = {
  id: "编号",
  target: "对象",
  type: "记录类型",
  summary: "内容摘要",
  risk: "风险等级",
  status: "当前状态",
  updatedAt: "更新时间",
  createdAt: "创建时间",
  creditScore: "信用分",
  verificationStatus: "手机号认证状态",
  verificationMethod: "手机号认证方式",
  phoneMasked: "脱敏手机号",
  verifiedAt: "认证时间",
  revokedAt: "撤销时间",
  revokedBy: "撤销操作人",
  revokeReason: "撤销原因",
  provider: "认证服务商",
  providerReference: "服务商业务编号",
  method: "认证方式",
  result: "处理结果",
  targetType: "对象类型",
  targetId: "对象编号",
  eventId: "行程编号",
  eventTitle: "所属行程",
  senderId: "发送人编号",
  messageType: "消息类型",
  score: "评分",
  itemType: "内容类型",
  userId: "上传人编号",
  kind: "媒体用途",
  mimeType: "文件格式",
  resource: "业务模块",
  action: "执行操作",
  beforeStatus: "操作前状态",
  afterStatus: "操作后状态",
  detail: "操作说明",
  extra: "补充信息",
  city: "城市",
  province: "省份",
  imageUrl: "图片地址",
  sort: "显示顺序",
  linkedAction: "是否联动处置",
  previewPath: "预览接口"
}

const statusLabels: Record<string, string> = {
  all: "全部状态",
  normal: "正常",
  active: "未处置",
  handled: "已处置",
  disabled: "已禁用",
  recruiting: "招募中",
  full: "已满员",
  finished: "已结束",
  cancelled: "已取消",
  removed: "已下架",
  pending: "待处理",
  processing: "处理中",
  approved: "已通过",
  resolved: "已处理",
  rejected: "已拒绝",
  hidden: "已隐藏",
  unverified: "未认证",
  verified: "已认证",
  reverify_required: "需要重新认证",
  revoked: "认证已撤销",
  passed: "已通过",
  failed: "未通过",
  accepted: "已受理",
  deleted: "已删除"
}

const resourceLabels: Record<string, string> = {
  users: "用户",
  user: "用户",
  events: "滑雪行程",
  event: "行程图片",
  applications: "加入申请",
  application: "加入申请",
  messages: "群聊消息",
  message: "群聊消息",
  reviews: "滑后评价",
  review: "滑后评价",
  reports: "举报",
  report: "举报",
  dicts: "雪场字典",
  dict: "雪场字典",
  uploads: "媒体文件",
  upload: "媒体文件",
  "content-reviews": "内容巡检",
  "audit-logs": "操作审计"
}

const actionLabels: Record<string, string> = {
  disable_user: "禁用用户",
  enable_user: "启用用户",
  delist_event: "下架行程",
  restore_event: "恢复行程",
  reject_application: "拒绝加入申请",
  hide_message: "隐藏群聊消息",
  restore_message: "恢复群聊消息",
  hide_review: "隐藏评价",
  restore_review: "恢复评价",
  resolve_report: "处理举报",
  approve_upload: "通过媒体审核",
  reject_upload: "拒绝或下架媒体",
  disable_dict: "停用雪场",
  enable_dict: "启用雪场",
  create_dict: "新增雪场",
  update_dict: "修改雪场",
  require_phone_reverification: "要求重新认证手机号",
  revoke_phone_verification: "撤销手机号认证",
  review_content: "处置内容"
}

const verificationLabels: Record<string, string> = {
  unverified: "未认证",
  pending: "认证中",
  verified: "已完成手机号认证",
  reverify_required: "需要重新认证",
  revoked: "认证已撤销"
}

const enumLabels: Record<string, string> = {
  text: "文本消息",
  image: "图片消息",
  system: "系统消息",
  avatars: "用户头像",
  avatar: "用户头像",
  events: "行程图片",
  event: "滑雪行程",
  pass: "机器审核通过",
  risky: "机器识别为风险内容",
  review: "需要人工复核",
  legacy_migration: "历史数据迁移",
  account_deleted: "账号已注销",
  sms: "短信验证码",
  sms_code: "短信验证码",
  sms_phone: "手机号短信认证",
  phone_sms: "短信验证码",
  wechat: "微信内容安全",
  aliyun: "阿里云",
  aliyun_sms: "阿里云短信",
  aliyun_sms_auth: "阿里云短信认证",
  aliyun_dysmsapi: "阿里云短信",
  admin: "管理员",
  development: "开发环境",
  production: "生产环境",
  test: "测试环境",
  ok: "正常",
  connected: "已连接",
  high: "高",
  medium: "中",
  low: "低",
  snowboard: "单板",
  ski: "双板",
  both: "单双板均可",
  beginner: "新手",
  primary: "初级",
  intermediate: "中级",
  advanced: "高级",
  self_drive: "自驾同行",
  public_transport: "公共交通",
  "image/jpeg": "JPEG 图片",
  "image/png": "PNG 图片",
  "image/webp": "WebP 图片",
  "image/gif": "GIF 图片"
}

export function adminFieldLabel(key: string) {
  return fieldLabels[key] || key
}

export function adminStatusLabel(value: unknown, resource?: AdminResource | string) {
  const status = String(value || "")
  if (status === "pending" && resource === "uploads") return "待审核"
  return statusLabels[status] || status || "—"
}

export function adminResourceLabel(value: unknown) {
  const resource = String(value || "")
  return resourceLabels[resource] || resource || "—"
}

export function adminActionLabel(value: unknown) {
  const action = String(value || "")
  return actionLabels[action] || action || "—"
}

export function adminVerificationLabel(value: unknown) {
  const status = String(value || "")
  return verificationLabels[status] || statusLabels[status] || status || "未认证"
}

export function adminRiskLabel(value: unknown) {
  const risk = String(value || "")
  return enumLabels[risk.toLowerCase()] || risk || "—"
}

export function adminEnumLabel(value: unknown) {
  const text = String(value || "")
  return enumLabels[text.toLowerCase()] || resourceLabels[text] || text || "—"
}

export function adminDateLabel(value: unknown) {
  if (!value) return "—"
  const date = new Date(String(value))
  return Number.isNaN(date.getTime())
    ? String(value)
    : new Intl.DateTimeFormat("zh-CN", { dateStyle: "medium", timeStyle: "short" }).format(date)
}

export function adminDetailLabel(value: unknown) {
  return String(value || "")
    .replace(/\brequestId=/g, "请求编号：")
    .replace(/\bip=/g, "网络地址：")
    .replace(/\bsnowboard\b/gi, "单板")
    .replace(/\bski\b/gi, "双板")
    .replace(/\bboth\b/gi, "单双板均可")
    .replace(/\bbeginner\b/gi, "新手")
    .replace(/\bprimary\b/gi, "初级")
    .replace(/\bintermediate\b/gi, "中级")
    .replace(/\badvanced\b/gi, "高级")
    .replace(/\bself_drive\b/gi, "自驾同行")
    .replace(/\bpublic_transport\b/gi, "公共交通")
}

export function adminValueLabel(key: string, value: unknown, resource?: AdminResource | string) {
  if (value === null || value === undefined || value === "") return "—"
  if (typeof value === "boolean") return value ? "是" : "否"
  if (["status", "beforeStatus", "afterStatus"].includes(key)) return adminStatusLabel(value, resource)
  if (key === "verificationStatus") return adminVerificationLabel(value)
  if (key === "resource") return adminResourceLabel(value)
  if (key === "action") return adminActionLabel(value)
  if (key === "risk") return adminRiskLabel(value)
  if (key === "result") return adminEnumLabel(value)
  if (key === "type" || key === "target") {
    const text = String(value)
    const matched = text.match(/^([a-z-]+)\s+#(.+)$/i)
    return matched ? `${adminResourceLabel(matched[1])} #${matched[2]}` : adminEnumLabel(text)
  }
  if (key === "summary") {
    const text = String(value)
    const [action, ...detailParts] = text.split(" · ")
    return actionLabels[action]
      ? [adminActionLabel(action), adminDetailLabel(detailParts.join(" · "))].filter(Boolean).join(" · ")
      : adminDetailLabel(text)
  }
  if (["targetType", "itemType"].includes(key)) return adminResourceLabel(value)
  if (["messageType", "kind", "mimeType", "method", "verificationMethod", "provider"].includes(key)) {
    return adminEnumLabel(value)
  }
  if (key === "detail") return adminDetailLabel(value)
  if (key.endsWith("At") || key.endsWith("Time")) return adminDateLabel(value)
  return String(value)
}

export function isAdminDetailObject(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value)
}
