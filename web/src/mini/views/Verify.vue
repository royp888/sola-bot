<template>
  <div class="verify-page">
    <div v-if="state === 'loading'" class="state-card">
      <div class="spinner"></div>
      <p>正在加载验证组件…</p>
    </div>

    <div v-else-if="state === 'invalid'" class="state-card error">
      <div class="icon">⚠️</div>
      <h2>链接无效或已过期</h2>
      <p>请重新向机器人申请入群。</p>
    </div>

    <div v-else-if="state === 'pending' || state === 'verifying'" class="state-card">
      <h2>入群验证</h2>
      <p class="hint">请完成下方人机验证，通过后将自动批准您的入群申请。</p>
      <div id="cf-turnstile-widget"></div>
      <div v-if="state === 'verifying'" class="verifying-hint">验证中，请稍候…</div>
    </div>

    <div v-else-if="state === 'success'" class="state-card success">
      <div class="icon">✅</div>
      <h2>验证通过</h2>
      <p>已批准您的入群申请，请返回 Telegram 群组。</p>
    </div>

    <div v-else-if="state === 'failed'" class="state-card error">
      <div class="icon">❌</div>
      <h2>验证失败</h2>
      <p>{{ errorMessage }}</p>
      <button class="retry-btn" @click="resetWidget">重试</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { useRoute } from "vue-router";

type State = "loading" | "invalid" | "pending" | "verifying" | "success" | "failed";

const route = useRoute();
const state = ref<State>("loading");
const errorMessage = ref("验证未通过，请重试或重新申请入群。");

let widgetId: string | undefined;
let siteKey = import.meta.env.VITE_TURNSTILE_SITE_KEY as string | undefined;

const chatId = Number(route.query.chat);
const userId = Number(route.query.user);
const sig = (route.query.sig as string) || "";
const exp = Number(route.query.exp);

function isParamsValid(): boolean {
  return Boolean(chatId && userId && sig && exp && Date.now() / 1000 < exp);
}

async function fetchSiteKey(): Promise<void> {
  try {
    const resp = await fetch("/api/verify/turnstile/config");
    if (resp.ok) {
      const data = (await resp.json()) as { site_key?: string };
      if (data.site_key) siteKey = data.site_key;
    }
  } catch {
    // fall through to use VITE_TURNSTILE_SITE_KEY if set
  }
}

function loadTurnstileScript(): Promise<void> {
  return new Promise((resolve, reject) => {
    if ((window as any).turnstile) {
      resolve();
      return;
    }
    const script = document.createElement("script");
    script.src = "https://challenges.cloudflare.com/turnstile/v0/api.js";
    script.async = true;
    script.defer = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error("failed to load turnstile script"));
    document.head.appendChild(script);
  });
}

function renderWidget(key: string): void {
  const ts = (window as any).turnstile;
  if (!ts) return;
  widgetId = ts.render("#cf-turnstile-widget", {
    sitekey: key,
    callback: onTurnstileToken,
    "error-callback": () => {
      state.value = "failed";
      errorMessage.value = "Turnstile 验证组件出错，请刷新重试。";
    },
    "expired-callback": () => {
      state.value = "pending";
      ts.reset(widgetId);
    },
  });
}

function resetWidget(): void {
  state.value = "pending";
  errorMessage.value = "验证未通过，请重试或重新申请入群。";
  const ts = (window as any).turnstile;
  if (ts && widgetId !== undefined) {
    ts.reset(widgetId);
  }
}

async function onTurnstileToken(token: string): Promise<void> {
  state.value = "verifying";
  try {
    const resp = await fetch("/api/verify/turnstile", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        chat_id: chatId,
        user_id: userId,
        sig,
        exp,
        cf_token: token,
      }),
    });
    const data = (await resp.json()) as { ok?: boolean; message?: string; error?: string };
    if (resp.ok && data.ok) {
      state.value = "success";
    } else {
      state.value = "failed";
      errorMessage.value = data.error || data.message || "验证未通过，请重新申请入群。";
    }
  } catch {
    state.value = "failed";
    errorMessage.value = "网络错误，请稍后重试。";
  }
}

onUnmounted(() => {
  const ts = (window as any).turnstile;
  if (ts && widgetId !== undefined) {
    ts.remove(widgetId);
  }
});

onMounted(async () => {
  if (!isParamsValid()) {
    state.value = "invalid";
    return;
  }

  await fetchSiteKey();

  if (!siteKey) {
    state.value = "failed";
    errorMessage.value = "验证服务未配置，请联系管理员。";
    return;
  }

  try {
    await loadTurnstileScript();
    state.value = "pending";
    renderWidget(siteKey);
  } catch {
    state.value = "failed";
    errorMessage.value = "验证脚本加载失败，请检查网络后刷新重试。";
  }
});
</script>

<style scoped>
.verify-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  padding: 24px 16px;
  background: var(--tg-theme-bg-color, #ffffff);
  color: var(--tg-theme-text-color, #000000);
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

.state-card {
  text-align: center;
  max-width: 360px;
  width: 100%;
}

.state-card h2 {
  font-size: 1.2rem;
  margin-bottom: 8px;
}

.state-card p {
  color: var(--tg-theme-hint-color, #6b7280);
  margin-bottom: 16px;
}

.hint {
  font-size: 0.9rem;
}

.icon {
  font-size: 3rem;
  margin-bottom: 12px;
}

.spinner {
  width: 36px;
  height: 36px;
  border: 3px solid var(--tg-theme-hint-color, #d1d5db);
  border-top-color: var(--tg-theme-button-color, #2563eb);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin: 0 auto 16px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

#cf-turnstile-widget {
  display: flex;
  justify-content: center;
  margin: 16px 0;
}

.verifying-hint {
  font-size: 0.85rem;
  color: var(--tg-theme-hint-color, #6b7280);
}

.retry-btn {
  margin-top: 12px;
  padding: 8px 24px;
  background: var(--tg-theme-button-color, #2563eb);
  color: var(--tg-theme-button-text-color, #ffffff);
  border: none;
  border-radius: 8px;
  font-size: 0.95rem;
  cursor: pointer;
}

.retry-btn:hover {
  opacity: 0.9;
}
</style>
