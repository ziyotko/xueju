<template>
  <div class="dashboard">
    <section class="page-hero">
      <div>
        <p class="eyebrow">运营概览</p>
        <h1>今日安全与行程状态</h1>
        <p class="hero-copy">关注内容审核、举报处理和行程风险，让小程序保持清晰、可审计、可运营。</p>
      </div>
      <el-button type="primary" size="large" :loading="loading" @click="loadHealth">刷新状态</el-button>
    </section>

    <section class="metric-grid">
      <div class="metric-card ok">
        <div class="metric-label">API 状态</div>
        <div class="metric-value">{{ health ? "已连接" : "未连接" }}</div>
        <div class="metric-meta">{{ health?.app || "xueju-api" }}</div>
      </div>
      <div class="metric-card">
        <div class="metric-label">数据库</div>
        <div class="metric-value">{{ health?.database || "-" }}</div>
        <div class="metric-meta">运行环境：{{ health?.env || "-" }}</div>
      </div>
      <div class="metric-card warn">
        <div class="metric-label">待处理申请</div>
        <div class="metric-value">{{ dashboard?.totals?.applications || 0 }}</div>
        <div class="metric-meta">发起人审核与运营观察</div>
      </div>
      <div class="metric-card danger">
        <div class="metric-label">待处理举报</div>
        <div class="metric-value">{{ dashboard?.reports.pending || 0 }}</div>
        <div class="metric-meta">用户、行程、消息、评价</div>
      </div>
    </section>

    <section class="content-grid">
      <div class="panel">
        <div class="panel-head">
          <div>
            <h2>运营数据</h2>
            <p>这些数字来自真实 API，随数据库内容变化。</p>
          </div>
          <el-tag type="success" effect="plain">实时读取</el-tag>
        </div>
        <div class="totals-grid">
          <div><strong>{{ dashboard?.totals?.users || 0 }}</strong><span>用户</span></div>
          <div><strong>{{ dashboard?.totals?.events || 0 }}</strong><span>滑雪局</span></div>
          <div><strong>{{ dashboard?.totals?.reviews || 0 }}</strong><span>正常评价</span></div>
          <div><strong>{{ dashboard?.reports.handled || 0 }}</strong><span>已处理举报</span></div>
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
import { getAdminDashboard, getHealth, type AdminDashboard, type HealthInfo } from "../api/http"

const loading = ref(false)
const health = ref<HealthInfo | null>(null)
const dashboard = ref<AdminDashboard | null>(null)

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
    dashboard.value = await getAdminDashboard()
  } catch {
    health.value = null
    dashboard.value = null
  } finally {
    loading.value = false
  }
}

onMounted(loadHealth)
</script>
