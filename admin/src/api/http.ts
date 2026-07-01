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

export async function getAdminDashboard() {
  return unwrap(http.get<ApiBody<AdminDashboard>>("/admin/dashboard"))
}

export async function getAdminPage(resource: AdminResource) {
  return unwrap(http.get<ApiBody<PageResult<Record<string, unknown>>>>(`/admin/${resource}`))
}

export async function getAdminPageWithParams(resource: AdminResource, params: Record<string, unknown>) {
  return unwrap(http.get<ApiBody<PageResult<Record<string, unknown>>>>(`/admin/${resource}`, { params }))
}

export async function runAdminAction(resource: AdminResource, id: string | number, payload: AdminActionPayload) {
  return unwrap(http.post<ApiBody<{ status: string }>>(`/admin/${resource}/${id}/actions`, payload))
}

export interface HealthInfo {
  app: string
  env: string
  database: string
  timestamp: string
}

export type AdminResource = "users" | "events" | "applications" | "messages" | "reviews" | "reports" | "content-reviews" | "dicts"

export interface AdminActionPayload {
  action: string
  status?: string
  reason?: string
  result?: string
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
