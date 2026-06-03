<template>
  <div class="page-stack">
    <PageHeader eyebrow="活动运营" title="活动抽奖" description="管理群内抽奖活动、参与方式和开奖结果。">
      <template #meta>
        <span class="header-meta">页面时间按北京时间（东八区）显示</span>
      </template>
      <template #actions>
        <el-button :icon="Refresh" :loading="loading" @click="loadLotteries">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">创建抽奖</el-button>
      </template>
    </PageHeader>

    <div class="summary-grid">
      <div class="summary-card">
        <div class="summary-label">当前活动</div>
        <div class="summary-value">{{ filteredLotteries.length }}</div>
        <div class="summary-meta">匹配当前筛选条件</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">进行中</div>
        <div class="summary-value">{{ statusCounts.active }}</div>
        <div class="summary-meta">待开奖活动</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">已开奖</div>
        <div class="summary-value">{{ statusCounts.ended }}</div>
        <div class="summary-meta">已完成结果发放</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">参与次数</div>
        <div class="summary-value">{{ totalEntries }}</div>
        <div class="summary-meta">当前列表累计参与</div>
      </div>
    </div>

    <PanelSection title="抽奖列表" description="支持按群组、状态和参与方式查看活动进度。">
      <template #actions>
        <div class="panel-toolbar">
          <div class="control-cluster filters">
            <div class="filter-control filter-control-wide">
              <ChatSelect v-model="selectedChatId" />
            </div>
            <el-select v-model="statusFilter" class="filter-control" clearable placeholder="状态">
              <el-option label="进行中" value="active" />
              <el-option label="已开奖" value="ended" />
              <el-option label="已取消" value="cancelled" />
            </el-select>
            <el-select v-model="joinTypeFilter" class="filter-control" clearable placeholder="参与方式">
              <el-option label="按钮参与" value="button" />
              <el-option label="口令参与" value="keyword" />
              <el-option label="按钮 + 口令" value="both" />
            </el-select>
          </div>
          <div class="filter-summary">
            <span>群组 {{ selectedChatId || '全部' }}</span>
            <span>状态 {{ statusFilter || '全部' }}</span>
            <span>参与方式 {{ joinTypeFilter || '全部' }}</span>
          </div>
        </div>
      </template>

      <div class="table-wrap">
        <el-table class="table-compact" :data="filteredLotteries" size="small" stripe>
        <el-table-column prop="title" label="标题" min-width="180" />
        <el-table-column prop="prize" label="奖品" min-width="140" />
        <el-table-column prop="chat_id" label="目标群组" min-width="140" />
        <el-table-column prop="cost_points" label="积分成本" width="100" />
        <el-table-column label="参与方式" width="130">
          <template #default="{ row }">
            <el-tag :type="joinTypeTag(row.join_type)" effect="plain">{{ joinTypeLabel(row.join_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="参与人数" width="110">
          <template #default="{ row }">{{ row.entry_count ?? row.participants ?? 0 }}</template>
        </el-table-column>
        <el-table-column prop="winner_count" label="中奖数" width="90" />
        <el-table-column label="开奖时间（北京时间）" min-width="180">
          <template #default="{ row }">{{ formatDateTime(row.end_at) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)" effect="dark">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="190" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="showEntries(row)">参与者</el-button>
            <el-dropdown @command="handleLotteryCommand($event, row)">
              <el-button size="small">
                更多
                <el-icon class="el-icon--right"><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="winners">中奖名单</el-dropdown-item>
                  <el-dropdown-item command="cancel" :disabled="row.status !== 'active'">取消活动</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
      </div>
    </PanelSection>

    <el-dialog v-model="dialogVisible" title="创建抽奖" width="560px">
      <el-form label-position="top">
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="目标群组">
              <ChatSelect v-model="form.chat_id" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="开奖时间（北京时间）">
              <el-date-picker
                v-model="form.end_at"
                class="wide-control"
                type="datetime"
                format="YYYY-MM-DD HH:mm"
                value-format="YYYY-MM-DD HH:mm:ss"
                placeholder="选择开奖时间（北京时间）"
              />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="标题">
          <el-input v-model="form.title" />
        </el-form-item>
        <el-form-item label="奖品">
          <el-input v-model="form.prize" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="参与方式">
              <el-segmented v-model="form.join_type" :options="joinTypeOptions" class="wide-control" />
            </el-form-item>
          </el-col>
          <el-col v-if="requiresJoinKeyword" :xs="24" :md="12">
            <el-form-item label="参与口令">
              <el-input v-model="form.join_keyword" maxlength="64" show-word-limit placeholder="例如 888" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :xs="24" :md="8">
            <el-form-item label="参与成本">
              <el-input-number v-model="form.cost_points" class="wide-control" :min="0" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="8">
            <el-form-item label="人数上限">
              <el-input-number v-model="form.max_participants" class="wide-control" :min="0" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="8">
            <el-form-item label="中奖人数">
              <el-input-number v-model="form.winner_count" class="wide-control" :min="1" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitLottery">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="entriesVisible" :title="entriesTitle" width="560px">
      <el-table class="table-compact" :data="entries" size="small" stripe>
        <el-table-column prop="user_id" label="用户 ID" min-width="150" />
        <el-table-column label="用户名" min-width="130">
          <template #default="{ row }">{{ row.username ? `@${row.username}` : '-' }}</template>
        </el-table-column>
        <el-table-column prop="joined_at" label="参与时间" min-width="170" />
        <el-table-column label="中奖" width="90">
          <template #default="{ row }">
            <el-tag :type="row.is_winner ? 'success' : 'info'" effect="dark">{{ row.is_winner ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { MoreFilled, Plus, Refresh } from "@element-plus/icons-vue";
import ChatSelect from "@/components/ChatSelect.vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { cancelLottery, createLottery, fetchLotteries, fetchLotteryEntries, fetchLotteryWinners } from "@/api/lottery";
import type { ChatID, LotteryEntryRecord, LotteryPayload, LotteryRecord } from "@/types/api";
import { formatChinaDateTime, parseChinaLocalDateTimeToISO } from "@/utils/datetime";

const loading = ref(false);
const saving = ref(false);
const dialogVisible = ref(false);
const entriesVisible = ref(false);
const lotteries = ref<LotteryRecord[]>([]);
const entries = ref<LotteryEntryRecord[]>([]);
const entriesTitle = ref("参与者");
const cancellingId = ref<ChatID>();
const selectedChatId = ref<ChatID | "">("");
const statusFilter = ref<LotteryRecord["status"] | "">("");
const joinTypeFilter = ref<LotteryRecord["join_type"] | "">("");
const form = reactive<LotteryPayload>({
  chat_id: "",
  title: "",
  prize: "",
  cost_points: 0,
  max_participants: 0,
  winner_count: 1,
  end_at: "",
  created_by: "",
  join_type: "button",
  join_keyword: "",
});
const joinTypeOptions = [
  { label: "按钮", value: "button" },
  { label: "口令", value: "keyword" },
  { label: "按钮+口令", value: "both" },
];
const requiresJoinKeyword = computed(() => form.join_type === "keyword" || form.join_type === "both");

const filteredLotteries = computed(() => {
  return lotteries.value.filter((item) => {
    const matchesChat = selectedChatId.value ? String(item.chat_id) === String(selectedChatId.value) : true;
    const matchesStatus = statusFilter.value ? item.status === statusFilter.value : true;
    const matchesJoinType = joinTypeFilter.value ? item.join_type === joinTypeFilter.value : true;
    return matchesChat && matchesStatus && matchesJoinType;
  });
});

const statusCounts = computed(() => {
  return filteredLotteries.value.reduce(
    (acc, item) => {
      acc[item.status] += 1;
      return acc;
    },
    { active: 0, ended: 0, cancelled: 0 } as Record<LotteryRecord["status"], number>,
  );
});

const totalEntries = computed(() => filteredLotteries.value.reduce((sum, item) => sum + Number(item.entry_count ?? item.participants ?? 0), 0));

function formatDateTime(value?: string | null): string {
  return formatChinaDateTime(value, "-");
}

function parseNumericId(value?: ChatID): number | undefined {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function optionalText(value?: string | null): string | undefined {
  const text = value?.trim();
  return text || undefined;
}

function toRFC3339(value?: string | null): string | undefined {
  return parseChinaLocalDateTimeToISO(value);
}

function buildPayload(): LotteryPayload | undefined {
  const chatId = parseNumericId(form.chat_id);
  if (!chatId) {
    ElMessage.warning("请输入有效的群组 ID");
    return undefined;
  }
  if (!form.title.trim()) {
    ElMessage.warning("请输入抽奖标题");
    return undefined;
  }
  const joinType = form.join_type || "button";
  const joinKeyword = optionalText(form.join_keyword);
  if ((joinType === "keyword" || joinType === "both") && !joinKeyword) {
    ElMessage.warning("请输入参与口令");
    return undefined;
  }
  return {
    chat_id: chatId,
    title: form.title.trim(),
    prize: form.prize.trim(),
    cost_points: form.cost_points,
    max_participants: form.max_participants,
    winner_count: form.winner_count,
    end_at: toRFC3339(form.end_at),
    created_by: parseNumericId(form.created_by) ?? 0,
    join_type: joinType,
    join_keyword: joinType === "button" ? undefined : joinKeyword,
  };
}

function openCreate(): void {
  Object.assign(form, {
    title: "",
    prize: "",
    cost_points: 0,
    max_participants: 0,
    winner_count: 1,
    end_at: "",
    created_by: "",
    join_type: "button",
    join_keyword: "",
  });
  dialogVisible.value = true;
}

async function loadLotteries(): Promise<void> {
  loading.value = true;
  try {
    lotteries.value = await fetchLotteries();
  } catch (error) {
    lotteries.value = [];
    ElMessage.error(errorMessage(error));
  } finally {
    loading.value = false;
  }
}

async function submitLottery(): Promise<void> {
  const payload = buildPayload();
  if (!payload) return;
  saving.value = true;
  try {
    await createLottery(payload);
    ElMessage.success("抽奖已创建");
    dialogVisible.value = false;
    await loadLotteries();
  } catch (error) {
    ElMessage.error(errorMessage(error));
  } finally {
    saving.value = false;
  }
}

async function cancel(row: LotteryRecord): Promise<void> {
  await ElMessageBox.confirm(`确认取消抽奖「${row.title}」？`, "取消抽奖", {
    type: "warning",
    confirmButtonText: "取消抽奖",
    cancelButtonText: "返回",
  });
  cancellingId.value = row.id;
  try {
    await cancelLottery(row.id);
    ElMessage.success("抽奖已取消");
    await loadLotteries();
  } catch (error) {
    ElMessage.error(errorMessage(error));
  } finally {
    cancellingId.value = undefined;
  }
}

async function showEntries(row: LotteryRecord): Promise<void> {
  entriesTitle.value = `抽奖「${row.title || row.id}」参与者`;
  try {
    entries.value = await fetchLotteryEntries(row.id);
    entriesVisible.value = true;
  } catch (error) {
    ElMessage.error(errorMessage(error));
  }
}

async function showWinners(row: LotteryRecord): Promise<void> {
  entriesTitle.value = `抽奖「${row.title || row.id}」中奖名单`;
  try {
    entries.value = await fetchLotteryWinners(row.id);
    entriesVisible.value = true;
  } catch (error) {
    ElMessage.error(errorMessage(error));
  }
}

function handleLotteryCommand(command: string, row: LotteryRecord): void {
  if (command === "winners") {
    void showWinners(row);
    return;
  }
  if (command === "cancel") {
    void cancel(row);
  }
}

function statusLabel(status: LotteryRecord["status"]): string {
  return { active: "进行中", ended: "已开奖", cancelled: "已取消" }[status];
}

function statusTag(status: LotteryRecord["status"]): "success" | "info" | "danger" {
  if (status === "active") return "success";
  if (status === "ended") return "info";
  return "danger";
}

function joinTypeLabel(joinType?: LotteryRecord["join_type"]): string {
  if (joinType === "keyword") return "口令参与";
  if (joinType === "both") return "按钮+口令";
  return "按钮参与";
}

function joinTypeTag(joinType?: LotteryRecord["join_type"]): "success" | "warning" | "primary" {
  if (joinType === "keyword") return "warning";
  if (joinType === "both") return "primary";
  return "success";
}

function errorMessage(error: unknown): string {
  const payload = (error as { payload?: { error?: string } })?.payload;
  const status = (error as { status?: number })?.status;
  return payload?.error || (status ? `接口返回 ${status}` : "接口不可用");
}

onMounted(loadLotteries);
</script>

<style scoped>
.header-meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 5px 10px;
  border: 1px solid var(--app-border);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.04);
}
.panel-toolbar {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: min(100%, 980px);
}

.filters :deep(.chat-select) {
  width: 100%;
}

.wide-control {
  width: 100%;
}
</style>
