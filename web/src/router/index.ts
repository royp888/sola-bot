import { createRouter, createWebHistory } from "vue-router";
import { getStoredToken } from "@/api/session";

const AdminLayout = () => import("@/layouts/AdminLayout.vue");
const AdminConfigView = () => import("@/views/AdminConfigView.vue");
const AutoRepliesView = () => import("@/views/AutoRepliesView.vue");
const BackupView = () => import("@/views/BackupView.vue");
const BansView = () => import("@/views/BansView.vue");
const BotsView = () => import("@/views/BotsView.vue");
const ChatsView = () => import("@/views/ChatsView.vue");
const DashboardView = () => import("@/views/DashboardView.vue");
const InviteLinksView = () => import("@/views/InviteLinksView.vue");
const KeywordsView = () => import("@/views/KeywordsView.vue");
const LevelsView = () => import("@/views/LevelsView.vue");
const LoginView = () => import("@/views/LoginView.vue");
const LotteryView = () => import("@/views/LotteryView.vue");
const PointLogsView = () => import("@/views/PointLogsView.vue");
const PointsConfigView = () => import("@/views/PointsConfigView.vue");
const PostsView = () => import("@/views/PostsView.vue");
const StatsView = () => import("@/views/StatsView.vue");
const TemplatesView = () => import("@/views/TemplatesView.vue");
const UsersView = () => import("@/views/UsersView.vue");
const ViolationsView = () => import("@/views/ViolationsView.vue");

const appName = import.meta.env.VITE_APP_NAME?.trim() || "Sola Bot";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/login",
      name: "login",
      component: LoginView,
      meta: {
        title: "登录",
        public: true,
      },
    },
    {
      path: "/dashboard",
      redirect: "/",
    },
    {
      path: "/",
      component: AdminLayout,
      meta: {
        requiresAuth: true,
      },
      children: [
        {
          path: "",
          name: "dashboard",
          component: DashboardView,
          meta: {
            title: "概览",
          },
        },
        {
          path: "bots",
          name: "bots",
          component: BotsView,
          meta: {
            title: "机器人管理",
          },
        },
        {
          path: "chats",
          name: "chats",
          component: ChatsView,
          meta: {
            title: "群组 / 频道",
          },
        },
        {
          path: "points-config",
          redirect: "/points/config",
        },
        {
          path: "users",
          name: "users",
          component: UsersView,
          meta: {
            title: "成员管理",
          },
        },
        {
          path: "points/config",
          name: "points-config",
          component: PointsConfigView,
          meta: {
            title: "积分规则",
          },
        },
        {
          path: "points/logs",
          name: "points-logs",
          component: PointLogsView,
          meta: {
            title: "积分记录",
          },
        },
        {
          path: "admin/config",
          name: "admin-config",
          component: AdminConfigView,
          meta: {
            title: "群组设置",
          },
        },
        {
          path: "admin/bans",
          name: "admin-bans",
          component: BansView,
          meta: {
            title: "封禁与警告",
          },
        },
        {
          path: "levels",
          name: "levels",
          component: LevelsView,
          meta: {
            title: "等级规则",
          },
        },
        {
          path: "keywords",
          name: "keywords",
          component: KeywordsView,
          meta: {
            title: "关键词规则",
          },
        },
        {
          path: "auto-replies",
          name: "auto-replies",
          component: AutoRepliesView,
          meta: {
            title: "自动回复",
          },
        },
        {
          path: "violations",
          name: "violations",
          component: ViolationsView,
          meta: {
            title: "违规记录",
          },
        },
        {
          path: "posts",
          name: "posts",
          component: PostsView,
          meta: {
            title: "发布任务",
          },
        },
        {
          path: "templates",
          name: "templates",
          component: TemplatesView,
          meta: {
            title: "内容模板",
          },
        },
        {
          path: "lottery",
          name: "lottery",
          component: LotteryView,
          meta: {
            title: "活动抽奖",
          },
        },
        {
          path: "invite-links",
          name: "invite-links",
          component: InviteLinksView,
          meta: {
            title: "邀请链接追踪",
          },
        },
        {
          path: "backup",
          name: "backup",
          component: BackupView,
          meta: {
            title: "备份恢复",
          },
        },
        {
          path: "stats",
          name: "stats",
          component: StatsView,
          meta: {
            title: "运营分析",
          },
        },
      ],
    },
  ],
  scrollBehavior() {
    return { top: 0 };
  },
});

router.beforeEach((to) => {
  if (!to.meta.public && to.meta.requiresAuth && !getStoredToken()) {
    return {
      name: "login",
      query: {
        redirect: to.fullPath,
      },
    };
  }

  const title = to.matched
    .slice()
    .reverse()
    .find((record) => record.meta?.title)?.meta?.title as string | undefined;

  document.title = title ? `${title} · ${appName}` : appName;
  return true;
});

export default router;

