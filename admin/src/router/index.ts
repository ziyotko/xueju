import { createRouter, createWebHistory } from "vue-router"
import DashboardView from "../views/DashboardView.vue"
import AdminListView from "../views/AdminListView.vue"

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      name: "dashboard",
      component: DashboardView
    },
    {
      path: "/users",
      name: "users",
      component: AdminListView,
      props: { resource: "users", title: "用户管理", actionText: "禁用用户" }
    },
    {
      path: "/events",
      name: "events",
      component: AdminListView,
      props: { resource: "events", title: "滑雪行程管理", actionText: "下架行程" }
    },
    {
      path: "/content-reviews",
      name: "content-reviews",
      component: AdminListView,
      props: { resource: "content-reviews", title: "内容审核", actionText: "驳回内容" }
    },
    {
      path: "/messages",
      name: "messages",
      component: AdminListView,
      props: { resource: "messages", title: "消息管理", actionText: "隐藏消息" }
    },
    {
      path: "/reviews",
      name: "reviews",
      component: AdminListView,
      props: { resource: "reviews", title: "评价管理", actionText: "隐藏评价" }
    },
    {
      path: "/reports",
      name: "reports",
      component: AdminListView,
      props: { resource: "reports", title: "举报处理", actionText: "标记已处理" }
    }
  ]
})

export default router
