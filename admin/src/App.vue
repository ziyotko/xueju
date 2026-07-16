<template>
  <el-config-provider :locale="zhCn">
    <router-view v-if="isLoginPage" />
    <el-container v-else class="layout">
      <el-aside width="248px" class="sidebar">
        <div class="brand">
          <div class="brand-mark">雪</div>
          <div>
            <div class="brand-name">雪局后台</div>
            <div class="brand-subtitle">行程组局运营台</div>
          </div>
        </div>

        <el-menu
          router
          :default-active="route.path"
          background-color="transparent"
          text-color="#9aa6bd"
          active-text-color="#ffffff"
        >
          <el-menu-item index="/">
            <el-icon><HomeFilled /></el-icon>
            <span>控制台</span>
          </el-menu-item>
          <el-menu-item-group title="运营管理">
            <el-menu-item index="/users"><el-icon><User /></el-icon><span>用户管理</span></el-menu-item>
            <el-menu-item index="/events"><el-icon><Calendar /></el-icon><span>滑雪局</span></el-menu-item>
            <el-menu-item index="/applications"><el-icon><Tickets /></el-icon><span>加入申请</span></el-menu-item>
          </el-menu-item-group>
          <el-menu-item-group title="风控治理">
            <el-menu-item index="/moderation"><el-icon><View /></el-icon><span>审核中心</span></el-menu-item>
            <el-menu-item index="/reports"><el-icon><Warning /></el-icon><span>举报处理</span></el-menu-item>
            <el-menu-item index="/reviews"><el-icon><Star /></el-icon><span>评价管理</span></el-menu-item>
          </el-menu-item-group>
          <el-menu-item-group title="系统设置">
            <el-menu-item index="/dicts"><el-icon><Collection /></el-icon><span>字典管理</span></el-menu-item>
            <el-menu-item index="/audit-logs"><el-icon><Document /></el-icon><span>操作审计</span></el-menu-item>
          </el-menu-item-group>
        </el-menu>
      </el-aside>

      <el-container>
        <el-header class="topbar">
          <div>
            <div class="topbar-title">滑雪行程组局工具管理端</div>
            <div class="topbar-subtitle">内容安全、举报处理与基础运营</div>
          </div>
          <div class="topbar-actions">
            <el-tag effect="plain" type="success">接口已接入</el-tag>
            <el-button round @click="openDocs">运营指南</el-button>
            <el-button round @click="logout">退出</el-button>
          </div>
        </el-header>

        <el-main>
          <router-view />
        </el-main>
      </el-container>
    </el-container>
  </el-config-provider>
</template>

<script setup lang="ts">
import { Calendar, Collection, Document, HomeFilled, Star, Tickets, User, View, Warning } from "@element-plus/icons-vue"
import zhCn from "element-plus/es/locale/lang/zh-cn"
import { computed } from "vue"
import { useRoute, useRouter } from "vue-router"

const route = useRoute()
const router = useRouter()
const isLoginPage = computed(() => route.path === "/login")

function openDocs() {
  window.open("https://developers.weixin.qq.com/miniprogram/product/", "_blank")
}

function logout() {
  localStorage.removeItem("xueju_admin_token")
  localStorage.removeItem("xueju_admin_user")
  router.replace("/login")
}
</script>
