import axios, { AxiosError, type AxiosResponse } from "axios"
import { ElMessage } from "element-plus"

interface ApiBody<T> {
  code: number
  message: string
  data: T
}

export const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || "/api",
  timeout: 10000
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem("xueju_admin_token")
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiBody<unknown>>) => {
	if (error.response?.status === 401 && window.location.pathname !== "/login") {
	  localStorage.removeItem("xueju_admin_token")
	  localStorage.removeItem("xueju_admin_user")
	  const redirect = encodeURIComponent(window.location.pathname + window.location.search)
	  window.location.assign(`/login?redirect=${redirect}`)
	}
    const message = error.response?.data?.message || "网络连接失败"
    ElMessage.error(message)
    return Promise.reject(error)
  }
)

async function unwrap<T>(promise: Promise<AxiosResponse<ApiBody<T>>>) {
  const response = await promise
  const body = response.data
  if (body.code === 0) {
    return body.data
  }

  ElMessage.error(body.message || "请求失败")
  return Promise.reject(new Error(body.message || "请求失败"))
}

export async function getHealth() {
  return unwrap(http.get<ApiBody<HealthInfo>>("/health"))
}

export async function loginAdmin(payload: { username: string; password: string }) {
  return unwrap(http.post<ApiBody<{ token: string; user: { username: string; role: string } }>>("/admin/auth/login", payload))
}

export async function getAdminDashboard() {
  return unwrap(http.get<ApiBody<AdminDashboard>>("/admin/dashboard"))
}

export async function saveDict(payload: DictPayload, id?: string | number) {
  if (id) {
    return unwrap(http.put<ApiBody<{ id: number }>>(`/admin/dicts/${id}`, payload))
  }
  return unwrap(http.post<ApiBody<{ id: number }>>("/admin/dicts", payload))
}

export async function getAdminPage(resource: AdminResource) {
  return unwrap(http.get<ApiBody<PageResult<Record<string, unknown>>>>(`/admin/${resource}`))
}

export async function getAdminPageWithParams<T = Record<string, unknown>>(resource: AdminResource, params: Record<string, unknown>) {
  return unwrap(http.get<ApiBody<PageResult<T>>>(`/admin/${resource}`, { params }))
}

export async function uploadResortImage(file: File) {
  const formData = new FormData()
  formData.append("file", file)
  return unwrap(http.post<ApiBody<{ id: number; url: string }>>("/admin/uploads/resort-image", formData))
}

export async function runAdminAction(resource: AdminResource, id: string | number, payload: AdminActionPayload) {
  return unwrap(http.post<ApiBody<{ status: string }>>(`/admin/${resource}/${id}/actions`, payload))
}

export async function getUserVerifications(id: string | number) {
  return unwrap(http.get<ApiBody<Array<Record<string, unknown>>>>(`/admin/users/${id}/verifications`))
}

export async function getModerationSummary() {
  return unwrap(http.get<ApiBody<ModerationSummary>>("/admin/moderation/summary"))
}

export async function getModerationPage(tab: ModerationTab, params: Record<string, unknown>) {
  if (tab === "content") {
    return getAdminPageWithParams<ModerationRow>("content-reviews", { ...params, scope: "content" })
  }
  return getAdminPageWithParams<ModerationRow>(tab === "messages" ? "messages" : "uploads", params)
}

export async function getUploadPreview(id: string | number) {
  const response = await http.get<Blob>(`/admin/uploads/${id}/preview`, { responseType: "blob" })
  return response.data
}

export async function getAuditHistory(resource: string, targetId: string | number) {
  return getAdminPageWithParams<ModerationAuditRow>("audit-logs", {
    resource,
    targetId,
    page: 1,
    pageSize: 20
  })
}

export interface HealthInfo {
  app: string
  env: string
  database: string
  timestamp: string
}

export type AdminResource = "users" | "events" | "applications" | "messages" | "reviews" | "reports" | "content-reviews" | "dicts" | "uploads" | "audit-logs"

export type ModerationTab = "content" | "messages" | "media"

export interface ModerationRow {
  id: string
  initial: string
  target: string
  type: string
  summary: string
  risk: string
  status: string
  updatedAt: string
  itemType?: "user" | "event" | "application" | "review"
  eventId?: number
  eventTitle?: string
  senderId?: number
  messageType?: string
  userId?: number
  kind?: string
  mimeType?: string
  result?: string
  previewPath?: string
}

export interface ModerationAuditRow extends ModerationRow {
  beforeStatus?: string
  afterStatus?: string
  resource?: string
  targetId?: string
  action?: string
  detail?: string
}

export interface ModerationSummary {
  content: { total: number; active: number; handled: number }
  messages: { total: number; normal: number; hidden: number }
  media: { total: number; pending: number; approved: number; rejected: number }
}

export interface AdminActionPayload {
  action: string
  status?: string
  reason?: string
  result?: string
	linkedAction?: boolean
}

export interface DictPayload {
  name: string
  city: string
  province: string
  imageUrl?: string
  sort?: number
  status?: string
}

export interface PageResult<T> {
  list: T[]
  page: number
  pageSize: number
  total: number
}

export interface AdminDashboard {
  contentReview: {
    pending: number
    approved: number
    rejected: number
  }
  reports: {
    pending: number
    handled: number
  }
  totals?: {
    users: number
    events: number
    applications: number
    reviews: number
  }
  actions: string[]
}
