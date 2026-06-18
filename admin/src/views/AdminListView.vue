<template>
  <div>
    <div class="page-title">
      <h1>{{ title }}</h1>
      <el-button type="primary" :loading="loading" @click="loadData">刷新</el-button>
    </div>

    <el-alert
      class="notice"
      type="info"
      :closable="false"
      show-icon
      title="当前为 MVP 合规管理入口，后续接入数据库后展示真实记录。"
    />

    <el-table :data="page.list" border>
      <el-table-column prop="id" label="ID" width="100" />
      <el-table-column prop="target" label="对象" />
      <el-table-column prop="status" label="状态" width="140" />
      <el-table-column label="操作" width="180">
        <template #default>
          <el-button type="danger" plain disabled>{{ actionText }}</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty description="暂无待处理记录" />
      </template>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue"
import { getAdminPage, type AdminResource, type PageResult } from "../api/http"

const props = defineProps<{
  resource: AdminResource
  title: string
  actionText: string
}>()

const loading = ref(false)
const page = ref<PageResult<Record<string, unknown>>>({
  list: [],
  page: 1,
  pageSize: 10,
  total: 0
})

async function loadData() {
  loading.value = true
  try {
    page.value = await getAdminPage(props.resource)
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.notice {
  margin-bottom: 16px;
}
</style>
