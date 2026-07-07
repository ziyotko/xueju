import { createRouter, createWebHistory } from "vue-router"
import DashboardView from "../views/DashboardView.vue"
import AdminListView from "../views/AdminListView.vue"
import LoginView from "../views/LoginView.vue"

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/login", name: "login", component: LoginView, meta: { public: true } },
    { path: "/", name: "dashboard", component: DashboardView },
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
      props: { resource: "events", title: "滑雪局管理", actionText: "下架活动" }
    },
    {
      path: "/applications",
      name: "applications",
      component: AdminListView,
      props: { resource: "applications", title: "加入申请管理", actionText: "查看记录" }
    },
    {
      path: "/reports",
      name: "reports",
      component: AdminListView,
      props: { resource: "reports", title: "举报处理", actionText: "标记已处理" }
    },
    {
      path: "/reviews",
      name: "reviews",
      component: AdminListView,
      props: { resource: "reviews", title: "评价管理", actionText: "隐藏评价" }
    },
    {
      path: "/messages",
      name: "messages",
      component: AdminListView,
      props: { resource: "messages", title: "消息管理", actionText: "隐藏消息" }
    },
    {
      path: "/dicts",
      name: "dicts",
      component: AdminListView,
      props: { resource: "dicts", title: "字典管理", actionText: "停用雪场" }
    }
  ]
})

router.beforeEach((to) => {
  if (to.meta.public) return true
  if (localStorage.getItem("xueju_admin_token")) return true
  return { path: "/login", query: { redirect: to.fullPath } }
})

export default router
