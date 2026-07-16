<template>
  <div class="moderation-page">
    <section class="panel moderation-panel">
      <el-tabs v-model="activeTab" class="moderation-tabs">
        <el-tab-pane name="content">
          <template #label>内容巡检 <span class="moderation-tab-count">{{ summary?.content.total || 0 }}</span></template>
        </el-tab-pane>
        <el-tab-pane name="messages">
          <template #label>群聊巡检 <span class="moderation-tab-count">{{ summary?.messages.total || 0 }}</span></template>
        </el-tab-pane>
        <el-tab-pane name="media">
          <template #label>媒体审核 <span class="moderation-tab-count">待审 {{ summary?.media.pending || 0 }}</span></template>
        </el-tab-pane>
      </el-tabs>

      <div class="moderation-toolbar">
        <el-input
          v-model="currentState.keyword"
          class="search-input"
          clearable
          :placeholder="searchPlaceholder"
          @keyup.enter="applyFilters"
        />
        <el-select v-if="activeTab === 'content'" v-model="currentState.itemType" class="moderation-select" aria-label="内容类型">
          <el-option label="全部内容" value="all" />
          <el-option label="用户资料" value="user" />
          <el-option label="滑雪局" value="event" />
          <el-option label="加入申请" value="application" />
          <el-option label="滑后评价" value="review" />
        </el-select>
        <el-select v-model="currentState.status" class="moderation-select" aria-label="处置状态">
          <el-option v-for="option in statusOptions" :key="option.value" :label="option.label" :value="option.value" />
        </el-select>
        <el-button type="primary" @click="applyFilters">筛选</el-button>
        <el-button @click="resetFilters">重置</el-button>
      </div>

      <el-alert
        v-if="currentState.error"
        class="moderation-load-error"
        :title="currentState.error"
        type="error"
        show-icon
        :closable="false"
      >
        <template #default>
          <el-button link type="danger" @click="loadData()">重新加载</el-button>
        </template>
      </el-alert>

      <div class="panel-head moderation-list-head">
        <div>
          <h2>{{ tabTitle }}</h2>
          <p>{{ tabDescription }}</p>
        </div>
        <div class="panel-head-actions">
          <el-tag effect="plain">共 {{ currentState.total }} 条</el-tag>
          <el-button :loading="currentState.loading" @click="refreshCurrent">刷新</el-button>
        </div>
      </div>

      <div class="table-scroll-area">
        <el-table :data="currentState.list" v-loading="currentState.loading" class="soft-table" height="100%">
        <el-table-column prop="id" label="编号" width="112" show-overflow-tooltip />
        <el-table-column label="对象" min-width="190">
          <template #default="{ row }">
            <div class="object-cell">
              <div v-if="activeTab !== 'media'" class="object-avatar">{{ row.initial || '-' }}</div>
              <div v-else class="media-thumbnail" @click="showDetail(row)">
                <img v-if="previewUrls[row.id]" :src="previewUrls[row.id]" alt="待审媒体缩略图" />
                <span v-else>图片</span>
              </div>
              <div>
                <div class="object-title">{{ displayTarget(row) }}</div>
                <div class="object-subtitle">{{ rowTypeLabel(row) }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="内容" min-width="340">
          <template #default="{ row }">
            <span class="content-preview" :title="row.summary">{{ row.summary || "-" }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="activeTab === 'media'" label="机器审核" width="132">
          <template #default="{ row }">{{ machineResultLabel(row.result) }}</template>
        </el-table-column>
        <el-table-column label="处置状态" width="126">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" effect="plain">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="172">
          <template #default="{ row }">{{ formatDate(row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="230" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="showDetail(row)">查看</el-button>
            <el-button
              v-for="action in rowActions(row)"
              :key="action.action"
              link
              :type="action.danger ? 'danger' : 'primary'"
              @click="executeAction(row, action)"
            >{{ action.label }}</el-button>
          </template>
        </el-table-column>
        <template #empty><el-empty :description="emptyDescription" /></template>
        </el-table>
      </div>

      <div class="pagination-row">
        <el-pagination
          :current-page="currentState.page"
          :page-size="currentState.pageSize"
          layout="prev, pager, next, sizes"
          :total="currentState.total"
          :page-sizes="[10, 20, 50]"
          @current-change="changePage"
          @size-change="changePageSize"
        />
      </div>
    </section>

    <el-drawer v-model="detailVisible" title="审核详情" size="560px" destroy-on-close>
      <template v-if="selectedRow">
        <div v-if="activeTab === 'media'" class="drawer-media-preview">
          <img v-if="previewUrls[selectedRow.id]" :src="previewUrls[selectedRow.id]" alt="媒体审核预览" />
          <el-empty v-else description="图片暂时无法加载" />
        </div>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="编号">{{ selectedRow.id }}</el-descriptions-item>
          <el-descriptions-item label="类型">{{ rowTypeLabel(selectedRow) }}</el-descriptions-item>
          <el-descriptions-item label="对象">{{ displayTarget(selectedRow) }}</el-descriptions-item>
          <el-descriptions-item v-if="activeTab === 'messages'" label="所属行程">{{ selectedRow.eventTitle || `#${selectedRow.eventId}` }}</el-descriptions-item>
          <el-descriptions-item v-if="activeTab === 'media'" label="文件格式">{{ selectedRow.mimeType || '-' }}</el-descriptions-item>
          <el-descriptions-item v-if="activeTab === 'media'" label="机器结果">{{ machineResultLabel(selectedRow.result) }}</el-descriptions-item>
          <el-descriptions-item label="处置状态">{{ statusLabel(selectedRow.status) }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ formatDate(selectedRow.updatedAt) }}</el-descriptions-item>
        </el-descriptions>

        <div v-if="activeTab !== 'media'" class="drawer-section">
          <h3>完整内容</h3>
          <div class="drawer-content">{{ selectedRow.summary || "暂无内容" }}</div>
        </div>

        <div v-if="rowActions(selectedRow).length" class="drawer-actions">
          <el-button
            v-for="action in rowActions(selectedRow)"
            :key="action.action"
            :type="action.danger ? 'danger' : 'primary'"
            plain
            @click="executeAction(selectedRow, action)"
          >{{ action.label }}</el-button>
        </div>

        <div class="drawer-section">
          <h3>最近操作记录</h3>
          <div v-loading="historyLoading" class="audit-history">
            <el-timeline v-if="history.length">
              <el-timeline-item v-for="item in history" :key="item.id" :timestamp="formatDate(item.updatedAt)">
                <strong>{{ auditActionText(item) }}</strong>
                <p>{{ auditDetailText(item) }}</p>
              </el-timeline-item>
            </el-timeline>
            <el-empty v-else description="暂无处置记录" :image-size="72" />
          </div>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { ElMessage, ElMessageBox } from "element-plus"
import {
  getAuditHistory,
  getModerationPage,
  getModerationSummary,
  getUploadPreview,
  runAdminAction,
  type AdminResource,
  type ModerationAuditRow,
  type ModerationRow,
  type ModerationSummary,
  type ModerationTab
} from "../api/http"
import { adminActionLabel, adminDateLabel, adminDetailLabel, adminEnumLabel, adminStatusLabel } from "../utils/adminDisplay"

interface TabState {
  keyword: string
  status: string
  itemType: string
  page: number
  pageSize: number
  total: number
  list: ModerationRow[]
  loading: boolean
  loaded: boolean
  error: string
}

interface RowAction {
  label: string
  action: string
  resource: AdminResource
  targetId: string
  status: string
  danger: boolean
  result?: string
}

const route = useRoute()
const router = useRouter()
const summary = ref<ModerationSummary | null>(null)
const detailVisible = ref(false)
const selectedRow = ref<ModerationRow | null>(null)
const history = ref<ModerationAuditRow[]>([])
const historyLoading = ref(false)
const previewUrls = reactive<Record<string, string>>({})

function createState(): TabState {
  return { keyword: "", status: "all", itemType: "all", page: 1, pageSize: 10, total: 0, list: [], loading: false, loaded: false, error: "" }
}

const states = reactive<Record<ModerationTab, TabState>>({
  content: createState(),
  messages: createState(),
  media: createState()
})

function normalizedTab(value: unknown): ModerationTab {
  return value === "messages" || value === "media" ? value : "content"
}

const activeTab = computed<ModerationTab>({
  get: () => normalizedTab(route.query.tab),
  set: (tab) => {
    if (tab !== activeTab.value) router.replace({ path: "/moderation", query: { ...route.query, tab } })
  }
})

const currentState = computed(() => states[activeTab.value])
const statusOptions = computed(() => {
  if (activeTab.value === "content") return [
    { label: "全部状态", value: "all" }, { label: "未处置", value: "active" }, { label: "已处置", value: "handled" }
  ]
  if (activeTab.value === "messages") return [
    { label: "全部状态", value: "all" }, { label: "正常", value: "normal" }, { label: "已隐藏", value: "hidden" }
  ]
  return [
    { label: "全部状态", value: "all" }, { label: "待审核", value: "pending" },
    { label: "已通过", value: "approved" }, { label: "已拒绝", value: "rejected" }
  ]
})
const tabTitle = computed(() => ({ content: "内容巡检记录", messages: "群聊消息记录", media: "媒体审核记录" })[activeTab.value])
const tabDescription = computed(() => ({
  content: "巡检用户资料、行程内容、加入申请和滑后评价。",
  messages: "按发送人、行程或内容搜索群聊消息。",
  media: "查看图片原件、机器审核结论并进行人工复核。"
})[activeTab.value])
const searchPlaceholder = computed(() => ({
  content: "搜索对象、资料或内容",
  messages: "搜索发送人、行程或消息",
  media: "搜索媒体类型、格式或状态"
})[activeTab.value])
const emptyDescription = computed(() => ({ content: "暂无符合条件的内容", messages: "暂无符合条件的消息", media: "暂无符合条件的媒体" })[activeTab.value])

async function loadSummary() {
  try {
    summary.value = await getModerationSummary()
  } catch {
    summary.value = null
  }
}

async function loadData(tab: ModerationTab = activeTab.value) {
  const state = states[tab]
  state.loading = true
  state.error = ""
  try {
    const params: Record<string, unknown> = { page: state.page, pageSize: state.pageSize, keyword: state.keyword }
    if (tab === "content") {
      if (state.status !== "all") params.reviewState = state.status
      if (state.itemType !== "all") params.itemType = state.itemType
    } else if (state.status !== "all") {
      params.status = state.status
    }
    const result = await getModerationPage(tab, params)
    state.list = result.list
    state.total = result.total
    state.loaded = true
    if (tab === "media") await loadMediaPreviews(result.list)
  } catch {
    state.error = "审核列表加载失败，请稍后重试"
  } finally {
    state.loading = false
  }
}

async function refreshCurrent() {
  await Promise.all([loadSummary(), loadData()])
}

function applyFilters() {
  currentState.value.page = 1
  loadData()
}

function resetFilters() {
  const state = currentState.value
  state.keyword = ""
  state.status = "all"
  state.itemType = "all"
  state.page = 1
  loadData()
}

function changePage(page: number) {
  currentState.value.page = page
  loadData()
}

function changePageSize(pageSize: number) {
  currentState.value.pageSize = pageSize
  currentState.value.page = 1
  loadData()
}

function clearPreviewUrls() {
  Object.keys(previewUrls).forEach((id) => {
    URL.revokeObjectURL(previewUrls[id])
    delete previewUrls[id]
  })
}

async function loadMediaPreviews(rows: ModerationRow[]) {
  clearPreviewUrls()
  for (let index = 0; index < rows.length; index += 5) {
    const batch = rows.slice(index, index + 5)
    await Promise.allSettled(batch.map(async (row) => {
      const blob = await getUploadPreview(row.id)
      previewUrls[row.id] = URL.createObjectURL(blob)
    }))
  }
}

function rowTypeLabel(row: ModerationRow) {
  if (activeTab.value === "messages") return `${messageTypeLabel(row.messageType)} · ${row.eventTitle || `行程 #${row.eventId || "-"}`}`
  if (activeTab.value === "media") return `${mediaKindLabel(row.kind)} · 用户 #${row.userId || "-"}`
  return row.type || ({ user: "用户资料", event: "滑雪局", application: "加入申请", review: "滑后评价" } as Record<string, string>)[row.itemType || ""] || "内容"
}

function displayTarget(row: ModerationRow) {
  if (activeTab.value === "media") return `${mediaKindLabel(row.kind)} #${row.id}`
  return row.target || "—"
}

function auditActionText(item: ModerationAuditRow) {
  const rawAction = item.action || String(item.summary || "").split(" · ", 1)[0]
  return adminActionLabel(rawAction)
}

function auditDetailText(item: ModerationAuditRow) {
  const rawSummary = String(item.summary || "")
  const fallbackDetail = rawSummary.includes(" · ") ? rawSummary.slice(rawSummary.indexOf(" · ") + 3) : ""
  const detail = adminDetailLabel(item.detail || fallbackDetail)
  return [`操作人：${item.target || "—"}`, detail].filter(Boolean).join(" · ")
}

function rowActions(row: ModerationRow): RowAction[] {
  const rawId = String(row.id)
  const parts = rawId.split(":", 2)
  const numericId = parts.length === 2 ? parts[1] : rawId
  if (activeTab.value === "messages") {
    return [row.status === "hidden"
      ? { label: "恢复消息", action: "restore_message", resource: "messages", targetId: numericId, status: "normal", danger: false }
      : { label: "隐藏消息", action: "hide_message", resource: "messages", targetId: numericId, status: "hidden", danger: true }]
  }
  if (activeTab.value === "media") {
    if (row.status === "pending") return [
      { label: "通过", action: "approve_upload", resource: "uploads", targetId: numericId, status: "approved", danger: false, result: "人工审核通过" },
      { label: "拒绝", action: "reject_upload", resource: "uploads", targetId: numericId, status: "rejected", danger: true }
    ]
    if (row.status === "approved") return [{ label: "下架媒体", action: "reject_upload", resource: "uploads", targetId: numericId, status: "rejected", danger: true }]
    return [{ label: "恢复媒体", action: "approve_upload", resource: "uploads", targetId: numericId, status: "approved", danger: false, result: "人工复核恢复" }]
  }
  if (parts[0] === "user") return [row.status === "disabled"
    ? { label: "启用用户", action: "enable_user", resource: "users", targetId: numericId, status: "normal", danger: false }
    : { label: "禁用用户", action: "disable_user", resource: "users", targetId: numericId, status: "disabled", danger: true }]
  if (parts[0] === "event") return [row.status === "removed"
    ? { label: "恢复行程", action: "restore_event", resource: "events", targetId: numericId, status: "recruiting", danger: false }
    : { label: "下架行程", action: "delist_event", resource: "events", targetId: numericId, status: "removed", danger: true }]
  if (parts[0] === "application") return row.status === "pending"
    ? [{ label: "拒绝申请", action: "reject_application", resource: "applications", targetId: numericId, status: "rejected", danger: true }]
    : []
  if (parts[0] === "review") return [row.status === "hidden"
    ? { label: "恢复评价", action: "restore_review", resource: "reviews", targetId: numericId, status: "normal", danger: false }
    : { label: "隐藏评价", action: "hide_review", resource: "reviews", targetId: numericId, status: "hidden", danger: true }]
  return []
}

async function executeAction(row: ModerationRow, action: RowAction) {
  let reason = ""
  try {
    if (action.danger) {
      const prompt = await ElMessageBox.prompt(`确认执行“${action.label}”，请输入原因。`, "操作确认", {
        type: "warning",
        inputPlaceholder: "该原因会写入操作审计",
        inputValidator: (value) => !!String(value || "").trim() || "必须填写操作原因"
      })
      reason = prompt.value.trim()
    } else {
      await ElMessageBox.confirm(`确认执行“${action.label}”？`, "操作确认", { type: "info" })
    }
  } catch (error) {
    if (error === "cancel" || error === "close") return
    throw error
  }
  try {
    await runAdminAction(action.resource, action.targetId, {
      action: action.action,
      status: action.status,
      reason,
      result: action.action === "reject_upload" ? reason : action.result
    })
  } catch {
    return
  }
  ElMessage.success(`${action.label}成功`)
  await Promise.all([loadData(), loadSummary()])
  selectedRow.value = currentState.value.list.find((item) => item.id === row.id) || null
  if (detailVisible.value && selectedRow.value) await loadHistory(selectedRow.value)
}

function auditTarget(row: ModerationRow) {
  if (activeTab.value === "messages") return { resource: "messages", id: row.id }
  if (activeTab.value === "media") return { resource: "uploads", id: row.id }
  const [prefix, id] = String(row.id).split(":", 2)
  return { resource: ({ user: "users", event: "events", application: "applications", review: "reviews" } as Record<string, string>)[prefix] || prefix, id }
}

async function loadHistory(row: ModerationRow) {
  historyLoading.value = true
  try {
    const target = auditTarget(row)
    const result = await getAuditHistory(target.resource, target.id)
    history.value = result.list
  } catch {
    history.value = []
  } finally {
    historyLoading.value = false
  }
}

async function showDetail(row: ModerationRow) {
  selectedRow.value = row
  history.value = []
  detailVisible.value = true
  if (activeTab.value === "media" && !previewUrls[row.id]) {
    try {
      const blob = await getUploadPreview(row.id)
      previewUrls[row.id] = URL.createObjectURL(blob)
    } catch {
      // The global HTTP interceptor already presents a useful error.
    }
  }
  await loadHistory(row)
}

function statusType(status: string) {
  if (["hidden", "disabled", "removed", "rejected"].includes(status)) return "danger"
  if (["pending", "processing", "recruiting"].includes(status)) return "warning"
  return "success"
}

function statusLabel(status: string) {
  return adminStatusLabel(status, activeTab.value === "media" ? "uploads" : activeTab.value)
}

function messageTypeLabel(value?: string) {
  return value ? adminEnumLabel(value) : "消息"
}

function mediaKindLabel(value?: string) {
  return value ? adminEnumLabel(value) : "图片"
}

function machineResultLabel(value?: string) {
  return value ? adminEnumLabel(value) : "暂无结果"
}

function formatDate(value?: string) {
  return adminDateLabel(value)
}

watch(activeTab, (tab) => {
  detailVisible.value = false
  selectedRow.value = null
  history.value = []
  if (!states[tab].loaded) loadData(tab)
})

onMounted(() => {
  if (!route.query.tab || normalizedTab(route.query.tab) !== route.query.tab) {
    router.replace({ path: "/moderation", query: { ...route.query, tab: "content" } })
  }
  loadSummary()
  if (!states[activeTab.value].loaded) loadData(activeTab.value)
})

onBeforeUnmount(clearPreviewUrls)
</script>
