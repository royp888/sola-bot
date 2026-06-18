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

      <div class="nav-scroll">
        <nav class="nav" aria-label="主导航">
          <section v-for="section in navSections" :key="section.key" class="nav-section" :class="{ 'is-active': isSectionActive(section) }">
            <p v-if="!collapsed || isMobile" class="nav-section-label">{{ section.label }}</p>
            <div class="nav-section-list">
              <router-link
                v-for="item in section.items"
                :key="item.path"
                :to="item.path"
                class="nav-item"
                :class="{ 'is-active': isRouteActive(item) }"
                :aria-current="isRouteActive(item) ? 'page' : undefined"
                :title="item.label"
                @click="handleMenuNavigate"
              >
                <span class="nav-icon"><el-icon><component :is="item.icon" /></el-icon></span>
                <span v-if="!collapsed || isMobile" class="nav-copy">{{ item.label }}</span>
              </router-link>
            </div>
          </section>
        </nav>
      </div>
    </aside>

    <div class="content">
      <header class="topbar">
        <div class="topbar-inner">
          <div class="topbar-left">
            <el-button
              text
              :icon="isMobile ? Menu : (collapsed ? Expand : Fold)"
              :aria-label="isMobile ? '打开菜单' : (collapsed ? '展开菜单' : '收起菜单')"
              @click="toggleNavigation"
            />
            <div class="top-crumb" :title="topbarContext">{{ topbarContext }}</div>
          </div>

          <div class="topbar-right">
            <el-tag effect="plain" class="user-tag">{{ userRoleLabel }} · {{ userLabel }}</el-tag>
            <el-button text :icon="isDark ? Sunny : Moon" :aria-label="isDark ? '切换白天模式' : '切换夜间模式'" @click="toggleTheme" />
            <el-dropdown @command="handleCommand">
              <el-button :icon="Setting" circle />
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="logout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
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
  Moon,
  Setting,
  Sunny,
  Tickets,
  Tools,
  Trophy,
  UserFilled,
} from "@element-plus/icons-vue";
import { clearSession, getStoredUser } from "@/api/session";

interface NavItem {
  path: string;
  label: string;
  description: string;
  icon: unknown;
  matches?: string[];
}

interface NavSection {
  key: string;
  label: string;
  items: NavItem[];
}

const route = useRoute();
const router = useRouter();
const collapsed = ref(localStorage.getItem("sola-sidebar-collapsed") === "1");
const isMobile = ref(false);
const mobileNavOpen = ref(false);
const isDark = ref(document.documentElement.classList.contains("dark"));

const appName = "Sola 管理台";
const appDesc = "社群运营中心";
const brandInitial = "S";

const navSections: NavSection[] = [
  {
    key: "overview",
    label: "总览",
    items: [
      { path: "/", label: "运营总览", description: "优先查看异常、任务与系统状态", icon: House },
      { path: "/stats", label: "数据分析", description: "查看趋势、活跃度与积分表现", icon: DataAnalysis },
    ],
  },
  {
    key: "members",
    label: "成员运营",
    items: [
      { path: "/users", label: "成员管理", description: "批量处理成员、积分与封禁", icon: UserFilled },
      { path: "/points/logs", label: "积分流水", description: "按成员追踪积分变动与处理原因", icon: Tickets },
      { path: "/violations", label: "违规记录", description: "查看违规记录与处置进度", icon: CircleClose },
      { path: "/admin/bans", label: "封禁与警告", description: "管理封禁、警告与处理历史", icon: Lock },
    ],
  },
  {
    key: "community",
    label: "社群配置",
    items: [
      { path: "/admin/config", label: "群组设置", description: "管理群组接入、权限与管理员", icon: ChatDotRound },
      { path: "/chats", label: "群组会话", description: "查看已接入的群组与频道列表", icon: MessageBox },
      { path: "/invite-links", label: "邀请追踪", description: "跟踪邀请链接与成员转化情况", icon: Files },
      { path: "/bots", label: "机器人管理", description: "查看机器人令牌、状态与实例配置", icon: Cpu },
    ],
  },
  {
    key: "rules",
    label: "规则自动化",
    items: [
      { path: "/points/config", label: "积分规则", description: "配置奖励、扣分与排行榜策略", icon: Coin },
      { path: "/levels", label: "积分等级", description: "调整成长等级、门槛与展示头衔", icon: Trophy },
      { path: "/keywords", label: "关键词规则", description: "维护词库、匹配策略与触发方式", icon: ChatDotRound },
      { path: "/auto-replies", label: "自动回复", description: "配置关键词与场景自动回复", icon: MessageBox },
    ],
  },
  {
    key: "content",
    label: "内容活动",
    items: [
      { path: "/posts", label: "发布任务", description: "安排图文、视频与定时发布任务", icon: Calendar },
      { path: "/templates", label: "内容模板", description: "沉淀常用发布内容与消息模板", icon: Files },
      { path: "/lottery", label: "活动抽奖", description: "管理进行中与历史抽奖活动", icon: Tickets },
    ],
  },
  {
    key: "system",
    label: "系统",
    items: [
      { path: "/backup", label: "备份恢复", description: "管理备份策略与恢复操作", icon: Files },
      { path: "/settings", label: "系统设置", description: "Turnstile、管理员密码等全局配置", icon: Tools },
    ],
  },
];

