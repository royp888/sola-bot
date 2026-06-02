<template>
  <div class="page">
    <PageHeader eyebrow="Private Ops" title="积分流水" description="按 Chat 和用户查看加分、扣分记录，便于追溯运营调整，时间按北京时间显示。">
      <template #actions>
        <ChatSelect v-model="selectedChatId" @update:model-value="loadRank" />
        <UserSelect v-model="userId" :chat-id="selectedChatId" />
        <el-button :icon="Refresh" :loading="loading" @click="loadLogs">查询</el-button>
      </template>
    </PageHeader>

    <el-row :gutter="16">
      <el-col :xs="24" :lg="16">
        <PanelSection title="流水明细" description="后端约定接口：GET /api/points/logs/:chatID/:userID。">
          <div class="table-wrap">
          <el-table :data="logs" stripe>
            <el-table-column label="时间" min-width="180">
              <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="用户" min-width="140">
              <template #default="{ row }">{{ row.username || row.user_id }}</template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="140" />
            <el-table-column label="变化" width="120">
              <template #default="{ row }">
                <el-tag :type="row.delta >= 0 ? 'success' : 'danger'" effect="dark">
                  {{ row.delta >= 0 ? "+" : "" }}{{ row.delta }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="chat_id" label="Chat" min-width="120" />
          </el-table>
          </div>
          <el-pagination
            class="pager"
            layout="sizes, prev, pager, next"
            :page-sizes="[10, 20, 50, 100]"
            :current-page="currentPage"
            :page-size="pageSize"
            :total="pagerTotal"
            @size-change="onPageSizeChange"
            @current-change="onPageChange"
          />
        </PanelSection>
      </el-col>

      <el-col :xs="24" :lg="8">
        <PanelSection title="排行榜预览" description="同屏看当前周期 TOP10。">
          <div class="rank-tools">
            <el-segmented v-model="period" :options="periodOptions" @change="loadRank" />
          </div>
          <div class="rank-list">
            <div v-for="item in rank" :key="item.user_id" class="rank-row">
              <span class="rank-num">#{{ item.rank }}</span>
              <strong>{{ item.username || item.nickname || item.user_id }}</strong>
              <span>{{ rankPoints(item) }}</span>
            </div>
          </div>
        </PanelSection>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import { Refresh } from "@element-plus/icons-vue";
import { useRoute } from "vue-router";
import ChatSelect from "@/components/ChatSelect.vue";
import UserSelect from "@/components/UserSelect.vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { fetchPointLogs, fetchPointRank } from "@/api/points";
import type { PointLogRecord, PointRankRecord } from "@/types/api";
import { formatChinaDateTime } from "@/utils/datetime";

const route = useRoute();
const selectedChatId = ref("");
const userId = ref("");
const period = ref("all");
const loading = ref(false);
const currentPage = ref(1);
const pageSize = ref(20);
const logs = ref<PointLogRecord[]>([]);
const rank = ref<PointRankRecord[]>([]);
const periodOptions = [
  { label: "全部", value: "all" },
  { label: "今日", value: "day" },
  { label: "本周", value: "week" },
  { label: "本月", value: "month" },
];

const pagerTotal = computed(() => {
  const loadedBefore = (currentPage.value - 1) * pageSize.value;
  return loadedBefore + logs.value.length + (logs.value.length === pageSize.value ? 1 : 0);
});

function formatDateTime(value?: string | null): string {
  return formatChinaDateTime(value, "-");
}

function rankPoints(item: PointRankRecord): number {
  return item.total_points ?? item.points ?? 0;
}

async function loadLogs(): Promise<void> {
  if (!selectedChatId.value || !userId.value) {
    ElMessage.warning("请先选择群组和用户");
    return;
  }
  loading.value = true;
  try {
    logs.value = await fetchPointLogs(selectedChatId.value, userId.value, {
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value,
    });
  } catch {
    logs.value = [];
    ElMessage.error("接口不可用");
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
  if (!selectedChatId.value) return;
  try {
    rank.value = await fetchPointRank(selectedChatId.value, period.value);
  } catch {
    rank.value = [];
    ElMessage.error("接口不可用");
  }
}

onMounted(() => {
  const queryChatID = route.query.chat_id;
  if (typeof queryChatID === "string") {
    selectedChatId.value = queryChatID;
  }
  void loadRank();
});
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.user-input {
  width: 220px;
}

.rank-tools {
  margin-bottom: 14px;
}

.pager {
  justify-content: flex-end;
  margin-top: 14px;
}

.rank-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.rank-row {
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
}

.rank-num {
  color: var(--app-accent);
  font-weight: 700;
}

.rank-row span:last-child {
  color: var(--app-muted);
}

@media (max-width: 720px) {
  .user-input {
    width: 100%;
  }
}
</style>
