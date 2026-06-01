<template>
  <div class="page">
    <PageHeader eyebrow="Private Ops" title="私聊/用户运营" description="按群组查看用户积分排行，并处理手动加减分。">
      <template #actions>
        <el-input v-model="keyword" class="search-input" clearable placeholder="搜索用户名 / 昵称" />
        <ChatSelect v-model="selectedChatId" @loaded="onChatsLoaded" />
        <el-button :icon="Refresh" :loading="loading" @click="loadUsers">刷新</el-button>
      </template>
    </PageHeader>

    <PanelSection title="用户列表" description="支持成员 CSV 导出、批量封禁和批量调分。">
      <template #actions>
        <el-button :disabled="!selectedChatId" @click="downloadCsv">导出 CSV</el-button>
        <el-button :disabled="selectedRows.length === 0" type="warning" @click="openBatchAdjust">批量调分</el-button>
        <el-button :disabled="selectedRows.length === 0" type="danger" @click="submitBatchBan">批量封禁</el-button>
      </template>
      <el-table :data="filteredUsers" stripe @selection-change="selectedRows = $event">
        <el-table-column type="selection" width="48" />
        <el-table-column prop="username" label="用户" min-width="150">
          <template #default="{ row }">
            <strong>{{ row.display_name }}</strong>
            <div class="muted">{{ row.username }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="chat_id" label="Chat" min-width="120" />
        <el-table-column prop="total_points" label="积分" width="120" sortable />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)" effect="dark">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_seen_at" label="最后活跃" min-width="150" />
        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <el-button size="small" :icon="Coin" @click="openAdjust(row)">调分</el-button>
            <el-button size="small" type="danger" @click="openBan(row)">封禁</el-button>
            <el-button size="small" type="warning" @click="openMute(row)">禁言</el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <el-dialog v-model="adjustVisible" title="手动加减分" width="420px">
      <el-form label-position="top">
        <el-form-item label="用户">
          <el-input :model-value="currentUser?.username || ''" disabled />
        </el-form-item>
        <el-form-item label="积分变化">
          <el-input-number v-model="adjustForm.delta" class="wide-control" :min="-999999" :max="999999" />
        </el-form-item>
        <el-form-item label="原因">
          <el-input v-model="adjustForm.reason" maxlength="64" placeholder="manual_adjust" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="adjustVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitAdjust">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="batchVisible" title="批量调分" width="420px">
      <el-form label-position="top">
        <el-form-item label="已选用户">
          <el-input :model-value="`${selectedRows.length} 人`" disabled />
        </el-form-item>
        <el-form-item label="积分变化">
          <el-input-number v-model="batchForm.delta" class="wide-control" :min="-999999" :max="999999" />
        </el-form-item>
        <el-form-item label="原因">
          <el-input v-model="batchForm.reason" maxlength="64" placeholder="batch_adjust" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitBatchAdjust">执行</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Coin, Refresh } from "@element-plus/icons-vue";
import ChatSelect from "@/components/ChatSelect.vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { createBan } from "@/api/admin";
import { updateUserPoints } from "@/api/points";
import { batchUsers, exportUsersCsv, fetchUsers } from "@/api/users";
import type { ChatRecord, UserRecord } from "@/types/api";

const keyword = ref("");
const selectedChatId = ref("");
const loading = ref(false);
const saving = ref(false);
const adjustVisible = ref(false);
const batchVisible = ref(false);
const users = ref<UserRecord[]>([]);
const chats = ref<ChatRecord[]>([]);
const selectedRows = ref<UserRecord[]>([]);
const currentUser = ref<UserRecord>();
const adjustForm = reactive({ delta: 10, reason: "manual_adjust" });
const batchForm = reactive({ delta: 10, reason: "batch_adjust" });

const filteredUsers = computed(() => {
  const term = keyword.value.trim().toLowerCase();
  return users.value.filter((user) => {
    const matchesChat = selectedChatId.value ? String(user.chat_id) === String(selectedChatId.value) : true;
    const matchesKeyword = term
      ? `${user.username} ${user.display_name}`.toLowerCase().includes(term)
      : true;
    return matchesChat && matchesKeyword;
  });
});

function onChatsLoaded(items: ChatRecord[]): void {
  chats.value = items;
  if (!selectedChatId.value && items[0]) {
    selectedChatId.value = String(items[0].chat_id ?? items[0].id ?? "");
    void loadUsers();
  }
}