const allNavItems = navSections.flatMap((section) => section.items);

function matchesPath(pattern: string, currentPath: string): boolean {
  if (pattern === "/") {
    return currentPath === "/";
  }
  return currentPath === pattern || currentPath.startsWith(`${pattern}/`);
}

function isRouteActive(item: NavItem): boolean {
  return (item.matches ?? [item.path]).some((pattern) => matchesPath(pattern, route.path));
}

function isSectionActive(section: NavSection): boolean {
  return section.items.some((item) => isRouteActive(item));
}

const currentItem = computed<NavItem | undefined>(() => allNavItems.find((item) => isRouteActive(item)));

const currentSectionName = computed(() => navSections.find((section) => section.items.some((item) => isRouteActive(item)))?.label || "工作台");

const currentMetaTitle = computed(() => {
  const matched = route.matched
    .slice()
    .reverse()
    .find((record) => typeof record.meta?.title === "string");
  return matched?.meta?.title as string | undefined;
});

const currentTitle = computed(() => currentItem.value?.label || currentMetaTitle.value || "控制台");
const topbarContext = computed(() => {
  if (currentSectionName.value === currentTitle.value) {
    return currentTitle.value;
  }
  return `${currentSectionName.value} / ${currentTitle.value}`;
});
const storedUser = computed(() => getStoredUser());
const userLabel = computed(() => storedUser.value?.username || "未命名用户");
const userRoleLabel = computed(() => (storedUser.value?.role === "super_admin" ? "超级管理员" : "群主管理员"));

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
  localStorage.setItem("sola-sidebar-collapsed", collapsed.value ? "1" : "0");
}

function closeMobileNav(): void {
  mobileNavOpen.value = false;
}

function handleMenuNavigate(): void {
  if (isMobile.value) {
    closeMobileNav();
  }
}

function toggleTheme(): void {
  isDark.value = !isDark.value;
  document.documentElement.classList.toggle("dark", isDark.value);
  localStorage.setItem("sola-theme", isDark.value ? "dark" : "light");
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
  background: var(--app-bg);
}

.mobile-backdrop {
  position: fixed;
  inset: 0;
  z-index: 30;
  border: 0;
  background: var(--app-mask);
}

.sidebar {
  position: sticky;
  top: 0;
  display: flex;
  flex-direction: column;
  width: 244px;
  height: 100vh;
  padding: 16px 12px 14px;
  border-right: 1px solid var(--app-border);
  background: var(--app-surface);
  backdrop-filter: blur(14px);
}

.sidebar.collapsed {
  width: 78px;
}

.brand-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 14px;
  padding: 2px 10px 14px;
  border-bottom: 1px solid var(--app-border);
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.brand-mark {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border: 1px solid var(--app-accent-hover-border);
  border-radius: 10px;
  background: linear-gradient(180deg, var(--app-accent-soft), var(--app-tint-light));
  box-shadow: inset 0 1px 0 var(--app-tint-light);
  color: var(--app-text);
  font-size: 15px;
  font-weight: 800;
}

