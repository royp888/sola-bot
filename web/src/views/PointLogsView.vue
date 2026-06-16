<template>
  <div class="page-stack point-logs-page">
    <PageHeader
      eyebrow="成员运营"
      title="积分流水"
      description="先确定群组和成员，再查看这个成员在当前群组中的加分、扣分和处理原因。"
    >
      <template #meta>
        <span class="header-meta">{{ headerMeta }}</span>
      </template>
      <template #actions>
        <el-button :icon="Refresh" :loading="loading || rankState === 'loading'" @click="reloadCurrent">刷新</el-button>
      </template>
    </PageHeader>

    <PanelSection title="查询条件" description="先选群组，再选成员。查询范围确认清楚以后，结果区只展示当前会话对应的积分流水。">
      <div class="query-grid">
        <div class="query-field">
          <span class="query-label">1 选择群组</span>
          <ChatSelect :model-value="selectedChatId" @update:model-value="handleChatChange" @loaded="handleChatsLoaded" />
          <p class="query-help">{{ chatHelpText }}</p>
        </div>

        <div class="query-field">
          <span class="query-label">2 选择成员</span>
          <UserSelect
            :model-value="userId"
            :chat-id="selectedChatId"
            @update:model-value="handleUserChange"
            @loaded="handleUsersLoaded"
          />
          <p class="query-help">{{ userHelpText }}</p>
        </div>

        <div class="query-field query-field-action">
          <span class="query-label">3 查看结果</span>
          <el-button type="primary" :icon="Search" :loading="loading" @click="loadLogs">查看积分流水</el-button>
          <p class="query-help">会按当前群组与成员发起查询，只显示本次会话对应的结果。</p>
        </div>
      </div>

      <div class="query-foot">
        <div class="scope-note">
          <strong>{{ scopeHeadline }}</strong>
          <span>{{ scopeDescription }}</span>
        </div>
        <span v-if="selectionNotice" class="notice-chip">{{ selectionNotice }}</span>
      </div>
    </PanelSection>

    <div class="page-grid">
      <div class="main-stack">
        <PanelSection title="结果概览" description="先确认本次会话的群组、成员和本页摘要，再继续看明细。">
          <div class="result-overview-grid">
            <article v-for="card in resultCards" :key="card.label" class="result-card">
              <span class="result-card-label">{{ card.label }}</span>
              <strong class="result-card-value">{{ card.value }}</strong>
              <p class="result-card-note">{{ card.note }}</p>
            </article>
          </div>
        </PanelSection>

        <PanelSection title="积分流水" description="按时间查看该成员在当前群组中的积分变动，不再混入其他群组或其他成员记录。">
          <template #actions>
            <div class="result-toolbar point-log-toolbar">
              <strong>{{ resultHeadline }}</strong>
              <div class="result-toolbar-meta">
                <span>{{ pageSummary }}</span>
                <span>{{ paginationHint }}</span>
              </div>
            </div>
          </template>

          <div class="log-table-shell">
            <div v-if="logState === 'idle'" class="state-block is-empty state-block-spacious">
              <strong class="state-title">请选择群组和成员</strong>
              <p class="state-description">完成上方两步后，这里会展示当前会话对应的积分流水。</p>
            </div>

            <div v-else-if="logState === 'loading'" class="state-block is-loading state-block-spacious">
              <strong class="state-title">正在加载积分流水</strong>
              <p class="state-description">正在读取当前会话的积分变动记录，请稍候。</p>
            </div>

            <div v-else-if="logState === 'error'" class="state-block is-error state-block-spacious">
              <strong class="state-title">积分流水加载失败，请稍后重试</strong>
              <p class="state-description">当前会话的积分流水暂时无法读取，可以稍后重试，或重新选择群组和成员后再查询。</p>
              <el-button :icon="Refresh" @click="loadLogs">重新加载</el-button>
            </div>

            <div v-else-if="logState === 'empty'" class="state-block is-empty state-block-spacious">
              <strong class="state-title">暂无积分流水</strong>
              <p class="state-description">当前会话还没有产生积分变动记录。这个成员可能还没有触发签到、发言奖励或人工调整。</p>
            </div>

            <template v-else>
              <div class="mobile-list">
                <article v-for="row in logs" :key="String(row.id)" class="log-card">
                  <div class="log-card-head">
                    <div class="log-card-title">
                      <strong>{{ formatDateTime(row.created_at) }}</strong>
                      <span>{{ selectedUserLabel }}</span>
                    </div>
                    <el-tag :type="row.delta >= 0 ? 'success' : 'danger'" effect="plain" class="delta-tag">
                      {{ row.delta >= 0 ? "+" : "" }}{{ row.delta }} 分
                    </el-tag>
                  </div>
                  <p class="log-card-reason">{{ formatReason(row.reason) }}</p>
                </article>
              </div>

              <div class="table-wrap desktop-table">
                <el-table :data="logs" class="logs-table">
                  <el-table-column label="时间" min-width="188">
                    <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
                  </el-table-column>
                  <el-table-column label="积分变化" width="130">
                    <template #default="{ row }">
                      <el-tag :type="row.delta >= 0 ? 'success' : 'danger'" effect="plain" class="delta-tag">
                        {{ row.delta >= 0 ? "+" : "" }}{{ row.delta }} 分
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="原因" min-width="320">
                    <template #default="{ row }">{{ formatReason(row.reason) }}</template>
                  </el-table-column>
                </el-table>
              </div>

              <div class="table-footer">
                <span class="table-summary">{{ pageSummary }}，{{ deltaSummary }}。{{ paginationHint }}</span>
                <el-pagination
                  class="pager"
                  background
                  layout="prev, pager, next, sizes"
                  :page-size="pageSize"
                  :page-sizes="[10, 20, 50]"
                  :current-page="currentPage"
                  :total="pagerTotal"
                  @size-change="onPageSizeChange"
                  @current-change="onPageChange"
                />
              </div>
            </template>
          </div>
        </PanelSection>
      </div>

      <div class="side-stack">
        <PanelSection title="当前群组排行榜" description="这里展示当前群组积分前十成员，方便横向参考这个成员的大致位置，点击成员可直接切换查看他的流水。">
          <template #actions>
            <div class="rank-toolbar">
              <span class="rank-toolbar-hint">周期</span>
              <el-select v-model="period" size="small" class="rank-period">
                <el-option v-for="item in periodOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </div>
          </template>

          <div v-if="rankState === 'loading'" class="state-block is-loading rank-state-block">
            <strong class="state-title">正在加载排行榜</strong>
            <p class="state-description">正在获取当前群组的积分排行，请稍候。</p>
          </div>

          <div v-else-if="rankState === 'error'" class="state-block is-error rank-state-block">
            <strong class="state-title">排行榜加载失败</strong>
            <p class="state-description">请稍后重试，或确认当前群组是否已开启积分功能。</p>
          </div>

          <div v-else-if="!selectedChatId" class="state-block is-empty rank-state-block">
            <strong class="state-title">请选择群组</strong>
            <p class="state-description">选定群组后，这里会展示当前周期的积分前十名。</p>
          </div>

          <div v-else-if="!rank.length" class="state-block is-empty rank-state-block">
            <strong class="state-title">暂无排行榜数据</strong>
            <p class="state-description">当前周期还没有成员产生积分记录，后续有加分或扣分后会自动更新。</p>
          </div>

          <div v-else class="rank-list">
            <button
              v-for="item in rank"
              :key="String(item.user_id)"
              class="rank-row"
              :class="{ 'is-active': String(item.user_id) === userId }"
              @click="selectRankUser(item)"
            >
              <span class="rank-num">#{{ item.rank }}</span>
              <div class="rank-copy">
                <strong>{{ rankDisplayName(item) }}</strong>
                <span>{{ String(item.user_id) }}</span>
              </div>
              <span class="rank-points">{{ rankPoints(item) }} 分</span>
            </button>
          </div>
        </PanelSection>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { Refresh, Search } from "@element-plus/icons-vue";
