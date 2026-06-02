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

      <div v-if="!collapsed || isMobile" class="workspace-brief">
        <span class="workspace-kicker">当前工作域</span>
        <strong>{{ currentSectionName }}</strong>
        <span>{{ currentItemDescription }}</span>
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
          <span class="foot-label">当前用户</span>
          <strong>{{ userLabel }}</strong>
          <span class="foot-meta">{{ userRoleLabel }}</span>
        </div>
        <div class="foot-block">
          <span class="foot-label">环境</span>
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
          <div class="topbar-pill">
            <span>工作域</span>
            <strong>{{ currentSectionName }}</strong>
          </div>
          <div class="topbar-pill desktop-only">
            <span>API</span>
            <strong>{{ apiLabel }}</strong>
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

const router = useRouter();
const route = useRoute();
const collapsed = ref(false);
const isMobile = ref(false);
const mobileNavOpen = ref(false);
const apiBase = import.meta.env.VITE_API_BASE_URL?.trim() || "/api";
const appName = import.meta.env.VITE_APP_NAME?.trim() || "Sola Bot";
const appDesc = import.meta.env.VITE_APP_DESC?.trim() || "Telegram 运营管理后台";
const brandInitial = computed(() => appName.trim().slice(0, 1).toUpperCase() || "S");
const apiLabel = computed(() => (apiBase === "/api" ? "默认 API" : "自定义 API"));

const navSections: NavSection[] = [
  {
    key: "overview",
    label: "总览",
    items: [
      { index: "/", label: "运营总览", description: "今日待办、任务和异常", icon: House },
      { index: "/stats", label: "运营分析", description: "活跃、来源与趋势观察", icon: DataAnalysis },
    ],
  },
  {
    key: "assets",
    label: "资产",
    items: [
      { index: "/bots", label: "机器人管理", description: "接入状态与运行入口", icon: Cpu },
      { index: "/chats", label: "群组与频道", description: "运营资产与工作台入口", icon: ChatDotRound },
    ],
  },
  {
    key: "member-risk",
    label: "成员与风控",
    items: [
      { index: "/users", label: "成员管理", description: "筛选、调分和批量动作", icon: UserFilled },
      { index: "/points/config", label: "积分规则", description: "积分策略与冷却配置", icon: Coin },
      { index: "/points/logs", label: "积分记录", description: "积分流水和查询回放", icon: Tickets },
      { index: "/levels", label: "等级体系", description: "等级门槛与成长规则", icon: Trophy },
      { index: "/violations", label: "违规处理", description: "违规队列与处理动作", icon: Lock },
      { index: "/admin/bans", label: "封禁与警告", description: "封禁记录与人工介入", icon: CircleClose },
      { index: "/admin/config", label: "群组设置", description: "欢迎、验证和策略开关", icon: Setting },
    ],
  },
  {
    key: "content",
    label: "内容",
    items: [
      { index: "/posts", label: "发布任务", description: "定时发布与任务调度", icon: Calendar },
      { index: "/templates", label: "内容模板", description: "素材与复用内容库", icon: Files },
      { index: "/keywords", label: "关键词规则", description: "命中词与触发策略", icon: MessageBox },
      { index: "/auto-replies", label: "自动回复", description: "命中后回复与互动文案", icon: ChatDotRound },
    ],
  },
  {
    key: "growth",
    label: "增长",
    items: [
      { index: "/invite-links", label: "邀请链接", description: "拉新来源与渠道追踪", icon: Tickets },
      { index: "/lottery", label: "活动抽奖", description: "促活活动与奖励发放", icon: Trophy },
    ],
  },
  {
    key: "system",
    label: "系统",
    items: [{ index: "/backup", label: "备份与恢复", description: "数据导入导出与恢复", icon: Files }],
  },
];

const currentTitle = computed(() => {
  const matched = route.matched
    .slice()
    .reverse()
    .find((record) => typeof record.meta?.title === "string");
  return (matched?.meta?.title as string | undefined) ?? appName;
});

const activeSection = computed(() =>
  navSections.find((section) =>
    section.items.some((item) => item.index === route.path || (item.index !== "/" && route.path.startsWith(item.index))),
  ),
);

const activeItem = computed(() =>
  activeSection.value?.items.find((item) => item.index === route.path || (item.index !== "/" && route.path.startsWith(item.index))),
);

