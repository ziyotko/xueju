<template>
  <div class="dashboard">
    <div class="page-title">
      <h1>控制台</h1>
      <el-button type="primary" :loading="loading" @click="loadHealth">刷新状态</el-button>
    </div>

    <el-row :gutter="16">
      <el-col :xs="24" :md="8">
        <el-card shadow="never">
          <template #header>API 状态</template>
          <el-tag v-if="health" type="success">已连接</el-tag>
          <el-tag v-else type="info">等待连接</el-tag>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="8">
        <el-card shadow="never">
          <template #header>运行环境</template>
          <span>{{ health?.env || "-" }}</span>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="8">
        <el-card shadow="never">
          <template #header>数据库</template>
          <span>{{ health?.database || "-" }}</span>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="ops-row">
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header>内容审核</template>
          <el-statistic title="待审核" :value="dashboard?.contentReview.pending || 0" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header>举报处理</template>
          <el-statistic title="待处理" :value="dashboard?.reports.pending || 0" />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue"
import { getAdminDashboard, getHealth, type AdminDashboard, type HealthInfo } from "../api/http"

const loading = ref(false)
const health = ref<HealthInfo | null>(null)
const dashboard = ref<AdminDashboard | null>(null)

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

<style scoped>
.ops-row {
  margin-top: 16px;
}
</style>
