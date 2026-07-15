<template>
  <main class="login-page">
    <section class="login-panel">
      <div class="brand large">
        <div class="brand-mark">雪</div>
        <div>
          <div class="brand-name">雪局后台</div>
          <div class="brand-subtitle">运营管理登录</div>
        </div>
      </div>

      <el-form class="login-form" @submit.prevent="submit">
        <el-form-item>
          <el-input v-model="username" size="large" placeholder="账号" autocomplete="username" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="password" size="large" placeholder="密码" type="password" autocomplete="current-password" show-password />
        </el-form-item>
        <el-button class="login-button" type="primary" size="large" :loading="loading" @click="submit">登录</el-button>
      </el-form>
    </section>
  </main>
</template>

<script setup lang="ts">
import { ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import { ElMessage } from "element-plus"
import { loginAdmin } from "../api/http"

const route = useRoute()
const router = useRouter()
const username = ref("")
const password = ref("")
const loading = ref(false)

async function submit() {
  if (!username.value.trim() || !password.value) {
    ElMessage.warning("请输入账号和密码")
    return
  }
  loading.value = true
  try {
    const data = await loginAdmin({ username: username.value.trim(), password: password.value })
    localStorage.setItem("xueju_admin_token", data.token)
    localStorage.setItem("xueju_admin_user", JSON.stringify(data.user))
    await router.replace((route.query.redirect as string) || "/")
  } finally {
    loading.value = false
  }
}
</script>
