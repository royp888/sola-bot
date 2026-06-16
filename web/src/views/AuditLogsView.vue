<template>
  <div class="page-stack">
    <PageHeader eyebrow="审计日志" title="操作审计" description="记录所有群管操作，包括封禁、禁言、踢人、警告、关键词过滤等，时间按北京时间显示。">
      <template #actions>
        <el-button :icon="Refresh" :loading="loading" @click="loadLogs">刷新</el-button>
      </template>
    </PageHeader>

    <div class="summary-grid">
      <div class="summary-card">
        <div class="summary-label">当前结果</div>
        <div class="summary-value">{{ logs.length }}</div>
        <div class="summary-meta">匹配当前筛选条件</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">封禁操作</div>
        <div class="summary-value">{{ actionCounts.ban }}</div>
        <div class="summary-meta">ban / unban</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">禁言操作</div>
        <div class="summary-value">{{ actionCounts.mute }}</div>
        <div class="summary-meta">mute / unmute</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">过滤命中</div>
        <div class="summary-value">{{ actionCounts.keyword_filter }}</div>
        <div class="summary-meta">{{ nextCursor ? '还有更多结果可加载' : '当前页已加载完成' }}</div>
      </div>
    </div>

    <PanelSection title="审计日志列表" description="支持按群组、操作类型、操作者和目标用户筛选。">
      <template #actions>
        <div class="panel-toolbar">
          <div class="control-cluster filters">
            <div class="filter-control filter-control-wide">
              <ChatSelect v-model="filters.chatId" @update:model-value="onChatChanged" />
            </div>
            <el-select v-model="filters.action" class="filter-control" clearable placeholder="操作类型" @change="loadLogs">
              <el-option label="封禁" value="ban" />
              <el-option label="解封" value="unban" />
              <el-option label="禁言" value="mute" />
              <el-option label="解除禁言" value="unmute" />
              <el-option label="踢出" value="kick" />
              <el-option label="警告" value="warn" />
              <el-option label="清除警告" value="unwarn" />
              <el-option label="关键词过滤" value="keyword_filter" />
              <el-option label="积分调整" value="points_adjust" />
              <el-option label="验证通过" value="verify_pass" />
              <el-option label="验证超时" value="verify_timeout" />
            </el-select>
          </div>
          <div class="filter-summary">
            <span>群组 {{ filters.chatId || '全部' }}</span>
            <span>操作 {{ filters.action || '全部' }}</span>
          </div>
        </div>
      </template>

      <el-alert v-if="error" class="alert" type="error" :closable="false" show-icon title="服务暂时不可用" />
      <div class="table-wrap">
      <el-table class="table-compact" :data="logs" size="small" stripe v-loading="loading">
        <el-table-column label="操作者" min-width="140">
          <template #default="{ row }">
            <strong>{{ row.actor_telegram_id || '系统' }}</strong>
          </template>
        </el-table-column>
        <el-table-column label="群组" min-width="120">
          <template #default="{ row }">{{ row.chat_telegram_id }}</template>
        </el-table-column>
        <el-table-column prop="action" label="操作" min-width="130">
          <template #default="{ row }">
            <el-tag :type="actionTag(row.action)" effect="dark">{{ actionLabel(row.action) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="目标" min-width="120">
          <template #default="{ row }">
            <span v-if="row.target_telegram_id">{{ row.target_telegram_id }}</span>
            <span v-else class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="entity_type" label="实体类型" min-width="100" />
        <el-table-column label="详情" min-width="200">
          <template #default="{ row }">
            <span class="detail-text">{{ row.detail || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="时间" min-width="180">
          <template #default="{ row }">{{ formatDateTime(row.occurred_at) }}</template>
        </el-table-column>
      </el-table>
      </div>
      <div v-if="nextCursor" class="load-more">
        <el-button :loading="loadingMore" @click="loadMoreLogs">加载更多</el-button>
      </div>
    </PanelSection>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { Refresh } from "@element-plus/icons-vue";
import ChatSelect from "@/components/ChatSelect.vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { fetchAuditLogs } from "@/api/audit";
import type { AuditLogEntry, ChatID } from "@/types/api";
import { formatDateTime } from "@/utils/helpers";

const loading = ref(false);
const loadingMore = ref(false);
const error = ref(false);
const logs = ref<AuditLogEntry[]>([]);
const nextCursor = ref("");
const filters = reactive({ chatId: "", action: "" });

const actionCounts = computed(() => {
  return logs.value.reduce(
    (acc, item) => {
      if (item.action === "ban" || item.action === "unban") acc.ban += 1;
      else if (item.action === "mute" || item.action === "unmute") acc.mute += 1;
      else if (item.action === "keyword_filter") acc.keyword_filter += 1;
      return acc;
    },
    { ban: 0, mute: 0, keyword_filter: 0 },
  );
});

function actionLabel(action: string): string {
  return (
    {
      ban: "封禁",
      unban: "解封",
      mute: "禁言",
      unmute: "解除禁言",
      kick: "踢出",
      warn: "警告",
      unwarn: "清除警告",
      keyword_filter: "关键词过滤",
      auto_reply: "自动回复",
      points_adjust: "积分调整",
      lottery_create: "创建抽奖",
      lottery_cancel: "取消抽奖",
      lottery_draw: "开奖",
      post_create: "创建帖子",
      post_delete: "删除帖子",
      verify_pass: "验证通过",
      verify_timeout: "验证超时",
    }[action] ?? action
  );
}

function actionTag(action: string): "success" | "warning" | "danger" | "info" {
  if (action === "ban" || action === "kick") return "danger";
  if (action === "mute" || action === "warn") return "warning";
  if (action === "unban" || action === "unmute" || action === "verify_pass") return "success";
  return "info";
}

async function loadLogs(): Promise<void> {
  loading.value = true;
  error.value = false;
  nextCursor.value = "";
  try {
    const response = await fetchAuditLogs({
      chatId: filters.chatId,
      action: filters.action,
    });
    logs.value = response.items;
    nextCursor.value = response.next_cursor || "";
  } catch {
    logs.value = [];
    nextCursor.value = "";
    error.value = true;
    ElMessage.error("服务暂时不可用");
  } finally {
    loading.value = false;
  }
}

async function loadMoreLogs(): Promise<void> {
  if (!nextCursor.value) return;
  loadingMore.value = true;
  try {
    const response = await fetchAuditLogs({
      chatId: filters.chatId,
      action: filters.action,
      cursor: nextCursor.value,
    });
    logs.value = logs.value.concat(response.items);
    nextCursor.value = response.next_cursor || "";
  } catch {
    ElMessage.error("更多记录加载失败");
  } finally {
    loadingMore.value = false;
  }
}

function onChatChanged(): void {
  void loadLogs();
}

onMounted(loadLogs);
</script>

<style scoped>
.panel-toolbar {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: min(100%, 1040px);
}

.filters :deep(.chat-select) {
  width: 100%;
}

.alert {
  margin-bottom: 12px;
}

.muted {
  color: var(--app-muted);
  font-size: 12px;
}

.detail-text {
  color: var(--app-muted);
  font-size: 12px;
  word-break: break-all;
}

.load-more {
  display: flex;
  justify-content: center;
  margin-top: 16px;
}
</style>