import { useRoute } from "vue-router";
import ChatSelect from "@/components/ChatSelect.vue";
import UserSelect from "@/components/UserSelect.vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { fetchPointLogs, fetchPointRank } from "@/api/points";
import type { ChatRecord, PointLogRecord, PointRankRecord, UserRecord } from "@/types/api";
import { formatDateTime } from "@/utils/helpers";

type LogState = "idle" | "loading" | "error" | "empty" | "ready";
type RankState = "idle" | "loading" | "error" | "ready";

const route = useRoute();
const selectedChatId = ref("");
const userId = ref("");
const pendingUserId = ref("");
const period = ref("all");
const loading = ref(false);
const logState = ref<LogState>("idle");
const rankState = ref<RankState>("idle");
const currentPage = ref(1);
const pageSize = ref(20);
const logs = ref<PointLogRecord[]>([]);
const rank = ref<PointRankRecord[]>([]);
const nextCursor = ref("");
const chats = ref<ChatRecord[]>([]);
const users = ref<UserRecord[]>([]);
const selectionNotice = ref("");

const periodOptions = [
  { label: "全部", value: "all" },
  { label: "今日", value: "day" },
  { label: "本周", value: "week" },
  { label: "本月", value: "month" },
];

const selectedChat = computed(() => chats.value.find((item) => String(item.chat_id ?? item.id ?? "") === selectedChatId.value));

