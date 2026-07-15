<template>
  <div>
    <section class="page-hero compact">
      <div>
        <p class="eyebrow">{{ meta.kicker }}</p>
        <h1>{{ title }}</h1>
        <p class="hero-copy">{{ meta.description }}</p>
      </div>
      <div class="hero-actions">
        <el-button v-if="resource === 'dicts'" type="primary" @click="openDictDialog()">新增雪场</el-button>
        <el-button :loading="loading" @click="loadData">刷新</el-button>
      </div>
    </section>

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
          <h2>{{ meta.tableTitle }}</h2>
          <p>共 {{ page.total }} 条记录，当前展示第 {{ page.page }} 页。</p>
        </div>
        <el-tag effect="plain">{{ page.total }} 条</el-tag>
      </div>

      <el-table :data="page.list" v-loading="loading" class="soft-table" height="520">
        <el-table-column prop="id" label="ID" width="88" />
        <el-table-column label="对象" min-width="190">
          <template #default="{ row }">
            <div class="object-cell">
              <div class="object-avatar">{{ row.initial || "-" }}</div>
              <div>
                <div class="object-title">{{ row.target }}</div>
                <div class="object-subtitle">{{ row.type }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="summary" label="内容摘要" min-width="300" show-overflow-tooltip />
        <el-table-column label="风险" width="120">
          <template #default="{ row }">
            <el-tag :type="riskType(row.risk)" effect="light">{{ row.risk }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="130">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" effect="plain">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updatedAt" label="更新时间" width="150" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="showDetail(row)">查看</el-button>
            <el-button v-if="resource === 'dicts'" link type="primary" @click="openDictDialog(row)">编辑</el-button>
            <el-button link :type="primaryAction(row).danger ? 'danger' : 'primary'" @click="executeAction(row)">
              {{ primaryAction(row).text }}
            </el-button>
			<el-button v-if="resource === 'uploads' && row.status === 'pending'" link type="danger" @click="rejectUpload(row)">拒绝</el-button>
          </template>
        </el-table-column>
        <template #empty>
          <el-empty description="暂无记录" />
        </template>
      </el-table>

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

	<el-dialog v-model="detailVisible" title="记录详情" width="640px">
	  <el-descriptions v-if="selectedRow" :column="1" border>
		<el-descriptions-item v-for="(value, key) in selectedRow" :key="key" :label="String(key)">
		  {{ typeof value === 'object' ? JSON.stringify(value, null, 2) : value }}
		</el-descriptions-item>
	  </el-descriptions>
    </el-dialog>

    <el-dialog v-model="dictVisible" :title="dictForm.id ? '编辑雪场' : '新增雪场'" width="520px">
      <el-form label-width="88px">
        <el-form-item label="雪场名称"><el-input v-model="dictForm.name" /></el-form-item>
        <el-form-item label="城市"><el-input v-model="dictForm.city" /></el-form-item>
        <el-form-item label="省份"><el-input v-model="dictForm.province" /></el-form-item>
        <el-form-item label="图片地址"><el-input v-model="dictForm.imageUrl" /></el-form-item>
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
import { getAdminPageWithParams, runAdminAction, saveDict, type AdminResource, type PageResult } from "../api/http"

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
const page = ref<PageResult<Row>>({ list: [], page: 1, pageSize: 10, total: 0 })
const dictVisible = ref(false)
const savingDict = ref(false)
const dictForm = ref({ id: 0, name: "", city: "", province: "", imageUrl: "", sort: 0, status: "normal" })

const statusOptions = [
  { label: "全部", value: "all" },
  { label: "正常/招募中", value: "normal" },
  { label: "待处理", value: "pending" },
  { label: "处理中", value: "processing" },
  { label: "已处理", value: "resolved" },
  { label: "已拒绝", value: "rejected" },
  { label: "已隐藏", value: "hidden" },
  { label: "已禁用/下架", value: "disabled" },
  { label: "已移除", value: "removed" }
]

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
    return { text: props.actionText, action: "resolve_report", status: "resolved", result: "运营已处理", danger: false }
  }
  if (props.resource === "reviews") {
    return row.status === "hidden"
      ? { text: "恢复评价", action: "restore_review", status: "normal", danger: false }
      : { text: props.actionText, action: "hide_review", status: "hidden", danger: true }
  }
  if (props.resource === "messages" || props.resource === "content-reviews") {
    return { text: props.actionText, action: "hide_message", status: "hidden", danger: true }
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
  return { text: "查看记录", action: "noop", danger: false }
}

async function rejectUpload(row: Row) {
	const result = await ElMessageBox.prompt("请输入拒绝原因", "拒绝媒体", { inputPlaceholder: "例如：图片包含不适宜内容" })
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
  await ElMessageBox.confirm(`确认执行“${action.text}”？`, "操作确认", { type: action.danger ? "warning" : "info" })
  await runAdminAction(props.resource, row.id, action)
  ElMessage.success("操作成功")
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
  dictVisible.value = true
}

async function submitDict() {
  savingDict.value = true
  try {
    await saveDict(dictForm.value, dictForm.value.id || undefined)
    ElMessage.success("保存成功")
    dictVisible.value = false
    await loadData()
  } finally {
    savingDict.value = false
  }
}

function riskType(risk: string) {
  if (risk === "高") return "danger"
  if (risk === "中") return "warning"
  return "success"
}

function statusType(value: string) {
  if (["pending", "processing", "recruiting"].includes(value)) return "warning"
  if (["hidden", "disabled", "removed", "rejected"].includes(value)) return "danger"
  return "success"
}

function statusLabel(value: string) {
  const labels: Record<string, string> = {
    normal: "正常",
    disabled: "已禁用",
    recruiting: "招募中",
    full: "已满员",
    finished: "已结束",
    cancelled: "已取消",
    removed: "已下架",
    pending: "待处理",
    processing: "处理中",
    resolved: "已处理",
    rejected: "已拒绝",
    hidden: "已隐藏"
  }
  return labels[value] || value
}

watch(() => props.resource, () => {
  pageNumber.value = 1
  status.value = "all"
  keyword.value = ""
  loadData()
})
onMounted(loadData)
</script>
