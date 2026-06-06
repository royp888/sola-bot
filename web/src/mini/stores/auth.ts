import { ref, type Ref } from "vue";
import { request } from "@/api/http";
import { setSession, hasSession } from "@/api/session";

interface AuthState {
  token: Ref<string | null>;
  loading: Ref<boolean>;
  error: Ref<string | null>;
  initAuth: () => Promise<void>;
}

const token = ref<string | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);

function getTelegramInitData(): string {
  if (typeof window === "undefined") return "";

  const tg = (window as any).Telegram;
  if (tg?.WebApp?.initData) {
    return tg.WebApp.initData;
  }

  return "";
}

export function useAuth(): AuthState {
  async function initAuth(): Promise<void> {
    if (hasSession()) {
      token.value = "***";
      return;
    }

    const initData = getTelegramInitData();
    if (!initData) {
      error.value = "无法获取 Telegram 认证数据，请在 Telegram 客户端中打开此应用。";
      return;
    }

    loading.value = true;
    error.value = null;

    try {
      const resp = await request<{ accessToken: string; user: any }>("/auth/telegram", {
        method: "POST",
        body: { init_data: initData },
      });

      const accessToken = resp.accessToken || resp["access_token"];
      if (accessToken && resp.user) {
        setSession(accessToken, resp.user);
        token.value = "***";
      } else {
        error.value = "认证失败：服务端返回数据不完整。";
      }
    } catch (e: any) {
      error.value = e?.message || "认证请求失败，请重试。";
    } finally {
      loading.value = false;
    }
  }

  return { token, loading, error, initAuth };
}
