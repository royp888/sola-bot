<template>
  <el-select
    :model-value="modelValue"
    class="chat-select"
    allow-create
    filterable
    clearable
    :loading="loading"
    placeholder="选择群组或输入 Chat ID"
    @update:model-value="emit('update:modelValue', String($event ?? ''))"
  >
    <el-option
      v-for="chat in chats"
      :key="chatValue(chat)"
      :label="chatLabel(chat)"
      :value="chatValue(chat)"
    />
  </el-select>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { fetchChats } from "@/api/chats";
import type { ChatID, ChatRecord } from "@/types/api";

const props = defineProps<{
  modelValue: ChatID | "";
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
  loaded: [items: ChatRecord[]];
}>();

const chats = ref<ChatRecord[]>([]);
const loading = ref(false);

function chatValue(chat: ChatRecord): string {
  return String(chat.chat_id ?? chat.id ?? "");
}

function chatType(chat: ChatRecord): string {
  return String(chat.chat_type ?? chat.kind ?? "group");
}

function chatLabel(chat: ChatRecord): string {
  const title = chat.title || chat.username || chatValue(chat);
  return `${title} · ${chatType(chat)} · ${chatValue(chat)}`;
}

async function loadChats(): Promise<void> {
  loading.value = true;
  try {
    chats.value = await fetchChats();
    emit("loaded", chats.value);
    if (!props.modelValue && chats.value[0]) {
      emit("update:modelValue", chatValue(chats.value[0]));
    }
  } catch {
    chats.value = [];
    ElMessage.error("群组列表接口不可用");
  } finally {
    loading.value = false;
  }
}

watch(
  () => props.modelValue,
  (value) => {
    if (!value && chats.value[0]) {
      emit("update:modelValue", chatValue(chats.value[0]));
    }
  },
);

defineExpose({ loadChats });

onMounted(loadChats);
</script>

<style scoped>
.chat-select {
  width: 280px;
}

@media (max-width: 720px) {
  .chat-select {
    width: 100%;
  }
}
</style>
