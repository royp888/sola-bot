<template>
  <div class="page-stack dashboard-page">
    <PageHeader
      eyebrow="Control Room"
      title="运营总览"
      description="先处理异常，再推进今天的发布、风控和群组运营。"
    >
      <template #meta>
        <span class="header-meta">{{ focusSummary }}</span>
      </template>
      <template #actions>
        <el-button :icon="Refresh" @click="loadDashboard">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="router.push('/posts')">新建任务</el-button>
      </template>
    </PageHeader>

    <div class="metric-grid">
      <StatTile v-for="metric in summary.metrics" :key="metric.label" v-bind="metric" />
    </div>

    <div class="dashboard-grid">
      <div class="dashboard-main">
        <PanelSection title="优先处理" description="把失败任务、异常事件和健康风险放在第一屏。">
          <div class="queue-list">
            <button v-for="item in focusQueue" :key="item.title + item.detail" class="queue-item" :data-tone="item.tone" @click="router.push(item.path)">
              <div class="queue-head">
                <strong>{{ item.title }}</strong>
                <span>{{ item.action }}</span>
              </div>
              <p>{{ item.detail }}</p>
            </button>
          </div>
        </PanelSection>

        <PanelSection title="任务执行" description="定时发布与调度队列的当前状态。">
          <el-table :data="summary.jobs" size="small" stripe empty-text="暂无任务">
            <el-table-column prop="title" label="任务" min-width="180" />
            <el-table-column prop="schedule" label="计划" min-width="120" />
            <el-table-column prop="nextRun" label="下次执行" min-width="140" />
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <el-tag :type="jobTag(row.status)" effect="plain">{{ jobLabel(row.status) }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </PanelSection>
      </div>

      <div class="dashboard-side">
        <PanelSection title="快捷动作" description="高频入口按成员、内容、风控和增长组织。">
          <div class="action-grid">
            <button v-for="entry in quickEntries" :key="entry.path" class="action-card" @click="router.push(entry.path)">
              <span class="action-group">{{ entry.group }}</span>
              <div class="action-title">
                <el-icon><component :is="entry.icon" /></el-icon>
                <strong>{{ entry.title }}</strong>
              </div>
              <p>{{ entry.note }}</p>
            </button>
          </div>
        </PanelSection>

        <PanelSection title="系统健康" description="接口、队列和同步链路的即时状态。">
          <div class="health-list">
            <div v-for="item in summary.health" :key="item.label" class="health-row">
              <div class="health-copy">
                <div>
                  <strong>{{ item.label }}</strong>
                  <span>{{ item.note }}</span>
                </div>
                <span class="health-value">{{ item.value }}%</span>
              </div>
              <el-progress :percentage="item.value" :stroke-width="8" :status="progressStatus(item.value)" />
            </div>
          </div>
        </PanelSection>

        <PanelSection title="最近事件" description="只保留值得扫一眼的事件流。">
          <div v-if="summary.activity.length" class="activity-list">
            <div v-for="item in summary.activity" :key="item.title + item.time" class="activity-item" :data-tone="item.status">
              <div class="activity-head">
                <strong>{{ item.title }}</strong>
                <span>{{ item.time }}</span>
              </div>
              <p>{{ item.detail }}</p>
            </div>
          </div>
          <div v-else class="empty-copy">最近没有新的系统事件。</div>
        </PanelSection>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import {
  Calendar,
  CircleClose,
  Coin,
  MessageBox,
  Plus,
  Refresh,
  Tickets,
  Trophy,
  UserFilled,
} from "@element-plus/icons-vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import StatTile from "@/components/StatTile.vue";
import { fetchDashboardSummary } from "@/api/dashboard";
import type { DashboardSummary } from "@/types/api";

const router = useRouter();
const summary = reactive<DashboardSummary>({
  metrics: [],
  activity: [],
  jobs: [],
  health: [],
});

const quickEntries = [
  { group: "成员", title: "成员管理", note: "筛选成员并执行批量动作", path: "/users", icon: UserFilled },
  { group: "积分", title: "积分规则", note: "调整积分策略和冷却配置", path: "/points/config", icon: Coin },
  { group: "风控", title: "违规处理", note: "处理违规和封禁动作", path: "/violations", icon: CircleClose },
  { group: "内容", title: "发布任务", note: "安排定时发布和调度", path: "/posts", icon: Calendar },
  { group: "日志", title: "积分记录", note: "回放积分流水和原因", path: "/points/logs", icon: Tickets },
  { group: "增长", title: "活动抽奖", note: "推进促活和奖励发放", path: "/lottery", icon: Trophy },
  { group: "规则", title: "关键词规则", note: "查看命中词与触发策略", path: "/keywords", icon: MessageBox },
];

const focusQueue = computed(() => {
  const items: Array<{ title: string; detail: string; tone: "success" | "warning" | "danger" | "info"; action: string; path: string }> = [];

  summary.jobs
    .filter((job) => job.status !== "live")
    .slice(0, 2)
    .forEach((job) => {
      items.push({
        title: `${job.title} ${jobLabel(job.status)}`,
        detail: `计划 ${job.schedule}，下次执行 ${job.nextRun}`,
        tone: job.status === "failed" ? "danger" : "warning",
        action: "查看任务",
        path: "/posts",
      });
    });

  summary.activity
    .filter((item) => item.status === "warning" || item.status === "danger")
    .slice(0, 2)
    .forEach((item) => {
      items.push({
        title: item.title,
        detail: item.detail,
        tone: item.status,
        action: "查看事件",
        path: item.status === "danger" ? "/violations" : "/stats",
      });
    });

  summary.health
    .filter((item) => item.value < 80)
    .slice(0, 2)
    .forEach((item) => {
      items.push({
        title: `${item.label} 需要关注`,
        detail: `${item.note}，当前健康值 ${item.value}%`,
        tone: item.value < 60 ? "danger" : "warning",
        action: "查看分析",
        path: "/stats",
      });
    });

  if (items.length === 0) {
    items.push({
      title: "当前没有高优先级异常",
      detail: "任务、健康和近期事件都处于可接受范围，可以继续推进日常运营。",
      tone: "success",
      action: "查看分析",
      path: "/stats",
    });
  }

  return items.slice(0, 5);
});

const focusSummary = computed(() => {
  const risky = focusQueue.value.filter((item) => item.tone !== "success").length;
  return risky > 0 ? `${risky} 项待优先处理` : "当前运行平稳";
});

async function loadDashboard(): Promise<void> {
  try {
    const response = await fetchDashboardSummary();
    Object.assign(summary, response);
  } catch {
    Object.assign(summary, { metrics: [], activity: [], jobs: [], health: [] });
    ElMessage.error("运营概览接口不可用");
  }
}

function jobTag(status: DashboardSummary["jobs"][number]["status"]): "success" | "warning" | "danger" {
  if (status === "live") return "success";
  if (status === "paused") return "warning";
  return "danger";
}

function jobLabel(status: DashboardSummary["jobs"][number]["status"]): string {
  if (status === "live") return "运行中";
  if (status === "paused") return "已暂停";
  return "失败";
}

function progressStatus(value: number): "success" | "warning" | "exception" {
  if (value >= 85) return "success";
  if (value >= 60) return "warning";
  return "exception";
}

onMounted(loadDashboard);
</script>

<style scoped>
.header-meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
  border: 1px solid var(--app-border);
  border-radius: 999px;
  background: var(--app-surface);
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(320px, 0.9fr);
  gap: 16px;
}