.brand-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.brand-copy strong {
  font-size: 13px;
}

.brand-copy span {
  color: var(--app-muted);
  font-size: 11px;
}

.nav-scroll {
  flex: 1;
  overflow-y: auto;
  margin-right: -4px;
  padding-right: 4px;
  padding-bottom: 8px;
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.nav-section {
  display: flex;
  flex-direction: column;
  gap: 7px;
  padding: 7px 8px;
  border-radius: 14px;
  transition: background 0.18s ease, border-color 0.18s ease;
}

.nav-section.is-active {
  background: var(--app-nav-section-active);
}

.nav-section-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.nav-section-label {
  margin: 0;
  padding: 0 0 0 12px;
  color: var(--app-nav-label);
  font-size: 11px;
  font-weight: 500;
  line-height: 1.4;
}

.nav-section.is-active .nav-section-label {
  color: var(--app-nav-label-active);
}

.nav-item {
  position: relative;
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  min-height: 40px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: 10px;
  color: inherit;
  text-decoration: none;
  transition: border-color 0.18s ease, background 0.18s ease, color 0.18s ease;
}

.nav-item::before {
  content: "";
  position: absolute;
  left: 3px;
  top: 20%;
  bottom: 20%;
  width: 2px;
  border-radius: 999px;
  background: transparent;
  transition: background 0.18s ease;
}

.nav-item:hover {
  border-color: var(--app-nav-hover-border);
  background: var(--app-nav-hover-bg);
}

.nav-item.is-active {
  border-color: var(--app-nav-active-border);
  background: var(--app-nav-active-bg);
}

.nav-item.is-active::before {
  background: var(--app-nav-indicator);
}

.nav-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--app-nav-icon);
  transition: background 0.18s ease, border-color 0.18s ease, color 0.18s ease;
}

.nav-icon :deep(.el-icon) {
  font-size: 14px;
}

.nav-item:hover .nav-icon {
  border-color: var(--app-tint-light);
  background: var(--app-tint-light);
  color: var(--app-nav-icon-hover);
}

.nav-item.is-active .nav-icon {
  border-color: var(--app-nav-active-border);
  background: var(--app-tint-light);
  color: var(--app-nav-icon-active);
}

.nav-copy {
  min-width: 0;
  color: var(--app-nav-text);
  font-size: 13px;
  font-weight: 560;
  line-height: 1.2;
}

.nav-item.is-active .nav-copy {
  color: var(--app-nav-text-active);
}

.sidebar.collapsed .brand-row {
  justify-content: center;
  padding-inline: 0;
}

.sidebar.collapsed .brand {
  justify-content: center;
}

.sidebar.collapsed .nav-item {
  grid-template-columns: 1fr;
  justify-items: center;
  padding: 0;
}

.content {
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: transparent;
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  min-height: 58px;
  border-bottom: 1px solid var(--app-border);
  background: color-mix(in srgb, var(--app-surface) 80%, transparent);
  backdrop-filter: blur(12px);
}

.topbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  width: 100%;
  max-width: 1320px;
  min-height: 58px;
  margin: 0 auto;
  padding: 0 28px;
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

.top-crumb {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--app-nav-text);
  font-size: 12px;
  font-weight: 600;
}

.user-tag {
  color: var(--app-muted-strong);
  background: var(--app-surface-2);
  border-color: var(--app-border);
}

.viewport {
  width: 100%;
  max-width: 1320px;
  margin: 0 auto;
  padding: 18px 28px 28px;
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
    width: 236px;
  }

  .sidebar.collapsed {
    width: 76px;
  }

  .topbar-inner,
  .viewport {
    padding-inline: 20px;
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
    width: min(86vw, 308px);
    transform: translateX(-100%);
    transition: transform 0.2s ease;
  }

  .sidebar.mobile-open {
    transform: translateX(0);
  }

  .nav-section-label {
    padding-left: 46px;
  }

  .user-tag {
    display: none;
  }

  .topbar-inner {
    padding: 0 14px;
  }

  .viewport {
    padding: 14px;
  }
}
</style>
