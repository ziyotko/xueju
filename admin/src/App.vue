<template>
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
        <el-menu-item index="/users">
          <el-icon><User /></el-icon>
          <span>用户管理</span>
        </el-menu-item>
        <el-menu-item index="/events">
          <el-icon><Calendar /></el-icon>
          <span>滑雪局</span>
        </el-menu-item>
        <el-menu-item index="/applications">
          <el-icon><Tickets /></el-icon>
          <span>加入申请</span>
        </el-menu-item>
        <el-menu-item index="/reports">
          <el-icon><Warning /></el-icon>
          <span>举报处理</span>
        </el-menu-item>
        <el-menu-item index="/reviews">
          <el-icon><Star /></el-icon>
          <span>评价管理</span>
        </el-menu-item>
        <el-menu-item index="/messages">
          <el-icon><ChatDotRound /></el-icon>
          <span>消息管理</span>
        </el-menu-item>
        <el-menu-item index="/dicts">
          <el-icon><Collection /></el-icon>
          <span>字典管理</span>
        </el-menu-item>
		<el-menu-item index="/content-reviews">
		  <el-icon><View /></el-icon>
		  <span>内容审核</span>
		</el-menu-item>
		<el-menu-item index="/uploads">
		  <el-icon><Picture /></el-icon>
		  <span>媒体审核</span>
		</el-menu-item>
		<el-menu-item index="/audit-logs">
		  <el-icon><Document /></el-icon>
		  <span>操作审计</span>
		</el-menu-item>
      </el-menu>

      <div class="sidebar-note">
        <div class="note-title">审核原则</div>
        <div class="note-copy">只保留滑雪行程组织能力，避免非行程社交、交易撮合和联系方式外露。</div>
      </div>
    </el-aside>

    <el-container>
      <el-header class="topbar">
        <div>
          <div class="topbar-title">滑雪行程组局工具管理端</div>
          <div class="topbar-subtitle">内容安全、举报处理与基础运营</div>
        </div>
        <div class="topbar-actions">
          <el-tag effect="plain" type="success">API 接入</el-tag>
          <el-button round @click="openDocs">运营指南</el-button>
          <el-button round @click="logout">退出</el-button>
        </div>
      </el-header>

      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { Calendar, ChatDotRound, Collection, Document, HomeFilled, Picture, Star, Tickets, User, View, Warning } from "@element-plus/icons-vue"
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
