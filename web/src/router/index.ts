import { createRouter, createWebHistory } from "vue-router";
import { getStoredToken } from "@/api/session";
import AdminLayout from "@/layouts/AdminLayout.vue";
import AdminConfigView from "@/views/AdminConfigView.vue";
import AutoRepliesView from "@/views/AutoRepliesView.vue";
import BackupView from "@/views/BackupView.vue";
import BansView from "@/views/BansView.vue";
import BotsView from "@/views/BotsView.vue";
import ChatsView from "@/views/ChatsView.vue";
import DashboardView from "@/views/DashboardView.vue";
import InviteLinksView from "@/views/InviteLinksView.vue";
import KeywordsView from "@/views/KeywordsView.vue";
import LevelsView from "@/views/LevelsView.vue";
import LoginView from "@/views/LoginView.vue";
import LotteryView from "@/views/LotteryView.vue";
import PointLogsView from "@/views/PointLogsView.vue";
import PointsConfigView from "@/views/PointsConfigView.vue";
import PostsView from "@/views/PostsView.vue";
import StatsView from "@/views/StatsView.vue";
import TemplatesView from "@/views/TemplatesView.vue";
import UsersView from "@/views/UsersView.vue";
import ViolationsView from "@/views/ViolationsView.vue";

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
            title: "Bot 接入",
          },
        },
        {
          path: "chats",
          name: "chats",
          component: ChatsView,
          meta: {
            title: "群组管理",
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
            title: "私聊/用户运营",
          },
        },
        {
          path: "points/config",
          name: "points-config",
          component: PointsConfigView,
          meta: {
            title: "积分配置",
          },
        },
        {
          path: "points/logs",
          name: "points-logs",
          component: PointLogsView,
          meta: {
            title: "积分流水",
          },
        },
        {
          path: "admin/config",
          name: "admin-config",
          component: AdminConfigView,
          meta: {
            title: "群组配置",
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
            title: "消息模板",
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
            title: "邀请链接",
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
            title: "数据分析",
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