const selectedUser = computed(() => users.value.find((item) => String(item.id) === userId.value));

const selectedChatLabel = computed(() => {
  if (!selectedChatId.value) return "未选择群组";
  return selectedChat.value?.title || `群组 ${selectedChatId.value}`;
});

const selectedUserLabel = computed(() => {
  if (!userId.value) return "未选择成员";
  return selectedUser.value?.display_name || selectedUser.value?.username || String(userId.value);
});

const chatHelpText = computed(() => {
  if (!selectedChatId.value) return "先选择一个群组，成员列表会按当前群组自动刷新。";
  return `当前群组：${selectedChatLabel.value}`;
});

const userHelpText = computed(() => {
  if (!selectedChatId.value) return "请先选择群组，再继续选择成员。";
  if (!userId.value) return "群组已确定，请继续选择要查看的成员。";
  return `当前成员：${selectedUserLabel.value}`;
});

const scopeHeadline = computed(() => {
  if (!selectedChatId.value) return "当前还没有查询范围";
  if (!userId.value) return `已进入 ${selectedChatLabel.value}，等待选择成员`;
  return `当前会话：${selectedChatLabel.value} / ${selectedUserLabel.value}`;
});

const scopeDescription = computed(() => {
  if (!selectedChatId.value) return "选择群组后，这里会说明当前查询范围。";
  if (!userId.value) return "成员选定以后，结果区只显示这个成员在当前群组的积分变动。";
  return "结果只显示当前群组内这个成员的积分变动，不会混入其他会话。";
});

const pagerTotal = computed(() => {
  const loadedBefore = (currentPage.value - 1) * pageSize.value;
  return loadedBefore + logs.value.length + (nextCursor.value ? 1 : 0);
});

const positiveCount = computed(() => logs.value.filter((item) => item.delta > 0).length);
const negativeCount = computed(() => logs.value.filter((item) => item.delta < 0).length);
const netDelta = computed(() => logs.value.reduce((sum, item) => sum + item.delta, 0));
const latestLogTime = computed(() => (logs.value[0]?.created_at ? formatDateTime(logs.value[0].created_at) : "暂无记录"));

const pageSummary = computed(() => {
  if (logState.value === "ready") {
    return `第 ${currentPage.value} 页，当前页展示 ${logs.value.length} 条记录`;
  }
  return `每页最多展示 ${pageSize.value} 条记录`;
});

const deltaSummary = computed(() => {
  if (logState.value !== "ready") return "等待查询结果";
  const prefix = netDelta.value > 0 ? "+" : "";
  return `本页净变化 ${prefix}${netDelta.value} 分，增加 ${positiveCount.value} 次，扣减 ${negativeCount.value} 次`;
});