.dashboard-main,
.dashboard-side {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.queue-list,
.action-grid,
.activity-list {
  display: grid;
  gap: 10px;
}

.queue-item,
.action-card {
  width: 100%;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius);
  background: var(--app-surface-2);
  color: var(--app-text);
  text-align: left;
  cursor: pointer;
}

.queue-item {
  padding: 14px;
}

.queue-item[data-tone="success"] {
  border-color: rgba(118, 181, 138, 0.32);
}

.queue-item[data-tone="warning"] {
  border-color: rgba(207, 160, 98, 0.32);
}

.queue-item[data-tone="danger"] {
  border-color: rgba(199, 116, 116, 0.32);
}

.queue-head,
.activity-head,
.health-copy {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.queue-head span,
.activity-head span,
.health-copy span,
.action-group {
  color: var(--app-muted);
  font-size: 12px;
}

.queue-item p,
.action-card p,
.activity-item p {
  margin: 6px 0 0;
  color: var(--app-muted);
  font-size: 12px;
  line-height: 1.55;
}

.action-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.action-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 118px;
  padding: 14px;
}

.action-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.health-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.health-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.health-copy strong {
  display: block;
  margin-bottom: 2px;
  font-size: 13px;
}

.health-value {
  min-width: 42px;
  text-align: right;
}

.activity-item {
  padding: 12px 14px;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius);
  background: var(--app-surface-2);
}

.activity-item[data-tone="warning"] {
  border-color: rgba(207, 160, 98, 0.28);
}

.activity-item[data-tone="danger"] {
  border-color: rgba(199, 116, 116, 0.28);
}

.empty-copy {
  color: var(--app-muted);
  font-size: 13px;
}

@media (max-width: 1180px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .action-grid {
    grid-template-columns: 1fr;
  }
}
</style>