const currentSectionName = computed(() => activeSection.value?.label ?? "控制台");
const currentItemDescription = computed(() => activeItem.value?.description ?? appDesc);
const userLabel = computed(() => getStoredUser()?.name ?? "Operator");
const userRoleLabel = computed(() => {
  const role = getStoredUser()?.role ?? "operator";
  return {
    owner: "Owner",
    admin: "Admin",
    operator: "Operator",
    super_admin: "Super Admin",
  }[role];
});

function syncViewport(): void {
  const mobile = window.innerWidth <= 720;
  isMobile.value = mobile;
  if (!mobile) {
    mobileNavOpen.value = false;
  }
}

function closeMobileNav(): void {
  mobileNavOpen.value = false;
}

function toggleNavigation(): void {
  if (isMobile.value) {
    mobileNavOpen.value = !mobileNavOpen.value;
    return;
  }
  collapsed.value = !collapsed.value;
}

function handleMenuNavigate(): void {
  if (isMobile.value) {
    closeMobileNav();
  }
}

function handleCommand(command: string): void {
  if (command === "logout") {
    clearSession();
    void router.push({ name: "login" });
  }
}

watch(
  () => route.path,
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
}

.mobile-backdrop {
  position: fixed;
  inset: 0;
  z-index: 30;
  border: 0;
  background: rgba(0, 0, 0, 0.46);
}

.sidebar {
  position: sticky;
  top: 0;
  display: flex;
  flex-direction: column;
  width: 264px;
  height: 100vh;
  padding: 16px 12px 14px;
  border-right: 1px solid var(--app-border);
  background: var(--app-surface);
}

.sidebar.collapsed {
  width: 88px;
}

.brand-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
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
  width: 38px;
  height: 38px;
  border: 1px solid rgba(120, 166, 255, 0.22);
  border-radius: 8px;
  background: var(--app-accent-soft);
  color: var(--app-text);
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

.workspace-brief {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 14px;
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius);
  background: var(--app-surface-2);
}

.workspace-kicker,
.foot-label,
.foot-meta {
  color: var(--app-muted);
  font-size: 12px;
}

.workspace-brief strong,
.foot-block strong {
  font-size: 13px;
  font-weight: 600;
}

.nav-scroll {
  flex: 1;
  overflow-y: auto;
  margin-right: -6px;
  padding-right: 6px;
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-right: 0;
}

.nav-section-label {
  margin: 10px 0 0;
  padding: 0 12px;
  color: var(--app-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  list-style: none;
  text-transform: uppercase;
}

.nav-item {
  position: relative;
  min-height: 42px;
}

.nav-active-line {
  position: absolute;
  left: 8px;
  top: 9px;
  bottom: 9px;
  width: 3px;
  border-radius: 999px;
  background: transparent;
}

.nav-item.is-active .nav-active-line {
  background: var(--app-accent);
}

.nav-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-copy small {
  color: var(--app-muted);
  font-size: 11px;
  line-height: 1.35;
}

.sidebar-footer {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid var(--app-border);
}

.foot-block {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.content {
  min-width: 0;
  background: var(--app-bg);
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 64px;
  padding: 0 20px;
  border-bottom: 1px solid var(--app-border);
  background: rgba(15, 17, 21, 0.96);
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
  gap: 3px;
  min-width: 0;
}

.top-crumb,
.top-route,
.topbar-pill span {
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
  font-size: 16px;
  font-weight: 700;
}

.topbar-pill {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 1px;
  padding: 8px 10px;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius);
  background: var(--app-surface);
  line-height: 1.2;
}

.topbar-pill strong {
  font-size: 12px;
}

.user-tag {
  color: var(--app-text);
}

.viewport {
  padding: 20px;
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
    width: 244px;
  }

  .sidebar.collapsed {
    width: 84px;
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
    border-right: 1px solid var(--app-border);
  }

  .sidebar.mobile-open {
    transform: translateX(0);
  }

  .sidebar-footer {
    grid-template-columns: 1fr;
  }

  .topbar {
    padding: 0 14px;
  }

  .topbar-right {
    gap: 8px;
  }

  .topbar-pill {
    display: none;
  }

  .viewport {
    padding: 14px;
  }
}
</style>