<template>
  <div class="page">
    <PageHeader eyebrow="Violations" title="违规记录" description="查看关键词、风控和管理员动作产生的违规记录。">
      <template #actions>
        <ChatSelect v-model="filters.chatId" @update:model-value="onChatChanged" />
        <UserSelect v-model="filters.userId" :chat-id="filters.chatId" />
        <el-select v-model="filters.type" class="filter-select" clearable placeholder="类型" @change="loadViolations">
          <el-option label="关键词" value="keyword" />
          <el-option label="刷屏" value="spam" />
          <el-option label="警告" value="warn" />
          <el-option label="禁言" value="mute" />
          <el-option label="封禁" value="ban" />
        </el-select>
        <el-select v-model="filters.status" class="filter-select" clearable placeholder="状态">
          <el-option label="待处理" value="open" />
          <el-option label="已处理" value="resolved" />
          <el-option label="忽略" value="ignored" />
        </el-select>
        <el-button :icon="Refresh" :loading="loading" @click="loadViolations">刷新</el-button>
      </template>
    </PageHeader>

    <PanelSection title="记录列表" description="接口：GET/PATCH /api/admin/violations。">
      <el-alert v-if="error" class="alert" type="error" :closable="false" show-icon title="接口不可用" />
      <el-table :data="violations" stripe v-loading="loading">
        <el-table-column label="用户" min-width="150">
          <template #default="{ row }">
            <strong>{{ row.username || row.user_id }}</strong>
            <div class="muted">{{ row.user_id }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="chat_id" label="Chat" min-width="120" />
        <el-table-column prop="type" label="类型" min-width="120" />
        <el-table-column prop="reason" label="原因" min-width="180" />
        <el-table-column prop="source" label="来源" min-width="120" />
        <el-table-column prop="count" label="次数" width="90" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)" effect="dark">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" min-width="170" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" :loading="updatingId === row.id" @click="openResolve(row)">
              处理
            </el-button>
            <el-button size="small" :loading="updatingId === row.id" @click="markIgnored(row)">
              忽略
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <el-dialog v-model="dialogVisible" title="处理违规记录" width="480px">
      <el-form label-position="top">
        <el-form-item label="状态">
          <el-select v-model="resolveForm.status" class="wide-control">
            <el-option label="待处理" value="open" />
            <el-option label="已处理" value="resolved" />
            <el-option label="忽略" value="ignored" />
          </el-select>
        </el-form-item>
        <el-form-item label="处理说明">
          <el-input v-model="resolveForm.resolution" type="textarea" :rows="4" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="updatingId === currentViolation?.id" @click="submitResolution">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { Refresh } from "@element-plus/icons-vue";
import ChatSelect from "@/components/ChatSelect.vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import UserSelect from "@/components/UserSelect.vue";
import { fetchAdminViolations, updateAdminViolation } from "@/api/violations";
import type { AdminViolationRecord, ChatID } from "@/types/api";

const loading = ref(false);
const error = ref(false);
const dialogVisible = ref(false);
const updatingId = ref<ChatID>();
const violations = ref<AdminViolationRecord[]>([]);
const currentViolation = ref<AdminViolationRecord>();
const filters = reactive({ chatId: "", userId: "", type: "", status: "" });
const resolveForm = reactive({ status: "resolved", resolution: "" });

function statusLabel(status: string): string {
  return (
    {
      open: "待处理",
      pending: "待处理",
      resolved: "已处理",
      ignored: "忽略",
    }[status] ?? status
  );
}

function statusTag(status: string): "success" | "warning" | "info" {
  if (status === "resolved") return "success";
  if (status === "ignored") return "info";
  return "warning";
}

async function loadViolations(): Promise<void> {
  loading.value = true;
  error.value = false;
  try {
    violations.value = await fetchAdminViolations({
      chatId: filters.chatId,
      userId: filters.userId,
      type: filters.type,
      status: filters.status,
    });
  } catch {
    violations.value = [];
    error.value = true;
    ElMessage.error("接口不可用");
  } finally {
    loading.value = false;
  }
}

function onChatChanged(): void {
  filters.userId = "";
  void loadViolations();
}

function openResolve(row: AdminViolationRecord): void {
  currentViolation.value = row;
  resolveForm.status = row.status || "resolved";
  resolveForm.resolution = row.resolution || "";
  dialogVisible.value = true;
}

async function submitResolution(): Promise<void> {
  if (!currentViolation.value) return;
  updatingId.value = currentViolation.value.id;
  try {
    await updateAdminViolation(currentViolation.value.id, {
      status: resolveForm.status,
      resolution: resolveForm.resolution.trim() || undefined,
    });
    ElMessage.success("违规记录已更新");
    dialogVisible.value = false;
    await loadViolations();
  } catch {
    ElMessage.error("接口不可用");
  } finally {
    updatingId.value = undefined;
  }
}

async function markIgnored(row: AdminViolationRecord): Promise<void> {
  updatingId.value = row.id;
  try {
    await updateAdminViolation(row.id, { status: "ignored", resolution: "ignored_by_admin" });
    ElMessage.success("违规记录已忽略");
    await loadViolations();
  } catch {
    ElMessage.error("接口不可用");
  } finally {
    updatingId.value = undefined;
  }
}

onMounted(loadViolations);
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.filter-input,
.filter-select {
  width: 180px;
}

.alert {
  margin-bottom: 12px;
}

.muted {
  color: var(--app-muted);
  font-size: 12px;
}

.wide-control {
  width: 100%;
}

@media (max-width: 720px) {
  .filter-input,
  .filter-select {
    width: 100%;
  }
}
</style>
