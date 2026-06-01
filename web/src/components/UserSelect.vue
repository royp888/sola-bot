<template>
  <el-select
    :model-value="modelValue"
    class="user-select"
    allow-create
    filterable
    clearable
    :disabled="!chatId"
    :loading="loading"
    placeholder="选择用户或输入用户 ID"
    @update:model-value="emit('update:modelValue', String($event ?? ''))"
  >
    <el-option
      v-for="user in users"
      :key="userValue(user)"
      :label="userLabel(user)"
      :value="userValue(user)"
    />
  </el-select>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { fetchUsers } from "@/api/users";
import type { ChatID, UserRecord } from "@/types/api";

const props = defineProps<{
  modelValue: ChatID | "";
  chatId: ChatID | "";
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
  loaded: [items: UserRecord[]];
}>();

const users = ref<UserRecord[]>([]);
const loading = ref(false);

function userValue(user: UserRecord): string {
  return String(user.id);
}

function userLabel(user: UserRecord): string {
  const name = user.display_name || user.username || userValue(user);
  return `${name} · ${userValue(user)} · ${user.total_points} 分`;
}

async function loadUsers(): Promise<void> {
  if (!props.chatId) {
    users.value = [];
    return;
  }
  loading.value = true;
  try {
    users.value = await fetchUsers({ chatId: props.chatId, limit: 100 });
    emit("loaded", users.value);
  } catch {
    users.value = [];
    ElMessage.error("用户列表接口不可用");
  } finally {
    loading.value = false;
  }
}

watch(
  () => props.chatId,
  () => {
    emit("update:modelValue", "");
    void loadUsers();
  },
  { immediate: true },
);

defineExpose({ loadUsers });
</script>

<style scoped>
.user-select {
  width: 220px;
}

@media (max-width: 720px) {
  .user-select {
    width: 100%;
  }
}
</style>
