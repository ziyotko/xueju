import { createRouter, createWebHistory } from "vue-router"

const DashboardView = () => import("../views/DashboardView.vue")
const AdminListView = () => import("../views/AdminListView.vue")
const LoginView = () => import("../views/LoginView.vue")
const ModerationView = () => import("../views/ModerationView.vue")

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
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
      path: "/dicts",
      name: "dicts",
      component: AdminListView,
      props: { resource: "dicts", title: "字典管理", actionText: "停用雪场" }
	},
	{ path: "/moderation", name: "moderation", component: ModerationView },
	{ path: "/content-reviews", redirect: (to) => ({ path: "/moderation", query: { ...to.query, tab: "content" } }) },
	{ path: "/messages", redirect: (to) => ({ path: "/moderation", query: { ...to.query, tab: "messages" } }) },
	{ path: "/uploads", redirect: (to) => ({ path: "/moderation", query: { ...to.query, tab: "media" } }) },
	{
	  path: "/audit-logs",
	  name: "audit-logs",
	  component: AdminListView,
	  props: { resource: "audit-logs", title: "操作审计", actionText: "查看记录" }
    }
  ]
})

router.beforeEach((to) => {
  if (to.meta.public) return true
  if (localStorage.getItem("xueju_admin_token")) return true
  return { path: "/login", query: { redirect: to.fullPath } }
})

export default router
