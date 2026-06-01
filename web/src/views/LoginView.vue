<template>
  <div class="login-shell">
    <section class="brand-panel">
      <p class="eyebrow">Telegram multi-tenant admin</p>
      <h1>{{ appName }}</h1>
      <p class="lede">用 Telegram 账号登录，绑定自己的群后只管理自己的数据。</p>

      <div class="grid">
        <div class="tile">
          <span>登录方式</span>
          <strong>Telegram Login Widget</strong>
        </div>
        <div class="tile">
          <span>Bot</span>
          <strong>@{{ botUsername || "请配置 VITE_BOT_USERNAME" }}</strong>
        </div>
      </div>
    </section>

    <section class="form-panel">
      <el-card shadow="never" class="login-card">
        <div class="card-head">
          <h2>登录后台</h2>
          <p>群主用 Telegram 一键登录；首次登录会自动注册。</p>
        </div>

        <div v-if="botUsername" ref="tgLoginRef" class="telegram-login"></div>
        <el-alert
          v-else
          type="warning"
          show-icon
          :closable="false"
          title="请在前端环境变量中配置 VITE_BOT_USERNAME，不带 @"
        />

        <el-divider>平台超管</el-divider>
        <el-form ref="formRef" :model="form" :rules="rules" label-position="top">
          <el-form-item label="账号" prop="email">
            <el-input v-model="form.email" placeholder="admin" autocomplete="username" />
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="请输入密码"
              autocomplete="current-password"
              show-password
              @keyup.enter="submitPasswordLogin"
            />
          </el-form-item>
          <el-button type="primary" class="submit-btn" :loading="loading" @click="submitPasswordLogin">
            超管登录
          </el-button>
        </el-form>
      </el-card>
    </section>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage, type FormInstance, type FormRules } from "element-plus";
import { login, telegramLogin } from "@/api/auth";
import { setSession } from "@/api/session";
import type { TelegramLoginPayload } from "@/types/api";

declare global {
  interface Window {
    onTelegramAuth?: (user: TelegramLoginPayload) => void;
  }
}

const router = useRouter();
const route = useRoute();
const formRef = ref<FormInstance>();
const tgLoginRef = ref<HTMLDivElement>();
const loading = ref(false);
const appName = import.meta.env.VITE_APP_NAME?.trim() || "Sola Bot";
const botUsername = (import.meta.env.VITE_BOT_USERNAME as string | undefined)?.replace(/^@/, "").trim() || "";

const form = reactive({
  email: "admin",
  password: "change-me",
});

const rules: FormRules<typeof form> = {
  email: [{ required: true, message: "请输入账号", trigger: "blur" }],
  password: [{ required: true, message: "请输入密码", trigger: "blur" }],
};

function redirectPath(): string {
  return typeof route.query.redirect === "string" ? route.query.redirect : "/";
}

async function finishLogin(response: { accessToken: string; user: Parameters<typeof setSession>[1] }): Promise<void> {
  setSession(response.accessToken, response.user);
  await router.push(redirectPath());
}

async function submitPasswordLogin(): Promise<void> {
  if (!formRef.value) return;
  const valid = await formRef.value.validate().catch(() => false);
  if (!valid) return;

  loading.value = true;
  try {
    await finishLogin(await login({ email: form.email, password: form.password }));
  } catch {
    ElMessage.error("超管登录失败，请检查账号密码");
  } finally {
    loading.value = false;
  }
}

function mountTelegramWidget(): void {
  if (!botUsername || !tgLoginRef.value) return;
  tgLoginRef.value.innerHTML = "";
  const script = document.createElement("script");
  script.src = "https://telegram.org/js/telegram-widget.js?22";
  script.async = true;
  script.setAttribute("data-telegram-login", botUsername);
  script.setAttribute("data-size", "large");
  script.setAttribute("data-onauth", "onTelegramAuth(user)");
  script.setAttribute("data-request-access", "write");
  tgLoginRef.value.appendChild(script);
}

onMounted(async () => {
  window.onTelegramAuth = async (user: TelegramLoginPayload) => {
    loading.value = true;
    try {
      await finishLogin(await telegramLogin(user));
    } catch {
      ElMessage.error("Telegram 登录验证失败，请确认 BotFather 已设置后台域名");
    } finally {
      loading.value = false;
    }
  };
  await nextTick();
  mountTelegramWidget();
});

onBeforeUnmount(() => {
  delete window.onTelegramAuth;
});
</script>

<style scoped>
.login-shell {
  display: grid;
  grid-template-columns: 1.1fr 0.9fr;
  min-height: 100vh;
  background:
    radial-gradient(circle at top left, rgba(94, 205, 195, 0.14), transparent 30%),
    radial-gradient(circle at bottom right, rgba(240, 179, 93, 0.12), transparent 28%),
    var(--app-bg);
}

.brand-panel,
.form-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}

.brand-panel {
  align-items: flex-start;
  flex-direction: column;
  gap: 20px;
}

.eyebrow {
  margin: 0;
  color: var(--app-accent);
  font-size: 12px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  font-size: clamp(34px, 6vw, 58px);
  line-height: 1;
}

.lede {
  max-width: 560px;
  margin: 0;
  color: var(--app-muted);
  font-size: 16px;
  line-height: 1.7;
}

.grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  width: min(560px, 100%);
}

.tile {
  padding: 16px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
}

.tile span {
  display: block;
  color: var(--app-muted);
  font-size: 12px;
}

.tile strong {
  display: block;
  margin-top: 8px;
  font-size: 14px;
}

.login-card {
  width: min(440px, 100%);
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: rgba(14, 19, 26, 0.92);
}

.card-head {
  margin-bottom: 18px;
}

.card-head h2 {
  margin: 0;
  font-size: 22px;
}

.card-head p {
  margin: 8px 0 0;
  color: var(--app-muted);
  line-height: 1.6;
}

.telegram-login {
  min-height: 48px;
}

.submit-btn {
  width: 100%;
  margin-top: 6px;
}

@media (max-width: 960px) {
  .login-shell {
    grid-template-columns: 1fr;
  }

  .brand-panel,
  .form-panel {
    padding: 24px;
  }
}
</style>
