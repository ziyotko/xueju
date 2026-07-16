<template>
  <div class="admin-list-page">
    <section class="toolbar panel">
      <el-input v-model="keyword" class="search-input" clearable placeholder="搜索对象、内容或状态" @keyup.enter="loadData" />
      <el-select v-model="status" class="status-select" placeholder="状态">
        <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
      <el-button plain @click="loadData">筛选</el-button>
    </section>

    <section class="table-panel panel">
      <div class="panel-head">
        <div>
          <h2>{{ title }}</h2>
          <p>{{ meta.description }} · 当前第 {{ page.page }} 页</p>
        </div>
        <div class="panel-head-actions">
          <el-tag effect="plain">{{ page.total }} 条</el-tag>
          <el-button v-if="resource === 'dicts'" type="primary" @click="openDictDialog()">新增雪场</el-button>
          <el-button :loading="loading" @click="loadData">刷新</el-button>
        </div>
      </div>

      <div class="table-scroll-area">
        <el-table :data="page.list" v-loading="loading" class="soft-table" height="100%">
        <el-table-column prop="id" label="编号" width="88" />
        <el-table-column label="对象" min-width="190">
          <template #default="{ row }">
            <div class="object-cell">
              <div class="object-avatar">{{ row.initial || "-" }}</div>
              <div>
                <div class="object-title">{{ displayRowTarget(row) }}</div>
                <div class="object-subtitle">{{ displayRowType(row) }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="内容摘要" min-width="300" show-overflow-tooltip>
          <template #default="{ row }">{{ displayRowSummary(row) }}</template>
        </el-table-column>
        <el-table-column label="风险" width="120">
          <template #default="{ row }">
            <el-tag :type="riskType(row.risk)" effect="light">{{ adminRiskLabel(row.risk) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="130">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" effect="plain">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="resource === 'users'" label="手机号认证" width="170">
          <template #default="{ row }">
            <div>{{ verificationLabel(row.verificationStatus) }}</div>
            <div class="object-subtitle">{{ row.phoneMasked || "未绑定" }}</div>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="172">
          <template #default="{ row }">{{ formatDate(row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" :width="actionColumnWidth" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="showDetail(row)">查看</el-button>
            <el-button v-if="resource === 'dicts'" link type="primary" @click="openDictDialog(row)">编辑</el-button>
            <el-button v-if="primaryAction(row).action !== 'noop'" link :type="primaryAction(row).danger ? 'danger' : 'primary'" @click="executeAction(row)">
              {{ primaryAction(row).text }}
            </el-button>
			<el-button v-if="resource === 'users'" link type="primary" @click="showVerifications(row)">认证记录</el-button>
			<el-button v-if="resource === 'users' && row.verificationStatus === 'verified'" link type="warning" @click="changeVerification(row, false)">要求重认证</el-button>
			<el-button v-if="resource === 'users' && row.verificationStatus !== 'revoked'" link type="danger" @click="changeVerification(row, true)">撤销认证</el-button>
			<el-button v-if="resource === 'uploads' && row.status === 'pending'" link type="danger" @click="rejectUpload(row)">拒绝</el-button>
          </template>
        </el-table-column>
        <template #empty>
          <el-empty description="暂无记录" />
        </template>
        </el-table>
      </div>

      <div class="pagination-row">
        <el-pagination
          v-model:current-page="pageNumber"
          v-model:page-size="pageSize"
          layout="prev, pager, next, sizes"
          :total="page.total"
          :page-sizes="[10, 20, 50]"
          @change="loadData"
        />
      </div>
    </section>

	<el-dialog v-model="detailVisible" title="记录详情" width="680px">
	  <el-descriptions v-if="selectedRow" :column="1" border class="record-detail">
		<el-descriptions-item v-for="([key, value]) in detailEntries" :key="key" :label="adminFieldLabel(key)">
          <div v-if="detailImageUrl(key, value)" class="detail-image">
            <el-image
              :src="detailImageUrl(key, value)"
              :preview-src-list="[detailImageUrl(key, value)]"
              fit="cover"
              preview-teleported
            >
              <template #error><div class="detail-image-error">图片加载失败</div></template>
            </el-image>
            <a :href="detailImageUrl(key, value)" target="_blank" rel="noopener noreferrer">查看原图</a>
          </div>
          <div v-else-if="isAdminDetailObject(value)" class="detail-object">
            <div v-for="(nestedValue, nestedKey) in value" :key="String(nestedKey)" class="detail-object-row">
              <span>{{ adminFieldLabel(String(nestedKey)) }}</span>
              <strong>{{ adminValueLabel(String(nestedKey), nestedValue, resource) }}</strong>
            </div>
          </div>
          <span v-else>{{ adminValueLabel(key, value, resource) }}</span>
		</el-descriptions-item>
	  </el-descriptions>
    </el-dialog>

	<el-dialog v-model="verificationVisible" title="手机号认证记录" width="760px">
	  <el-table :data="verificationRecords">
		<el-table-column label="状态" width="150">
          <template #default="{ row }">{{ adminVerificationLabel(row.status) }}</template>
        </el-table-column>
		<el-table-column prop="phoneMasked" label="脱敏手机号" width="150" />
		<el-table-column label="认证方式" width="140">
          <template #default="{ row }">{{ adminEnumLabel(row.method) }}</template>
        </el-table-column>
		<el-table-column label="服务商" width="150">
          <template #default="{ row }">{{ adminEnumLabel(row.provider) }}</template>
        </el-table-column>
		<el-table-column label="认证时间" min-width="190">
          <template #default="{ row }">{{ adminDateLabel(row.verifiedAt) }}</template>
        </el-table-column>
	  </el-table>
	</el-dialog>

    <el-dialog v-model="dictVisible" :title="dictForm.id ? '编辑雪场' : '新增雪场'" width="520px">
      <el-form label-width="88px">
        <el-form-item label="雪场名称"><el-input v-model="dictForm.name" /></el-form-item>
        <el-form-item label="城市"><el-input v-model="dictForm.city" /></el-form-item>
        <el-form-item label="省份"><el-input v-model="dictForm.province" /></el-form-item>
        <el-form-item label="雪场图片" required>
          <div class="resort-image-field">
            <div v-if="dictImagePreview" class="resort-image-preview">
              <img :src="dictImagePreview" alt="雪场图片预览" />
              <el-button class="resort-image-remove" type="danger" size="small" @click="removeDictImage">移除</el-button>
            </div>
            <el-upload
              accept="image/jpeg,image/png"
              :auto-upload="false"
              :show-file-list="false"
              :on-change="handleDictImageChange"
            >
              <el-button>{{ dictImagePreview ? "更换图片" : "选择图片" }}</el-button>
            </el-upload>
            <div class="resort-image-tip">支持 JPG、PNG，大小不超过 5MB</div>
          </div>
        </el-form-item>
        <el-form-item label="排序"><el-input-number v-model="dictForm.sort" :min="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dictVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingDict" @click="submitDict">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { ElMessage, ElMessageBox } from "element-plus"
import type { UploadFile } from "element-plus"
import { getAdminPageWithParams, getUserVerifications, runAdminAction, saveDict, uploadResortImage, type AdminResource, type PageResult } from "../api/http"
import {
  adminActionLabel,
  adminDateLabel,
  adminDetailLabel,
  adminEnumLabel,
  adminFieldLabel,
  adminResourceLabel,
  adminRiskLabel,
  adminStatusLabel,
  adminValueLabel,
  adminVerificationLabel,
  isAdminDetailObject
} from "../utils/adminDisplay"

type Row = Record<string, any>

const props = defineProps<{
  resource: AdminResource
  title: string
  actionText: string
}>()

const loading = ref(false)
const keyword = ref("")
const status = ref("all")
const pageNumber = ref(1)
const pageSize = ref(10)
const detailVisible = ref(false)
const selectedRow = ref<Row | null>(null)
const verificationVisible = ref(false)
const verificationRecords = ref<Row[]>([])
const page = ref<PageResult<Row>>({ list: [], page: 1, pageSize: 10, total: 0 })
const dictVisible = ref(false)
const savingDict = ref(false)
const dictForm = ref({ id: 0, name: "", city: "", province: "", imageUrl: "", sort: 0, status: "normal" })
const dictImageFile = ref<File | null>(null)
const dictImagePreview = ref("")
const detailOrder = ["id", "target", "type", "summary", "status", "risk", "resource", "targetId", "action", "beforeStatus", "afterStatus", "detail", "updatedAt"]
const detailEntries = computed(() => Object.entries(selectedRow.value || {})
  .filter(([key]) => key !== "initial" && key !== "previewPath")
  .sort(([left], [right]) => {
    const leftIndex = detailOrder.indexOf(left)
    const rightIndex = detailOrder.indexOf(right)
    return (leftIndex < 0 ? detailOrder.length : leftIndex) - (rightIndex < 0 ? detailOrder.length : rightIndex)
  }))

const statusOptionMap: Partial<Record<AdminResource, Array<{ label: string; value: string }>>> = {
  users: [{ label: "正常", value: "normal" }, { label: "已禁用", value: "disabled" }],
  events: [{ label: "招募中", value: "recruiting" }, { label: "已满员", value: "full" }, { label: "已结束", value: "finished" }, { label: "已取消", value: "cancelled" }, { label: "已下架", value: "removed" }],
  applications: [{ label: "待处理", value: "pending" }, { label: "已通过", value: "approved" }, { label: "已拒绝", value: "rejected" }, { label: "已取消", value: "cancelled" }],
  reports: [{ label: "待处理", value: "pending" }, { label: "处理中", value: "processing" }, { label: "已处理", value: "resolved" }, { label: "已拒绝", value: "rejected" }],
  reviews: [{ label: "正常", value: "normal" }, { label: "已隐藏", value: "hidden" }],
  messages: [{ label: "正常", value: "normal" }, { label: "已隐藏", value: "hidden" }],
  dicts: [{ label: "正常", value: "normal" }, { label: "已禁用", value: "disabled" }],
  uploads: [{ label: "待审核", value: "pending" }, { label: "已通过", value: "approved" }, { label: "已拒绝", value: "rejected" }]
}

const statusOptions = computed(() => [
  { label: "全部", value: "all" },
  ...(statusOptionMap[props.resource] || [
    { label: "正常", value: "normal" },
    { label: "待处理", value: "pending" },
    { label: "已处理", value: "resolved" },
    { label: "已拒绝", value: "rejected" },
    { label: "已隐藏", value: "hidden" },
    { label: "已禁用", value: "disabled" },
    { label: "已下架", value: "removed" }
  ])
])

const metaMap: Record<AdminResource, { kicker: string; description: string; tableTitle: string }> = {
  users: { kicker: "账号与信用", description: "查看用户状态、信用信息，必要时禁用或恢复账号。", tableTitle: "用户记录" },
  events: { kicker: "行程治理", description: "处理不合规滑雪局内容，对异常行程执行下架或恢复。", tableTitle: "滑雪局记录" },
  applications: { kicker: "加入审核", description: "查看用户提交的加入申请，按状态追踪处理情况。", tableTitle: "申请记录" },
  reports: { kicker: "风险响应", description: "集中处理用户举报，形成可追踪的运营处置记录。", tableTitle: "举报记录" },
  reviews: { kicker: "评价与信用", description: "维护滑后评价秩序，隐藏恶意或异常评价。", tableTitle: "评价记录" },
  messages: { kicker: "群聊管理", description: "查看局内群聊消息，必要时隐藏不适宜内容。", tableTitle: "消息记录" },
  "content-reviews": { kicker: "内容安全", description: "复核昵称、备注、群聊、评价与举报内容。", tableTitle: "内容审核记录" },
  dicts: { kicker: "基础字典", description: "管理雪场、城市与标签等基础运营数据。", tableTitle: "字典记录" }
	,uploads: { kicker: "图片安全", description: "审核头像和行程图片，只有通过后才能公开访问。", tableTitle: "待审媒体" }
	,"audit-logs": { kicker: "操作留痕", description: "查看管理员对用户、内容和举报执行的关键操作。", tableTitle: "审计日志" }
}

const meta = computed(() => metaMap[props.resource])
const actionColumnWidth = computed(() => props.resource === "users" ? 420 : 260)

async function loadData() {
  loading.value = true
  try {
    page.value = await getAdminPageWithParams(props.resource, {
      page: pageNumber.value,
      pageSize: pageSize.value,
      keyword: keyword.value,
      status: status.value
    })
  } finally {
    loading.value = false
  }
}

function showDetail(row: Row) {
  selectedRow.value = row
  detailVisible.value = true
}

function primaryAction(row: Row) {
  if (props.resource === "users") {
    return row.status === "disabled"
      ? { text: "启用用户", action: "enable_user", status: "normal", danger: false }
      : { text: props.actionText, action: "disable_user", status: "disabled", danger: true }
  }
  if (props.resource === "events") {
    return row.status === "removed"
      ? { text: "恢复活动", action: "restore_event", status: "recruiting", danger: false }
      : { text: props.actionText, action: "delist_event", status: "removed", danger: true }
  }
  if (props.resource === "reports") {
    return row.status === "resolved"
      ? { text: "已处理", action: "noop", danger: false }
      : { text: props.actionText, action: "resolve_report", status: "resolved", result: "运营已处理", danger: false }
  }
  if (props.resource === "reviews") {
    return row.status === "hidden"
      ? { text: "恢复评价", action: "restore_review", status: "normal", danger: false }
      : { text: props.actionText, action: "hide_review", status: "hidden", danger: true }
  }
  if (props.resource === "messages") {
    return row.status === "hidden"
      ? { text: "恢复消息", action: "restore_message", status: "normal", danger: false }
      : { text: props.actionText, action: "hide_message", status: "hidden", danger: true }
  }
  if (props.resource === "content-reviews") {
    if (["disabled", "removed", "rejected", "hidden", "resolved"].includes(row.status)) {
      return { text: "已处置", action: "noop", danger: false }
    }
    const kind = String(row.id || "").split(":", 1)[0]
    const labels: Record<string, string> = {
      user: "禁用用户", event: "下架行程", application: "拒绝申请",
      message: "隐藏消息", review: "隐藏评价", report: "处理举报"
    }
    return { text: labels[kind] || props.actionText, action: "review_content", status: "hidden", danger: true }
  }
  if (props.resource === "dicts") {
    return row.status === "disabled"
      ? { text: "启用雪场", action: "enable_dict", status: "normal", danger: false }
      : { text: props.actionText, action: "disable_dict", status: "disabled", danger: true }
  }
	if (props.resource === "uploads") {
	  return row.status === "approved"
		? { text: "已通过", action: "noop", danger: false }
		: { text: "通过审核", action: "approve_upload", status: "approved", result: "人工审核通过", danger: false }
	}
	if (props.resource === "audit-logs") return { text: "查看记录", action: "noop", danger: false }
  if (props.resource === "applications") {
    return row.status === "pending"
      ? { text: "拒绝申请", action: "reject_application", status: "rejected", danger: true }
      : { text: "查看记录", action: "noop", danger: false }
  }
  return { text: "查看记录", action: "noop", danger: false }
}

function displayRowTarget(row: Row) {
  if (props.resource === "reports" && row.targetType) return `${adminResourceLabel(row.targetType)} #${row.targetId}`
  if (props.resource === "uploads" && row.kind) return `${adminEnumLabel(row.kind)} #${row.id}`
  return row.target || "—"
}

function displayRowType(row: Row) {
  if (props.resource === "audit-logs") {
    const matched = String(row.type || "").match(/^([a-z-]+)\s+#(.+)$/i)
    const resource = row.resource || matched?.[1]
    const targetId = row.targetId || matched?.[2]
    return resource ? `${adminResourceLabel(resource)}${targetId ? ` · 对象 #${targetId}` : ""}` : "操作记录"
  }
  return adminEnumLabel(row.type)
}

function displayRowSummary(row: Row) {
  if (props.resource !== "audit-logs") return row.summary ? adminDetailLabel(row.summary) : "—"
  const rawSummary = String(row.summary || "")
  const [rawAction, ...rawDetail] = rawSummary.split(" · ")
  const action = row.action || rawAction
  const detail = row.detail || rawDetail.join(" · ")
  return [adminActionLabel(action), adminDetailLabel(detail)].filter((item) => item && item !== "—").join(" · ") || "—"
}

async function rejectUpload(row: Row) {
	const result = await ElMessageBox.prompt("请输入拒绝原因", "拒绝媒体", {
    inputPlaceholder: "例如：图片包含不适宜内容",
    inputValidator: (value) => !!String(value || "").trim() || "必须填写拒绝原因"
  })
	await runAdminAction("uploads", row.id, { action: "reject_upload", status: "rejected", result: result.value })
	ElMessage.success("已拒绝")
	await loadData()
}

async function executeAction(row: Row) {
  const action = primaryAction(row)
  if (action.action === "noop") {
    showDetail(row)
    return
  }
  if (props.resource === "reports") {
    const result = await ElMessageBox.prompt("请输入处理结果", "举报处理", {
      inputValue: row.result || "运营已处理",
      inputPlaceholder: "例如：已核实并下架相关内容"
    })
	const linkedAction = await ElMessageBox.confirm("是否同时处置被举报对象？用户将被禁用，行程将下架，消息或评价将隐藏。", "联动处置", {
	  confirmButtonText: "同时处置",
	  cancelButtonText: "仅处理举报",
	  type: "warning"
	}).then(() => true).catch(() => false)
	await runAdminAction(props.resource, row.id, { ...action, result: result.value, linkedAction })
    ElMessage.success("操作成功")
    await loadData()
    return
  }
  let reason = ""
  if (action.danger) {
    const result = await ElMessageBox.prompt(`确认执行“${action.text}”，请输入原因。`, "操作确认", {
      inputPlaceholder: "该原因会写入审计日志",
      inputValidator: (value) => !!String(value || "").trim() || "必须填写操作原因",
      type: "warning"
    })
    reason = result.value
  } else {
    await ElMessageBox.confirm(`确认执行“${action.text}”？`, "操作确认", { type: "info" })
  }
  await runAdminAction(props.resource, row.id, { ...action, reason })
  ElMessage.success("操作成功")
  await loadData()
}

async function showVerifications(row: Row) {
  verificationRecords.value = await getUserVerifications(row.id) as Row[]
  verificationVisible.value = true
}

async function changeVerification(row: Row, revoke: boolean) {
  const result = await ElMessageBox.prompt("请输入处理原因", revoke ? "撤销手机号认证" : "要求重新认证", {
    inputPlaceholder: "该原因会写入审计日志",
    inputValidator: (value) => !!String(value || "").trim() || "必须填写原因"
  })
  await runAdminAction("users", row.id, { action: revoke ? "revoke_phone_verification" : "require_phone_reverification", reason: result.value })
  ElMessage.success("手机号认证状态已更新")
  await loadData()
}

function openDictDialog(row?: Row) {
  const extra = row?.extra || {}
  dictForm.value = {
    id: Number(row?.id || 0),
    name: row?.target || "",
    city: extra.city || "",
    province: extra.province || "",
    imageUrl: extra.imageUrl || "",
    sort: Number(extra.sort || 0),
    status: row?.status || "normal"
  }
  clearDictImageFile()
  dictImagePreview.value = dictForm.value.imageUrl
  dictVisible.value = true
}

function clearDictImageFile() {
  if (dictImagePreview.value.startsWith("blob:")) URL.revokeObjectURL(dictImagePreview.value)
  dictImageFile.value = null
}

function handleDictImageChange(uploadFile: UploadFile) {
  const file = uploadFile.raw
  if (!file) return
  if (!["image/jpeg", "image/png"].includes(file.type)) {
    ElMessage.warning("仅支持 JPG、PNG 图片")
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    ElMessage.warning("图片不能超过 5MB")
    return
  }
  clearDictImageFile()
  dictImageFile.value = file
  dictImagePreview.value = URL.createObjectURL(file)
}

function removeDictImage() {
  clearDictImageFile()
  dictImagePreview.value = ""
  dictForm.value.imageUrl = ""
}

async function submitDict() {
  if (!dictForm.value.name.trim() || !dictForm.value.city.trim()) {
    ElMessage.warning("请填写雪场名称和城市")
    return
  }
  if (!dictImageFile.value && !dictForm.value.imageUrl) {
    ElMessage.warning("请上传雪场图片")
    return
  }
  savingDict.value = true
  try {
    if (dictImageFile.value) {
      const uploaded = await uploadResortImage(dictImageFile.value)
      dictForm.value.imageUrl = uploaded.url
    }
    await saveDict(dictForm.value, dictForm.value.id || undefined)
    ElMessage.success("保存成功")
    clearDictImageFile()
    dictVisible.value = false
    await loadData()
  } finally {
    savingDict.value = false
  }
}

function riskType(risk: string) {
  const label = adminRiskLabel(risk)
  if (label === "高") return "danger"
  if (label === "中") return "warning"
  return "success"
}

function statusType(value: string) {
  if (["pending", "processing", "recruiting"].includes(value)) return "warning"
  if (["hidden", "disabled", "removed", "rejected"].includes(value)) return "danger"
  return "success"
}

function statusLabel(value: string) {
  return adminStatusLabel(value, props.resource)
}

function verificationLabel(value: string) {
  return adminVerificationLabel(value)
}

function formatDate(value?: string) {
  return adminDateLabel(value)
}

function detailImageUrl(key: string, value: unknown) {
  if (typeof value !== "string" || !value.trim()) return ""
  const imageFields = ["image", "imageUrl", "avatarUrl", "publicUrl"]
  if (!imageFields.includes(key) && !(props.resource === "uploads" && key === "summary")) return ""
  const url = value.trim()
  if (!/^(https?:\/\/|\/uploads\/)/i.test(url)) return ""
  if (!url.startsWith("/uploads/")) return url
  const apiBase = String(import.meta.env.VITE_API_BASE_URL || "")
  if (/^https?:\/\//i.test(apiBase)) {
    try {
      return new URL(url, apiBase).toString()
    } catch {
      return url
    }
  }
  return url
}

watch(() => props.resource, () => {
  pageNumber.value = 1
  status.value = "all"
  keyword.value = ""
  loadData()
})
onMounted(loadData)
</script>

<style scoped>
.resort-image-field {
  display: grid;
  gap: 10px;
  width: 100%;
}

.resort-image-preview {
  position: relative;
  width: 240px;
  height: 140px;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  background: var(--el-fill-color-light);
}

.resort-image-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.resort-image-remove {
  position: absolute;
  right: 8px;
  bottom: 8px;
}

.resort-image-tip {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
}

.detail-image {
  display: grid;
  justify-items: start;
  gap: 8px;
}

.detail-image :deep(.el-image) {
  width: min(360px, 100%);
  height: 210px;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  background: var(--el-fill-color-light);
  cursor: zoom-in;
}

.detail-image-error {
  display: grid;
  width: 100%;
  height: 100%;
  place-items: center;
  color: var(--el-text-color-secondary);
}

.detail-image a {
  color: var(--el-color-primary);
  font-size: 13px;
  text-decoration: none;
}
</style>
