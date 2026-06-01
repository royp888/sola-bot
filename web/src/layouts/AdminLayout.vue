<template>
  <div class="shell">
    <aside class="sidebar" :class="{ collapsed }">
      <div class="brand">
        <div class="brand-mark">{{ brandInitial }}</div>
        <div v-if="!collapsed" class="brand-copy">
          <strong>{{ appName }}</strong>
          <span>{{ appDesc }}</span>
        </div>
      </div>

      <el-menu
        class="nav"
        :default-active="route.path"
        :collapse="collapsed"
        :router="true"
        background-color="transparent"
        text-color="var(--app-text)"
        active-text-color="var(--app-accent)"
      >
        <el-menu-item index="/">
          <el-icon><House /></el-icon>
          <template #title>概览</template>
        </el-menu-item>
        <el-menu-item index="/bots">
          <el-icon><Cpu /></el-icon>
          <template #title>Bots</template>
        </el-menu-item>
        <el-menu-item index="/chats">
          <el-icon><ChatDotRound /></el-icon>
          <template #title>Chats</template>
        </el-menu-item>
        <el-sub-menu index="points">
          <template #title>
            <el-icon><Coin /></el-icon>
            <span>积分系统</span>
          </template>
          <el-menu-item index="/users">
            <el-icon><User /></el-icon>
            <template #title>用户管理</template>
          </el-menu-item>
          <el-menu-item index="/points/config">
            <el-icon><Coin /></el-icon>
            <template #title>积分配置</template>
          </el-menu-item>
          <el-menu-item index="/points/logs">
            <el-icon><Document /></el-icon>
            <template #title>积分流水</template>
          </el-menu-item>
        </el-sub-menu>
        <el-sub-menu index="admin">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>群组管理</span>
          </template>
          <el-menu-item index="/admin/config">
            <el-icon><Tickets /></el-icon>
            <template #title>群组配置</template>
          </el-menu-item>
          <el-menu-item index="/admin/bans">
            <el-icon><Lock /></el-icon>
            <template #title>封禁列表</template>
          </el-menu-item>
        </el-sub-menu>
        <el-sub-menu index="rules">
          <template #title>
            <el-icon><Tickets /></el-icon>
            <span>规则治理</span>
          </template>
          <el-menu-item index="/levels">
            <el-icon><TrendCharts /></el-icon>
            <template #title>等级规则</template>
          </el-menu-item>
          <el-menu-item index="/keywords">
            <el-icon><Document /></el-icon>
            <template #title>关键词规则</template>
          </el-menu-item>
          <el-menu-item index="/auto-replies">
            <el-icon><ChatDotRound /></el-icon>
            <template #title>自动回复</template>
          </el-menu-item>
          <el-menu-item index="/violations">
            <el-icon><Lock /></el-icon>
            <template #title>违规记录</template>
          </el-menu-item>
        </el-sub-menu>
        <el-sub-menu index="content">
          <template #title>
            <el-icon><Files /></el-icon>
            <span>内容调度</span>
          </template>
          <el-menu-item index="/posts">
            <el-icon><Calendar /></el-icon>
            <template #title>定时发帖</template>
          </el-menu-item>
          <el-menu-item index="/templates">
            <el-icon><Document /></el-icon>
            <template #title>消息模板</template>
          </el-menu-item>
          <el-menu-item index="/lottery">
            <el-icon><Trophy /></el-icon>
            <template #title>抽奖管理</template>
          </el-menu-item>
        </el-sub-menu>
        <el-sub-menu index="growth">
          <template #title>
            <el-icon><ChatDotRound /></el-icon>
            <span>增长追踪</span>
          </template>
          <el-menu-item index="/invite-links">
            <el-icon><Tickets /></el-icon>
            <template #title>邀请链接</template>
          </el-menu-item>
        </el-sub-menu>
        <el-menu-item index="/stats">
          <el-icon><TrendCharts /></el-icon>
          <template #title>数据统计</template>
        </el-menu-item>
        <el-menu-item index="/backup">
          <el-icon><Files /></el-icon>
          <template #title>备份恢复</template>
        </el-menu-item>
      </el-menu>

      <div v-if="!collapsed" class="sidebar-footer">
        <span class="foot-label">API base</span>
        <strong>{{ apiBase }}</strong>
      </div>
    </aside>

    <div class="content">
      <header class="topbar">
        <div class="topbar-left">
          <el-button text :icon="collapsed ? Expand : Fold" @click="collapsed = !collapsed" />
          <div>
            <div class="top-title">{{ currentTitle }}</div>
            <div class="top-subtitle">运营后台 / 频道 / 群 / 统计</div>
          </div>
        </div>

        <div class="topbar-right">
          <el-tag effect="dark" type="info">REST</el-tag>
          <el-tag effect="dark" type="success">{{ userLabel }}</el-tag>
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
import { computed, ref } from "vue";
import { useRouter, useRoute } from "vue-router";
import {
  ChatDotRound,
  Coin,
  Calendar,
  Document,
  Files,
  Cpu,
  Expand,
  Fold,
  Lock,
  House,
  Setting,
  Tickets,
  Trophy,
  User,
  TrendCharts,
} from "@element-plus/icons-vue";
import { clearSession, getStoredUser } from "@/api/session";