const paginationHint = computed(() => {
  if (logState.value !== "ready") return "查询完成后会显示翻页状态";
  return nextCursor.value ? "还有更多记录，可继续翻页" : "当前已到最后一页";
});

const headerMeta = computed(() => {
  if (logState.value === "loading") return "正在更新当前会话";
  if (!selectedChatId.value) return "先选群组再查成员";
  if (!userId.value) return "群组已就绪，等待选择成员";
  if (logState.value === "error") return "当前会话查询失败";
  if (logState.value === "empty") return "当前会话暂无积分流水";
  if (logState.value === "ready") return "当前会话已对齐";
  return "准备查询当前会话";
});

const resultHeadline = computed(() => {
  if (!selectedChatId.value || !userId.value) return "尚未开始查询";
  if (logState.value === "loading") return `正在加载 ${selectedUserLabel.value} 的积分流水`;
  return `${selectedUserLabel.value} 在 ${selectedChatLabel.value} 的积分流水`;
});

const resultCards = computed(() => [
  {
    label: "当前群组",
    value: selectedChatLabel.value,
    note: selectedChatId.value ? "已锁定当前会话范围" : "选择群组后可继续查询成员记录",
  },
  {
    label: "当前成员",
    value: selectedUserLabel.value,
    note: userId.value ? "结果只展示这名成员的记录" : "选择成员后才能查看积分流水",
  },
  {
    label: "本页记录",
    value: logState.value === "ready" ? `${logs.value.length} 条` : logState.value === "loading" ? "加载中" : "未查询",
    note: logState.value === "ready" ? pageSummary.value : "开始查询后会显示当前页条数",
  },
  {
    label: "最近一条",
    value: logState.value === "ready" ? latestLogTime.value : "暂无记录",
    note: logState.value === "ready" ? deltaSummary.value : "查询完成后会显示最近记录时间与本页变化",
  },
]);

const reasonLabelMap: Record<string, string> = {
  "message:text": "文本消息",
  "message:photo": "图片消息",
  "message:video": "视频消息",
  "message:voice": "语音消息",
  "message:audio": "音频消息",
  "message:document": "文件消息",
  "message:animation": "动图消息",
  "message:sticker": "表情贴纸",
  "message:location": "位置消息",
  "message:contact": "联系人消息",
  "message:poll": "投票消息",
  "message:story": "动态消息",
  "signin": "签到",
  "checkin": "签到",
  "daily_checkin": "每日签到",
  "invite": "邀请奖励",
  "invite_success": "邀请成功奖励",
  "manual_adjust": "人工调整",
  "admin:adjust": "管理员调整",
  "admin:bonus": "管理员加分",
  "admin:deduct": "管理员扣分",
  "ban": "封禁处理",
  "unban": "解除封禁",
  "warn": "警告处理",
  "lottery": "抽奖奖励",
  "task_reward": "任务奖励",
};

function formatReason(reason?: string | null): string {
  if (!reason) return "系统未记录原因";

  const normalized = reason.trim();
  if (!normalized) return "系统未记录原因";

  if (reasonLabelMap[normalized]) {
    return reasonLabelMap[normalized];
  }

  if (normalized.startsWith("message:")) {
    const type = normalized.slice("message:".length);
    return `消息互动 · ${type}`;
  }

  if (normalized.startsWith("admin:")) {
    const action = normalized.slice("admin:".length);
    return `管理员操作 · ${action}`;
  }

  return normalized
    .replace(/_/g, " ")
    .replace(/:/g, " · ");
}

function rankPoints(item: PointRankRecord): number {
  return item.total_points ?? item.points ?? 0;
}

function rankDisplayName(item: PointRankRecord): string {
  return item.username || item.nickname || String(item.user_id);
}

function resetLogsState(state: LogState = "idle"): void {
  logs.value = [];
  nextCursor.value = "";
  currentPage.value = 1;
  logState.value = state;
}

function handleChatsLoaded(items: ChatRecord[]): void {
  chats.value = items;
}

