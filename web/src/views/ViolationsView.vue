<template>
  <div class="page-stack">
    <PageHeader eyebrow="Violations" title="违规处理" description="查看命中记录、处理状态和跨群风控动作。">
      <template #actions>
        <el-button :icon="Refresh" :loading="loading" @click="loadViolations">刷新</el-button>
      </template>
    </PageHeader>

    <div class="summary-grid">
      <div class="summary-card">
        <div class="summary-label">当前结果</div>
        <div class="summary-value">{{ violations.length }}</div>
        <div class="summary-meta">匹配当前筛选条件</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">待处理</div>
        <div class="summary-value">{{ statusCounts.open }}</div>
        <div class="summary-meta">需要人工确认或处理</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">已处理</div>
        <div class="summary-value">{{ statusCounts.resolved }}</div>
        <div class="summary-meta">已完成闭环</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">忽略记录</div>
        <div class="summary-value">{{ statusCounts.ignored }}</div>
        <div class="summary-meta">{{ nextCursor ? '还有更多结果可加载' : '当前页已加载完成' }}</div>
      </div>
    </div>

    <PanelSection title="违规列表" description="支持按群组、用户、类型和状态快速缩小范围。">
      <template #actions>
        <div class="panel-toolbar">
          <div class="control-cluster filters">
            <div class="filter-control filter-control-wide">
              <ChatSelect v-model="filters.chatId" @update:model-value="onChatChanged" />
            </div>
            <div class="filter-control filter-control-wide">
              <UserSelect v-model="filters.userId" :chat-id="filters.chatId" @update:model-value="loadViolations" />
            </div>
            <el-select v-model="filters.type" class="filter-control" clearable placeholder="类型" @change="loadViolations">
              <el-option label="关键词" value="keyword" />
              <el-option label="刷屏" value="spam" />
              <el-option label="警告" value="warn" />
              <el-option label="禁言" value="mute" />
              <el-option label="封禁" value="ban" />
            </el-select>
            <el-select v-model="filters.status" class="filter-control" clearable placeholder="状态" @change="loadViolations">
              <el-option label="待处理" value="open" />
              <el-option label="已处理" value="resolved" />
              <el-option label="忽略" value="ignored" />
            </el-select>
          </div>
          <div class="filter-summary">
            <span>群组 {{ filters.chatId || '全部' }}</span>
            <span>用户 {{ filters.userId || '全部' }}</span>
            <span>状态 {{ filters.status || '全部' }}</span>
          </div>
        </div>
      </template>

      <el-alert v-if="error" class="alert" type="error" :closable="false" show-icon title="接口不可用" />
      <el-table class="table-compact" :data="violations" size="small" stripe v-loading="loading">
        <el-table-column label="用户" min-width="170">
          <template #default="{ row }">
            <strong>{{ row.username || row.user_id }}</strong>
            <div class="muted">{{ row.user_id }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="chat_id" label="Chat" min-width="120" />
        <el-table-column prop="type" label="类型" min-width="110" />
        <el-table-column prop="reason" label="原因" min-width="180" />
        <el-table-column prop="source" label="来源" min-width="120" />
        <el-table-column prop="count" label="次数" width="90" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)" effect="dark">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" min-width="170" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" :loading="updatingId === row.id" @click="openResolve(row)">处理</el-button>
            <el-button size="small" type="info" :loading="updatingId === row.id" @click="markIgnored(row)">忽略</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="nextCursor" class="load-more">
        <el-button :loading="loadingMore" @click="loadMoreViolations">加载更多</el-button>
      </div>
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
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { Refresh } from "@element-plus/icons-vue";
import ChatSelect from "@/components/ChatSelect.vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import UserSelect from "@/components/UserSelect.vue";
import { fetchAdminViolations, updateAdminViolation } from "@/api/violations";
import type { AdminViolationRecord, ChatID } from "@/types/api";

const loading = ref(false);
const loadingMore = ref(false);
const error = ref(false);
const dialogVisible = ref(false);
const updatingId = ref<ChatID>();
const violations = ref<AdminViolationRecord[]>([]);
const currentViolation = ref<AdminViolationRecord>();
const nextCursor = ref("");
const filters = reactive({ chatId: "", userId: "", type: "", status: "" });
const resolveForm = reactive({ status: "resolved", resolution: "" });

const statusCounts = computed(() => {
  return violations.value.reduce(
    (acc, item) => {
      const key = item.status === "resolved" || item.status === "ignored" ? item.status : "open";
      acc[key] += 1;
      return acc;
    },
    { open: 0, resolved: 0, ignored: 0 },
  );
});

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
  nextCursor.value = "";
  try {
    const response = await fetchAdminViolations({
      chatId: filters.chatId,
      userId: filters.userId,
      type: filters.type,
      status: filters.status,
    });
    violations.value = response.items;
    nextCursor.value = response.next_cursor || "";
  } catch {
    violations.value = [];
    nextCursor.value = "";
    error.value = true;
    ElMessage.error("接口不可用");
  } finally {
    loading.value = false;
  }
}

async function loadMoreViolations(): Promise<void> {
  if (!nextCursor.value) return;
  loadingMore.value = true;
  try {
    const response = await fetchAdminViolations({
      chatId: filters.chatId,
      userId: filters.userId,
      type: filters.type,
      status: filters.status,
      cursor: nextCursor.value,
    });
    violations.value = violations.value.concat(response.items);
    nextCursor.value = response.next_cursor || "";
  } catch {
    ElMessage.error("更多记录加载失败");
  } finally {
    loadingMore.value = false;
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
.panel-toolbar {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: min(100%, 1040px);
}

.filters :deep(.chat-select),
.filters :deep(.user-select) {
  width: 100%;
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

.load-more {
  display: flex;
  justify-content: center;
  margin-top: 16px;
}
</style>
