<template>
  <div class="shell">
    <transition name="nav-fade">
      <button v-if="isMobile && mobileNavOpen" class="mobile-backdrop" type="button" aria-label="关闭菜单" @click="closeMobileNav" />
    </transition>

    <aside class="sidebar" :class="{ collapsed, 'mobile-open': mobileNavOpen }">
      <div class="brand-row">
        <div class="brand">
          <div class="brand-mark">{{ brandInitial }}</div>
          <div v-if="!collapsed || isMobile" class="brand-copy">
            <strong>{{ appName }}</strong>
            <span>{{ appDesc }}</span>
          </div>
        </div>
        <el-button v-if="isMobile" text :icon="Close" aria-label="关闭菜单" @click="closeMobileNav" />
      </div>

      <div v-if="!collapsed || isMobile" class="context-card">
        <span class="context-kicker">当前工作区</span>
        <strong>{{ currentTitle }}</strong>
        <p>{{ currentItemDescription || currentSectionName }}</p>
        <div class="context-tags">
          <span>{{ currentSectionName }}</span>
          <span>{{ userRoleLabel }}</span>
        </div>
      </div>

      <div class="nav-scroll">
        <el-menu
          class="nav"
          :default-active="route.path"
          :collapse="collapsed && !isMobile"
          :router="true"
          background-color="transparent"
          text-color="var(--app-text)"
          active-text-color="var(--app-text)"
        >
          <template v-for="section in navSections" :key="section.key">
            <li v-if="!collapsed || isMobile" class="nav-section-label">{{ section.label }}</li>
            <el-menu-item
              v-for="item in section.items"
              :key="item.index"
              :index="item.index"
              class="nav-item"
              @click="handleMenuNavigate"
            >
              <span class="nav-active-line" />
              <el-icon><component :is="item.icon" /></el-icon>
              <template #title>
                <div class="nav-copy">
                  <span>{{ item.label }}</span>
                  <small v-if="(!collapsed || isMobile) && item.description">{{ item.description }}</small>
                </div>
              </template>
            </el-menu-item>
          </template>
        </el-menu>
      </div>

      <div v-if="!collapsed || isMobile" class="sidebar-footer">
        <div class="foot-block">
          <span class="foot-label">登录身份</span>
          <strong>{{ userLabel }}</strong>
          <span class="foot-meta">{{ userRoleLabel }}</span>
        </div>
        <div class="foot-block foot-block-muted">
          <span class="foot-label">接口环境</span>
          <strong>{{ apiLabel }}</strong>
          <span class="foot-meta">{{ apiBase }}</span>
        </div>
      </div>
    </aside>

    <div class="content">
      <header class="topbar">
        <div class="topbar-left">
          <el-button
            text
            :icon="isMobile ? Menu : (collapsed ? Expand : Fold)"
            :aria-label="isMobile ? '打开菜单' : (collapsed ? '展开菜单' : '收起菜单')"
            @click="toggleNavigation"
          />
          <div class="topbar-copy">
            <div class="top-crumb">{{ currentSectionName }}</div>
            <div class="top-title-row">
              <div class="top-title">{{ currentTitle }}</div>
              <span v-if="currentItemDescription" class="top-route">{{ currentItemDescription }}</span>
            </div>
          </div>
        </div>

        <div class="topbar-right">
          <div class="topbar-context desktop-only">
            <span>当前模块</span>
            <strong>{{ currentSectionName }}</strong>
          </div>
          <el-tag effect="plain" class="user-tag">{{ userRoleLabel }} · {{ userLabel }}</el-tag>
          <el-dropdown @command="handleCommand">
            <el-button :icon="Setting" circle />
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <main class="viewport">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  Calendar,
  ChatDotRound,
  CircleClose,
  Close,
  Coin,
  Cpu,
  DataAnalysis,
  Expand,
  Files,
  Fold,
  House,
  Lock,
  Menu,
  MessageBox,
  Setting,
  Tickets,
  Trophy,
  UserFilled,
} from "@element-plus/icons-vue";
import { clearSession, getStoredUser } from "@/api/session";

interface NavItem {
  index: string;
  label: string;
  description: string;
  icon: unknown;
}

interface NavSection {
  key: string;
  label: string;
  items: NavItem[];
}