function handleUsersLoaded(items: UserRecord[]): void {
  users.value = items;
  if (pendingUserId.value) {
    userId.value = pendingUserId.value;
    pendingUserId.value = "";
    selectionNotice.value = `已恢复成员 ${selectedUserLabel.value}，点击“查看积分流水”即可加载结果。`;
  }
}

function handleChatChange(value: string): void {
  const previousChatId = selectedChatId.value;
  const hadUser = Boolean(userId.value);
  selectedChatId.value = value;
  users.value = [];
  userId.value = "";
  resetLogsState("idle");

  if (!value) {
    rank.value = [];
    rankState.value = "idle";
    selectionNotice.value = "";
    return;
  }

  if (previousChatId && previousChatId !== value && hadUser) {
    selectionNotice.value = "已切换群组，成员选择已重置，请重新选择成员。";
  } else {
    selectionNotice.value = "群组已更新，成员列表会按当前群组重新加载。";
  }

  void loadRank();
}

function handleUserChange(value: string): void {
  if (userId.value === value) return;
  userId.value = value;
  resetLogsState("idle");

  if (!value) {
    if (selectedChatId.value) {
      selectionNotice.value = "成员已清空，请重新选择后再查询。";
    }
    return;
  }

  selectionNotice.value = `成员已切换为 ${selectedUserLabel.value}，点击“查看积分流水”获取新结果。`;
}

async function loadLogs(): Promise<void> {
  if (!selectedChatId.value || !userId.value) {
    logState.value = "idle";
    ElMessage.warning("请先选择群组和成员");
    return;
  }

  loading.value = true;
  logState.value = "loading";
  try {
    const response = await fetchPointLogs(selectedChatId.value, userId.value, {
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value,
    });
    logs.value = response.items;
    nextCursor.value = response.next_cursor || "";
    logState.value = response.items.length ? "ready" : "empty";
    selectionNotice.value = "";
  } catch {
    logs.value = [];
    nextCursor.value = "";
    logState.value = "error";
    ElMessage.error("积分流水加载失败，请稍后重试");
  } finally {
    loading.value = false;
  }
}

function onPageSizeChange(size: number): void {
  pageSize.value = size;
  currentPage.value = 1;
  void loadLogs();
}

function onPageChange(page: number): void {
  currentPage.value = page;
  void loadLogs();
}

async function loadRank(): Promise<void> {
  if (!selectedChatId.value) {
    rank.value = [];
    rankState.value = "idle";
    return;
  }

  rankState.value = "loading";
  try {
    rank.value = await fetchPointRank(selectedChatId.value, period.value);
    rankState.value = "ready";
  } catch {
    rank.value = [];
    rankState.value = "error";
    ElMessage.error("排行榜加载失败，请稍后重试");
  }
}

function selectRankUser(item: PointRankRecord): void {
  const nextUserId = String(item.user_id);
  if (!selectedChatId.value) return;
  userId.value = nextUserId;
  selectionNotice.value = `已切换到 ${rankDisplayName(item)}，正在加载他的积分流水。`;
  currentPage.value = 1;
  void loadLogs();
}

function reloadCurrent(): void {
  if (selectedChatId.value) {
    void loadRank();
  }
  if (selectedChatId.value && userId.value) {
    void loadLogs();
  }
}

watch(period, () => {
  if (selectedChatId.value) {
    void loadRank();
  }
});

onMounted(() => {
  const queryChatId = route.query.chat_id;
  const queryUserId = route.query.user_id;

  if (typeof queryChatId === "string") {
    selectedChatId.value = queryChatId;
    if (typeof queryUserId === "string") {
      pendingUserId.value = queryUserId;
    }
    void loadRank();
  } else {
    resetLogsState("idle");
  }
});
</script>

<style scoped>
.header-meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.page-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.64fr) minmax(292px, 0.84fr);
  align-items: start;
  gap: 18px;
}

