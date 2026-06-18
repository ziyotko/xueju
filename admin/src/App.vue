<template>
  <el-container class="layout">
    <el-aside width="220px" class="sidebar">
      <div class="brand">雪局后台</div>
      <el-menu router default-active="/" background-color="#111827" text-color="#cbd5e1" active-text-color="#ffffff">
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
          <span>滑雪行程</span>
        </el-menu-item>
        <el-menu-item index="/content-reviews">
          <el-icon><DocumentChecked /></el-icon>
          <span>内容审核</span>
        </el-menu-item>
        <el-menu-item index="/messages">
          <el-icon><ChatDotRound /></el-icon>
          <span>消息管理</span>
        </el-menu-item>
        <el-menu-item index="/reviews">
          <el-icon><Star /></el-icon>
          <span>评价管理</span>
        </el-menu-item>
        <el-menu-item index="/reports">
          <el-icon><Warning /></el-icon>
          <span>举报处理</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="topbar">
        <span>滑雪行程组局工具管理端</span>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { Calendar, ChatDotRound, DocumentChecked, HomeFilled, Star, User, Warning } from "@element-plus/icons-vue"
</script>
