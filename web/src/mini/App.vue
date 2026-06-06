<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import ChatSelector from "@/mini/components/ChatSelector.vue";
import { useAuth } from "@/mini/stores/auth";

const router = useRouter();
const route = useRoute();
const { loading: authLoading, error: authError, initAuth } = useAuth();

const scheme = ref<"light" | "dark">("light");

const tabs = [
  { path: "/", label: "仪表盘", emoji: "📊" },
  { path: "/settings", label: "设置", emoji: "⚙️" },
  { path: "/publish", label: "发布", emoji: "📣" },
  { path: "/lottery", label: "抽奖", emoji: "🎁" },
] as const;

function isActive(path: string): boolean {
  return route.path === path;
}

function goTab(path: string): void {
  router.push(path);
}

function applyTelegramTheme(): void {
  const tg = (window as any).Telegram?.WebApp;
  if (!tg) return;

  const colorScheme = tg.colorScheme || "light";
  scheme.value = colorScheme;

  document.documentElement.style.setProperty(
    "--tg-theme-bg-color",
    tg.backgroundColor || tg.themeParams?.bg_color || ""
  );
  document.documentElement.style.setProperty(
    "--tg-theme-text-color",
    tg.themeParams?.text_color || ""
  );
  document.documentElement.style.setProperty(
    "--tg-theme-hint-color",
    tg.themeParams?.hint_color || ""
  );
  document.documentElement.style.setProperty(
    "--tg-theme-link-color",
    tg.themeParams?.link_color || ""
  );
  document.documentElement.style.setProperty(
    "--tg-theme-button-color",
    tg.themeParams?.button_color || ""
  );
  document.documentElement.style.setProperty(
    "--tg-theme-button-text-color",
    tg.themeParams?.button_text_color || ""
  );
  document.documentElement.style.setProperty(
    "--tg-theme-secondary-bg-color",
    tg.themeParams?.secondary_bg_color || tg.secondaryBackgroundColor || ""
  );

  if (!tg.themeParams) {
    document.documentElement.setAttribute("data-tg-theme", "absent");
  }

  document.documentElement.setAttribute("data-color-scheme", colorScheme);
}

function onThemeChanged(): void {
  applyTelegramTheme();
}

// Bind Telegram WebApp
function initTelegram(): void {
  const tg = (window as any).Telegram?.WebApp;
  if (!tg) return;

  try {
    tg.ready();
    tg.expand?.();
  } catch (_) {
    // Some methods may not be available
  }

  applyTelegramTheme();

  try {
    tg.onEvent?.("themeChanged", onThemeChanged);
  } catch (_) {
    // Event binding may fail in non-TG env
  }
}

onMounted(() => {
  initTelegram();
  initAuth();
});

onUnmounted(() => {
  const tg = (window as any).Telegram?.WebApp;
  if (tg?.offEvent) {
    try {
      tg.offEvent("themeChanged", onThemeChanged);
    } catch (_) {
      // ignore
    }
  }
});
</script>

<template>
  <div id="mini-app">
    <!-- Auth loading -->
    <div v-if="authLoading" class="scrollable" style="display: flex; align-items: center; justify-content: center;">
      <div>
        <div class="spinner"></div>
        <p style="text-align: center; color: var(--tg-hint); margin-top: 16px;">正在认证...</p>
      </div>
    </div>

    <!-- Auth error -->
    <div v-else-if="authError" class="scrollable" style="display: flex; align-items: center; justify-content: center;">
      <div style="text-align: center;">
        <div class="empty-icon">⚠️</div>
        <div class="error" style="margin-bottom: 16px;">{{ authError }}</div>
      </div>
    </div>

    <!-- Main app -->
    <template v-else>
      <ChatSelector />
      <div class="scrollable">
        <router-view />
      </div>
      <nav class="tabbar">
        <button
          v-for="tab in tabs"
          :key="tab.path"
          class="tabbar-item"
          :class="{ active: isActive(tab.path) }"
          @click="goTab(tab.path)"
        >
          <span class="tabbar-emoji">{{ tab.emoji }}</span>
          <span class="tabbar-label">{{ tab.label }}</span>
        </button>
      </nav>
    </template>
  </div>
</template>

<style scoped>
.tabbar {
  display: flex;
  background: var(--tg-bg);
  border-top: 1px solid var(--tg-hint);
  padding: 6px 0;
  flex-shrink: 0;
  /* safe area for iOS */
  padding-bottom: calc(6px + env(safe-area-inset-bottom, 0px));
}

.tabbar-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: none;
  background: none;
  cursor: pointer;
  padding: 6px 4px;
  color: var(--tg-hint);
  transition: color 0.2s;
  gap: 2px;
}

.tabbar-item.active {
  color: var(--tg-link);
}

.tabbar-emoji {
  font-size: 22px;
  line-height: 1;
}

.tabbar-label {
  font-size: 10px;
  font-weight: 500;
}
</style>
