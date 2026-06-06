<script setup lang="ts">
import { onMounted } from "vue";
import { fetchChats } from "@/api/chats";
import { useChatStore } from "@/mini/stores/chat";

const { chats, selectedChat, loading, error, setChats, selectChat } = useChatStore();

function getChatId(chat: any): string {
  return String(chat.chat_id ?? chat.id ?? "");
}

function getChatTitle(chat: any): string {
  return chat.title || `Chat ${getChatId(chat)}`;
}

async function loadChats(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    const data = await fetchChats();
    setChats(data);
    // Auto-select first chat if none selected
    if (!selectedChat.value && data.length > 0) {
      selectChat(data[0]);
    }
  } catch (e: any) {
    error.value = e?.message || "加载群列表失败";
  } finally {
    loading.value = false;
  }
}

function onSelectChat(event: Event): void {
  const select = event.target as HTMLSelectElement;
  const id = select.value;
  if (!id) {
    selectChat(null);
    return;
  }
  const chat = chats.value.find((c) => getChatId(c) === id);
  selectChat(chat || null);
}

onMounted(() => {
  loadChats();
});
</script>

<template>
  <div class="chat-selector">
    <select
      class="select chat-select"
      :value="selectedChat ? getChatId(selectedChat) : ''"
      @change="onSelectChat"
    >
      <option value="" disabled>选择群组...</option>
      <option
        v-for="chat in chats"
        :key="getChatId(chat)"
        :value="getChatId(chat)"
      >
        {{ getChatTitle(chat) }}
      </option>
    </select>
    <div v-if="loading" class="chat-selector-loading">加载中...</div>
  </div>
</template>

<style scoped>
.chat-selector {
  padding: 10px 16px;
  background: var(--tg-bg);
  border-bottom: 1px solid var(--tg-hint);
  flex-shrink: 0;
}

.chat-select {
  font-size: 14px;
}

.chat-selector-loading {
  font-size: 12px;
  color: var(--tg-hint);
  margin-top: 4px;
}
</style>