async function loadUsers(): Promise<void> {
  if (!selectedChatId.value) {
    users.value = [];
    ElMessage.warning("请先选择或输入 Chat ID");
    return;
  }
  loading.value = true;
  try {
    users.value = await fetchUsers({ keyword: keyword.value, chatId: selectedChatId.value });
    selectedRows.value = [];
  } catch {
    users.value = [];
    ElMessage.error("接口不可用");
  } finally {
    loading.value = false;
  }
}

async function downloadCsv(): Promise<void> {
  if (!selectedChatId.value) {
    ElMessage.warning("请先选择或输入 Chat ID");
    return;
  }
  try {
    const blob = await exportUsersCsv({ keyword: keyword.value, chatId: selectedChatId.value });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `sola-users-${selectedChatId.value}-${new Date().toISOString().slice(0, 10)}.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
    ElMessage.success("CSV 已导出");
  } catch {
    ElMessage.error("导出接口不可用");
  }
}

function openBatchAdjust(): void {
  if (selectedRows.value.length === 0) return;
  batchForm.delta = 10;
  batchForm.reason = "batch_adjust";
  batchVisible.value = true;
}

async function submitBatchAdjust(): Promise<void> {
  if (!selectedChatId.value || selectedRows.value.length === 0) return;
  saving.value = true;
  try {
    const result = await batchUsers({
      chat_id: selectedChatId.value,
      user_ids: selectedRows.value.map((user) => user.id),
      action: "adjust_points",
      delta: batchForm.delta,
      reason: batchForm.reason,
    });
    ElMessage.success(`已处理 ${result.success_count} 人`);
    batchVisible.value = false;
    await loadUsers();
  } catch {
    ElMessage.error("批量调分接口不可用");
  } finally {
    saving.value = false;
  }
}

async function submitBatchBan(): Promise<void> {
  if (!selectedChatId.value || selectedRows.value.length === 0) return;
  try {
    await ElMessageBox.confirm(`确认封禁选中的 ${selectedRows.value.length} 个用户？`, "批量封禁", {
      type: "warning",
      confirmButtonText: "确认封禁",
      cancelButtonText: "取消",
    });
  } catch {
    return;
  }
  saving.value = true;
  try {
    const result = await batchUsers({
      chat_id: selectedChatId.value,
      user_ids: selectedRows.value.map((user) => user.id),
      action: "ban",
      reason: "batch_ban_from_users_view",
    });
    ElMessage.success(`已封禁 ${result.success_count} 人`);
    await loadUsers();
  } catch {
    ElMessage.error("批量封禁接口不可用");
  } finally {
    saving.value = false;
  }
}

function openAdjust(user: UserRecord): void {
  currentUser.value = user;
  adjustForm.delta = 10;
  adjustForm.reason = "manual_adjust";
  adjustVisible.value = true;
}

async function submitAdjust(): Promise<void> {
  if (!currentUser.value) return;
  saving.value = true;
  try {
    await updateUserPoints(currentUser.value.chat_id, currentUser.value.id, { ...adjustForm });
    ElMessage.success("积分已更新");
    adjustVisible.value = false;
    await loadUsers();
  } catch {
    ElMessage.error("接口不可用");
  } finally {
    saving.value = false;
  }
}

async function openBan(user: UserRecord): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `确认封禁用户 ${user.username || user.id}？此操作需手动解封。`,
      "确认封禁",
      {
        type: "warning",
        confirmButtonText: "确认封禁",
        cancelButtonText: "取消",
      },
    );
  } catch {
    return;
  }
  saving.value = true;
  try {
    await createBan({
      chat_id: user.chat_id,
      user_id: user.id,
      reason: "ban_from_users_view",
    });
    ElMessage.success("封禁请求已提交");
    await loadUsers();
  } catch {
    ElMessage.error("接口不可用");
  } finally {
    saving.value = false;
  }
}

function openMute(user: UserRecord): void {
  ElMessageBox.alert(
    `后台暂未开放真实禁言接口。请在群内回复该用户消息后使用 /mute 30m，或使用 /mute ${user.id} 30m。`,
    "禁言入口",
    {
      confirmButtonText: "知道了",
      type: "warning",
    },
  );
}

function statusLabel(value: UserRecord["status"]): string {
  return { active: "正常", muted: "禁言", banned: "封禁" }[value];
}

function statusTag(value: UserRecord["status"]): "success" | "warning" | "danger" {
  if (value === "active") return "success";
  if (value === "muted") return "warning";
  return "danger";
}

onMounted(async () => {
  if (selectedChatId.value) {
    await loadUsers();
  }
});
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.search-input {
  width: 220px;
}

.muted {
  color: var(--app-muted);
  font-size: 12px;
}

.wide-control {
  width: 100%;
}

@media (max-width: 720px) {
  .search-input {
    width: 100%;
  }
}
</style>