const route = useRoute();
const router = useRouter();
const collapsed = ref(false);
const isMobile = ref(false);
const mobileNavOpen = ref(false);

const appName = "Sola 管理台";
const appDesc = "社群运营与自动化";
const brandInitial = "S";

const navSections: NavSection[] = [
  {
    key: "overview",
    label: "总览",
    items: [
      { index: "/dashboard", label: "运营总览", description: "优先异常、任务与系统状态", icon: House },
      { index: "/stats", label: "数据分析", description: "查看趋势、积分与活跃情况", icon: DataAnalysis },
    ],
  },
  {
    key: "accounts",
    label: "账号与对象",
    items: [
      { index: "/users", label: "成员管理", description: "批量处理成员、积分与封禁", icon: UserFilled },
      { index: "/groups", label: "群组设置", description: "管理群组接入与权限配置", icon: ChatDotRound },
    ],
  },
  {
    key: "rules",
    label: "社群规则",
    items: [
      { index: "/violations", label: "违规处理", description: "查看违规记录与处置状态", icon: CircleClose },
      { index: "/keywords", label: "敏感词", description: "维护规则词库与匹配策略", icon: Lock },
      { index: "/templates", label: "消息模板", description: "沉淀常用回复与发布模板", icon: Files },
    ],
  },
  {
    key: "growth",
    label: "成长与激励",
    items: [
      { index: "/points", label: "积分规则", description: "配置积分发放与排行榜策略", icon: Coin },
      { index: "/lotteries", label: "抽奖活动", description: "管理进行中与历史抽奖", icon: Trophy },
    ],
  },
  {
    key: "operations",
    label: "内容与运维",
    items: [
      { index: "/posts", label: "发布任务", description: "编排定时消息与自动发布", icon: Calendar },
      { index: "/schedules", label: "任务调度", description: "查看调度配置与执行状态", icon: Tickets },
      { index: "/system", label: "系统设置", description: "查看接口与机器人运行参数", icon: Cpu },
      { index: "/messages", label: "消息记录", description: "追踪消息投递与交互记录", icon: MessageBox },
    ],
  },
];

const currentItem = computed<NavItem | undefined>(() =>
  navSections.flatMap((section) => section.items).find((item) => route.path.startsWith(item.index)),
);

const currentSectionName = computed(() =>
  navSections.find((section) => section.items.some((item) => route.path.startsWith(item.index)))?.label || "工作台",
);

const currentTitle = computed(() => currentItem.value?.label || "控制台");
const currentItemDescription = computed(() => currentItem.value?.description || "");

const storedUser = computed(() => getStoredUser());
const userLabel = computed(() => storedUser.value?.username || "未命名用户");
const userRoleLabel = computed(() => (storedUser.value?.role === "super_admin" ? "超级管理员" : "群主管理员"));
const apiBase = computed(() => (import.meta.env.VITE_API_BASE_URL as string | undefined) || "/api");
const apiLabel = computed(() => (apiBase.value.startsWith("http") ? "远程接口" : "同源接口"));

function syncViewport(): void {
  isMobile.value = window.innerWidth <= 720;
  if (isMobile.value) {
    collapsed.value = false;
  }
}

function toggleNavigation(): void {
  if (isMobile.value) {
    mobileNavOpen.value = !mobileNavOpen.value;
    return;
  }
  collapsed.value = !collapsed.value;
}

function closeMobileNav(): void {
  mobileNavOpen.value = false;
}

function handleMenuNavigate(): void {
  if (isMobile.value) {
    closeMobileNav();
  }
}

function handleCommand(command: string): void {
  if (command === "logout") {
    clearSession();
    router.push("/login");
  }
}

watch(
  () => route.fullPath,
  () => {
    if (isMobile.value) {
      closeMobileNav();
    }
  },
);

onMounted(() => {
  syncViewport();
  window.addEventListener("resize", syncViewport);
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", syncViewport);
});
</script>

<style scoped>
.shell {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  min-height: 100vh;
  background:
    radial-gradient(circle at top left, rgba(73, 113, 197, 0.16), transparent 26%),
    linear-gradient(180deg, rgba(8, 11, 17, 0.98), rgba(10, 13, 19, 0.98));
}

