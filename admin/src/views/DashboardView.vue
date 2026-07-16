<template>
  <div class="dashboard">
    <section class="dashboard-command-row">
      <div>
        <h1>运营概览</h1>
        <p>今日安全、举报与行程状态</p>
      </div>
      <div class="dashboard-actions">
        <el-button plain @click="openModeration('content')">进入审核中心</el-button>
        <el-button type="primary" :loading="loading" @click="loadHealth">刷新状态</el-button>
      </div>
    </section>

    <section class="metric-grid">
      <div class="metric-card ok">
        <div class="metric-label">接口状态</div>
        <div class="metric-value">{{ health ? "已连接" : "未连接" }}</div>
        <div class="metric-meta">{{ health?.app || "xueju-api" }}</div>
      </div>
      <div class="metric-card">
        <div class="metric-label">数据库</div>
        <div class="metric-value">{{ adminEnumLabel(health?.database) }}</div>
        <div class="metric-meta">运行环境：{{ adminEnumLabel(health?.env) }}</div>
      </div>
      <button type="button" class="metric-card warn metric-button" @click="openModeration('media')">
        <div class="metric-label">待审媒体</div>
        <div class="metric-value">{{ moderation?.media.pending || 0 }}</div>
        <div class="metric-meta">点击进入媒体审核</div>
      </button>
      <button type="button" class="metric-card danger metric-button" @click="router.push('/reports')">
        <div class="metric-label">待处理举报</div>
        <div class="metric-value">{{ dashboard?.reports.pending || 0 }}</div>
        <div class="metric-meta">点击进入举报处理</div>
      </button>
    </section>

    <section class="content-grid">
      <div class="panel">
        <div class="panel-head">
          <div>
            <h2>审核概况</h2>
            <p>聚合内容巡检、群聊巡检和媒体审核的实时结果。</p>
          </div>
          <el-tag type="success" effect="plain">实时读取</el-tag>
        </div>
        <div class="totals-grid">
          <button type="button" @click="openModeration('content')"><strong>{{ moderation?.content.handled || 0 }}</strong><span>已处置内容</span></button>
          <button type="button" @click="openModeration('messages')"><strong>{{ moderation?.messages.hidden || 0 }}</strong><span>已隐藏消息</span></button>
          <button type="button" @click="openModeration('media')"><strong>{{ moderation?.media.approved || 0 }}</strong><span>已通过媒体</span></button>
          <button type="button" @click="router.push('/applications')"><strong>{{ dashboard?.totals?.applications || 0 }}</strong><span>待处理申请</span></button>
        </div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <div>
            <h2>合规边界</h2>
            <p>第一版只做滑雪行程组织，不做交易和非行程社交。</p>
          </div>
        </div>
        <div class="rule-list">
          <div v-for="item in rules" :key="item" class="rule-item">
            <span class="check-dot">✓</span>
            <span>{{ item }}</span>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue"
import { useRouter } from "vue-router"
import { getAdminDashboard, getHealth, getModerationSummary, type AdminDashboard, type HealthInfo, type ModerationSummary, type ModerationTab } from "../api/http"
import { adminEnumLabel } from "../utils/adminDisplay"

const router = useRouter()
const loading = ref(false)
const health = ref<HealthInfo | null>(null)
const dashboard = ref<AdminDashboard | null>(null)
const moderation = ref<ModerationSummary | null>(null)

const rules = [
  "不展示手机号、微信号、二维码",
  "同行交通仅作为行程说明",
  "住宿只保留需求备注",
  "风险内容统一提示用户修改"
]

async function loadHealth() {
  loading.value = true
  try {
    health.value = await getHealth()
    ;[dashboard.value, moderation.value] = await Promise.all([getAdminDashboard(), getModerationSummary()])
  } catch {
    health.value = null
    dashboard.value = null
    moderation.value = null
  } finally {
    loading.value = false
  }
}

function openModeration(tab: ModerationTab) {
  router.push({ path: "/moderation", query: { tab } })
}

onMounted(loadHealth)
</script>
