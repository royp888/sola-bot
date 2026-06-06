<script setup lang="ts">
import { ref, watch, onMounted } from "vue";
import { fetchAdminConfig, updateAdminConfig } from "@/api/admin";
import { fetchChats } from "@/api/chats";
import { useChatStore } from "@/mini/stores/chat";
import type { ChatAdminConfig, ChatRecord } from "@/types/api";

const { selectedChat } = useChatStore();

const config = ref<ChatAdminConfig | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);
const saving = ref(false);
const saveError = ref<string | null>(null);
const chatList = ref<ChatRecord[]>([]);

const form = ref({
  verify_enabled: false,
  points_enabled: false,
});

function getChatId(chat: ChatRecord): string {
  return String(chat.chat_id ?? chat.id ?? "");
}

function getChatTitle(chat: ChatRecord): string {
  return chat.title || `Chat ${getChatId(chat)}`;
}

async function loadChatList(): Promise<void> {
  try {
    chatList.value = await fetchChats();
  } catch (_) {
    // ignore
  }
}

async function loadConfig(): Promise<void> {
  const chat = selectedChat.value;
  if (!chat) {
    config.value = null;
    return;
  }

  const chatId = chat.chat_id ?? chat.id;
  loading.value = true;
  error.value = null;

  try {
    const resp = await fetchAdminConfig(chatId!);
    config.value = resp;
    form.value.verify_enabled = resp.verify_enabled ?? false;
    // The config may have a points_enabled; fall back to false
    form.value.points_enabled = (resp as any).points_enabled ?? false;
  } catch (e: any) {
    error.value = e?.message || "加载群配置失败";
    config.value = null;
  } finally {
    loading.value = false;
  }
}

async function saveConfig(): Promise<void> {
  const chat = selectedChat.value;
  if (!chat) return;

  const chatId = chat.chat_id ?? chat.id;
  saving.value = true;
  saveError.value = null;

  try {
    const payload: any = {
      verify_enabled: form.value.verify_enabled,
      points_enabled: form.value.points_enabled,
    };
    // Merge existing config to satisfy the full payload requirement
    if (config.value) {
      payload.welcome_text = config.value.welcome_text ?? "";
      payload.verify_timeout = config.value.verify_timeout ?? 300;
      payload.warn_limit = config.value.warn_limit ?? 3;
    }
    const updated = await updateAdminConfig(chatId!, payload);
    config.value = updated;
    form.value.verify_enabled = updated.verify_enabled ?? false;
    form.value.points_enabled = (updated as any).points_enabled ?? false;
  } catch (e: any) {
    saveError.value = e?.message || "保存失败";
  } finally {
    saving.value = false;
  }
}

function onSelectChat(event: Event): void {
  const select = event.target as HTMLSelectElement;
  const id = select.value;
  if (!id) return;
  const chat = chatList.value.find((c) => getChatId(c) === id);
  if (chat) {
    const { selectChat } = useChatStore();
    selectChat(chat);
  }
}

watch(selectedChat, () => {
  loadConfig();
});

onMounted(() => {
  loadChatList();
  if (selectedChat.value) {
    loadConfig();
  }
});
</script>

<template>
  <div class="scrollable">
    <div class="form-group" style="margin-bottom: 16px;">
      <label style="font-size: 13px; color: var(--tg-hint); display: block; margin-bottom: 6px;">目标群组</label>
      <select
        class="select"
        :value="selectedChat ? String(selectedChat.chat_id ?? (selectedChat as any).id ?? '') : ''"
        @change="onSelectChat"
      >
        <option value="" disabled>选择群组...</option>
        <option
          v-for="chat in chatList"
          :key="getChatId(chat)"
          :value="getChatId(chat)"
        >
          {{ getChatTitle(chat) }}
        </option>
      </select>
    </div>

    <div v-if="!selectedChat" class="empty">
      <div class="empty-icon">⚙️</div>
      <p>请选择一个群组</p>
    </div>

    <div v-else-if="loading" class="spinner"></div>

    <div v-else-if="error" class="error">{{ error }}</div>

    <div v-else-if="config">
      <div class="card">
        <div class="card-title">当前配置</div>
        <div style="font-size: 14px; color: var(--tg-hint); margin-top: 4px;">
          验证类型: {{ config.verify_enabled ? '已启用' : '已禁用' }}
          <template v-if="config.verify_timeout"> · 超时: {{ config.verify_timeout }}s</template>
        </div>
        <div style="font-size: 14px; color: var(--tg-hint); margin-top: 2px;">
          警告次数限制: {{ config.warn_limit ?? '-' }}
        </div>
        <div style="font-size: 14px; color: var(--tg-hint); margin-top: 2px;">
          积分规则: {{ (config as any).points_enabled ? '已启用' : '未设置' }}
        </div>
      </div>

      <div class="card">
        <div class="card-title">开关设置</div>
        <div class="toggle-row">
          <span class="toggle-label">验证开关</span>
          <label class="toggle">
            <input type="checkbox" v-model="form.verify_enabled" />
            <span class="toggle-slider"></span>
          </label>
        </div>
        <div class="toggle-row">
          <span class="toggle-label">积分开关</span>
          <label class="toggle">
            <input type="checkbox" v-model="form.points_enabled" />
            <span class="toggle-slider"></span>
          </label>
        </div>
      </div>

      <div v-if="saveError" class="error">{{ saveError }}</div>

      <button class="btn btn-block" :disabled="saving" @click="saveConfig">
        {{ saving ? '保存中...' : '保存设置' }}
      </button>
    </div>

    <div v-else class="empty">
      <div class="empty-icon">⚙️</div>
      <p>暂无配置数据</p>
    </div>
  </div>
</template>