.mobile-backdrop {
  position: fixed;
  inset: 0;
  z-index: 30;
  border: 0;
  background: rgba(2, 6, 12, 0.58);
}

.sidebar {
  position: sticky;
  top: 0;
  display: flex;
  flex-direction: column;
  width: 274px;
  height: 100vh;
  padding: 18px 14px 16px;
  border-right: 1px solid rgba(255, 255, 255, 0.05);
  background: rgba(9, 13, 20, 0.88);
  backdrop-filter: blur(20px);
}

.sidebar.collapsed {
  width: 92px;
}

.brand-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.brand-mark {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border: 1px solid rgba(120, 166, 255, 0.18);
  border-radius: 14px;
  background: linear-gradient(180deg, rgba(120, 166, 255, 0.2), rgba(120, 166, 255, 0.08));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.06);
  color: var(--app-text);
  font-size: 16px;
  font-weight: 800;
}

.brand-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.brand-copy strong {
  font-size: 14px;
}

.brand-copy span {
  color: var(--app-muted);
  font-size: 12px;
}

.context-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;
  padding: 14px;
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(20, 28, 41, 0.95), rgba(15, 21, 31, 0.95));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

.context-kicker,
.foot-label,
.foot-meta {
  color: var(--app-muted);
  font-size: 12px;
}

.context-card strong,
.foot-block strong {
  font-size: 14px;
  font-weight: 700;
}

.context-card p {
  margin: 0;
  color: var(--app-muted-strong);
  font-size: 12px;
  line-height: 1.55;
}

.context-tags {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.context-tags span {
  padding: 5px 10px;
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.03);
  color: var(--app-muted-strong);
  font-size: 11px;
}

.nav-scroll {
  flex: 1;
  overflow-y: auto;
  margin-right: -8px;
  padding-right: 8px;
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-right: 0;
}

.nav-section-label {
  margin: 10px 0 2px;
  padding: 0 12px;
  color: rgba(164, 177, 196, 0.68);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  list-style: none;
  text-transform: uppercase;
}

.nav-item {
  position: relative;
  min-height: 48px;
  border-radius: 14px;
}

.nav-active-line {
  position: absolute;
  left: 10px;
  top: 11px;
  bottom: 11px;
  width: 3px;
  border-radius: 999px;
  background: transparent;
}

.nav-item.is-active .nav-active-line {
  background: linear-gradient(180deg, #8fb5ff, #6d90dc);
}

.nav-copy {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.nav-copy span {
  font-size: 13px;
  font-weight: 600;
}

.nav-copy small {
  color: rgba(164, 177, 196, 0.72);
  font-size: 11px;
  line-height: 1.45;
}

.sidebar-footer {
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.foot-block {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 10px 12px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.02);
}

.foot-block-muted {
  opacity: 0.82;
}

.content {
  min-width: 0;
  background: transparent;
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 72px;
  padding: 0 24px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  background: rgba(9, 13, 20, 0.8);
  backdrop-filter: blur(18px);
}

.topbar-left,
.topbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.topbar-right {
  justify-content: flex-end;
}

.topbar-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.top-crumb,
.top-route,
.topbar-context span {
  color: var(--app-muted);
  font-size: 12px;
}

.top-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.top-title {
  font-size: 18px;
  font-weight: 760;
}

.topbar-context {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 12px;
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.03);
}

.topbar-context strong {
  font-size: 12px;
}

.user-tag {
  color: var(--app-text);
}

.viewport {
  padding: 24px;
}

.nav-fade-enter-active,
.nav-fade-leave-active {
  transition: opacity 0.18s ease;
}

.nav-fade-enter-from,
.nav-fade-leave-to {
  opacity: 0;
}

@media (max-width: 960px) {
  .sidebar {
    width: 252px;
  }

  .sidebar.collapsed {
    width: 86px;
  }

  .desktop-only {
    display: none;
  }
}

@media (max-width: 720px) {
  .shell {
    grid-template-columns: 1fr;
  }

  .sidebar {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    z-index: 40;
    width: min(88vw, 320px);
    transform: translateX(-100%);
    transition: transform 0.2s ease;
  }

  .sidebar.mobile-open {
    transform: translateX(0);
  }

  .topbar {
    padding: 0 14px;
  }

  .viewport {
    padding: 14px;
  }
}
</style>