const router = useRouter();
const route = useRoute();
const collapsed = ref(false);
const apiBase = import.meta.env.VITE_API_BASE_URL?.trim() || "/api";
const appName = import.meta.env.VITE_APP_NAME?.trim() || "Sola Bot";
const appDesc = import.meta.env.VITE_APP_DESC?.trim() || "Telegram 运营管理后台";
const brandInitial = computed(() => appName.trim().slice(0, 1).toUpperCase() || "S");

const currentTitle = computed(() => {
  const matched = route.matched
    .slice()
    .reverse()
    .find((record) => typeof record.meta?.title === "string");
  return (matched?.meta?.title as string | undefined) ?? appName;
});

const userLabel = computed(() => getStoredUser()?.name ?? "Operator");

function handleCommand(command: string): void {
  if (command === "logout") {
    clearSession();
    router.push({ name: "login" });
  }
}
</script>

<style scoped>
.shell {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  min-height: 100vh;
}

.sidebar {
  display: flex;
  flex-direction: column;
  width: 252px;
  padding: 18px 14px;
  border-right: 1px solid var(--app-border);
  background:
    linear-gradient(180deg, rgba(18, 24, 32, 0.98), rgba(12, 17, 23, 0.98)),
    radial-gradient(circle at top, rgba(94, 205, 195, 0.08), transparent 28%);
}

.sidebar.collapsed {
  width: 84px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 18px;
}

.brand-mark {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: 8px;
  background: linear-gradient(135deg, rgba(94, 205, 195, 0.25), rgba(240, 179, 93, 0.2));
  color: var(--app-text);
  font-weight: 800;
}

.brand-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.brand-copy strong {
  font-size: 14px;
}

.brand-copy span {
  color: var(--app-muted);
  font-size: 12px;
}

.nav {
  flex: 1;
  border-right: 0;
}

.sidebar-footer {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 18px;
  padding: 14px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
}

.foot-label {
  color: var(--app-muted);
  font-size: 12px;
}

.sidebar-footer strong {
  font-size: 13px;
  font-weight: 600;
}

.content {
  min-width: 0;
  background:
    linear-gradient(180deg, rgba(10, 14, 18, 0.2), rgba(10, 14, 18, 0.08)),
    var(--app-bg);
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
  padding: 0 22px;
  border-bottom: 1px solid var(--app-border);
  backdrop-filter: blur(14px);
  background: rgba(10, 14, 18, 0.78);
}

.topbar-left,
.topbar-right {
  display: flex;
  align-items: center;
  gap: 14px;
}

.top-title {
  font-size: 16px;
  font-weight: 700;
}

.top-subtitle {
  color: var(--app-muted);
  font-size: 12px;
}

.viewport {
  padding: 22px;
}

@media (max-width: 960px) {
  .shell {
    grid-template-columns: 84px minmax(0, 1fr);
  }

  .sidebar {
    width: 84px;
  }
}

@media (max-width: 720px) {
  .shell {
    grid-template-columns: 1fr;
  }

  .sidebar {
    width: auto;
    border-right: 0;
    border-bottom: 1px solid var(--app-border);
  }

  .topbar {
    padding: 0 16px;
  }

  .viewport {
    padding: 16px;
  }
}
</style>
