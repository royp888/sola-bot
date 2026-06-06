import { createRouter, createWebHashHistory } from "vue-router";

const Dashboard = () => import("@/mini/views/Dashboard.vue");
const ChatSettings = () => import("@/mini/views/ChatSettings.vue");
const QuickPublish = () => import("@/mini/views/QuickPublish.vue");
const Lottery = () => import("@/mini/views/Lottery.vue");

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: "/",
      name: "dashboard",
      component: Dashboard,
      meta: { title: "仪表盘" },
    },
    {
      path: "/settings",
      name: "settings",
      component: ChatSettings,
      meta: { title: "群设置" },
    },
    {
      path: "/publish",
      name: "publish",
      component: QuickPublish,
      meta: { title: "快捷发布" },
    },
    {
      path: "/lottery",
      name: "lottery",
      component: Lottery,
      meta: { title: "抽奖" },
    },
  ],
});

export default router;