.main-stack,
.side-stack {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.query-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

.query-field,
.result-card,
.rank-row,
.log-card {
  border: 1px solid var(--app-tint-medium);
  border-radius: 12px;
  background: var(--app-tint-subtle);
}

.query-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 126px;
  padding: 14px;
}

.query-field-action {
  justify-content: flex-start;
}

.query-field-action .el-button {
  width: 100%;
}

.query-label {
  color: var(--app-muted);
  font-size: 11px;
  font-weight: 700;
}

.query-help {
  margin: 0;
  color: var(--app-muted);
  font-size: 12px;
  line-height: 1.6;
}

.query-foot {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--app-table-border);
}

.scope-note {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.scope-note strong {
  font-size: 13px;
}

.scope-note span {
  color: var(--app-muted);
  font-size: 12px;
  line-height: 1.6;
}

.notice-chip {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 10px;
  border: 1px solid var(--app-accent-hover-border);
  border-radius: 999px;
  background: var(--app-accent-hover-bg);
  color: var(--app-muted-strong);
  font-size: 12px;
}

.point-log-toolbar {
  min-width: 0;
}

.result-overview-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.result-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 118px;
  padding: 14px;
}

.result-card-label {
  color: var(--app-muted);
  font-size: 12px;
  font-weight: 600;
}

.result-card-value {
  font-size: 20px;
  font-weight: 720;
  line-height: 1.25;
}

.result-card-note {
  margin: 0;
  color: var(--app-muted);
  font-size: 12px;
  line-height: 1.6;
}

.log-table-shell {
  overflow: hidden;
  border: 1px solid var(--app-table-border);
  border-radius: 12px;
  background: var(--app-inset-bg);
}

.state-block-spacious {
  min-height: 232px;
  justify-content: center;
  padding: 22px;
}

.logs-table {
  width: 100%;
}

.delta-tag {
  min-width: 88px;
  justify-content: center;
}

.table-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 12px 14px;
  border-top: 1px solid var(--app-table-border);
  background: var(--app-tint-subtle);
}

.table-summary {
  color: var(--app-muted);
  font-size: 12px;
  line-height: 1.6;
}

.pager {
  margin-left: auto;
}

.rank-toolbar {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

.rank-toolbar-hint {
  color: var(--app-muted);
  font-size: 12px;
}

.rank-period {
  width: 120px;
}

.rank-state-block {
  min-height: 164px;
}

.rank-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.rank-row {
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 12px 13px;
  color: var(--app-text);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.18s ease, background 0.18s ease, transform 0.18s ease;
}

.rank-row:hover,
.rank-row.is-active {
  border-color: var(--app-accent-hover-border);
  background: var(--app-accent-hover-bg);
  transform: translateY(-1px);
}

.rank-num {
  color: var(--app-accent);
  font-weight: 700;
}

.rank-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.rank-copy strong {
  font-size: 13px;
}

.rank-copy span,
.rank-points {
  color: var(--app-muted);
  font-size: 12px;
}

.rank-points {
  white-space: nowrap;
}

.log-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
}

.log-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.log-card-title {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.log-card-title strong {
  font-size: 13px;
}

.log-card-title span,
.log-card-reason {
  color: var(--app-muted);
  font-size: 12px;
  line-height: 1.6;
}

.log-card-reason {
  margin: 0;
  color: var(--app-muted-strong);
}

@media (max-width: 1180px) {
  .page-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .query-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .query-field-action {
    grid-column: 1 / -1;
  }
}

@media (max-width: 720px) {
  .query-grid,
  .result-overview-grid {
    grid-template-columns: 1fr;
  }

  .query-foot,
  .table-footer,
  .log-card-head,
  .rank-row {
    width: 100%;
  }

  .table-footer,
  .log-card-head {
    flex-direction: column;
    align-items: flex-start;
  }

  .pager {
    margin-left: 0;
  }

  .rank-toolbar {
    width: 100%;
    justify-content: space-between;
  }

  .rank-period {
    width: 132px;
  }

  .rank-row {
    grid-template-columns: 40px minmax(0, 1fr);
  }

  .rank-points {
    grid-column: 2;
  }
}
</style>


