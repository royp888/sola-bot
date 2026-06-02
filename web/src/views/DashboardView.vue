<template>
  <div class="page-stack dashboard-page">
    <PageHeader eyebrow="Control Room" title="运营总览" description="先处理异常，再安排今天的发布、群组运营和系统巡检。">
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

        <PanelSection title="任务执行" description="重点看最近要跑的任务和异常状态。">
          <el-table :data="summary.jobs" size="small" stripe empty-text="暂无任务">
            <el-table-column prop="title" label="任务" min-width="180" />
            <el-table-column prop="schedule" label="执行计划" min-width="140" />
            <el-table-column prop="nextRun" label="下次执行" min-width="160" />
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <el-tag :type="jobTag(row.status)" effect="plain">{{ jobLabel(row.status) }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </PanelSection>
      </div>

      <div class="dashboard-side">
        <PanelSection title="工作入口" description="只保留高频入口，按真实运营动作来组织。">
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

        <PanelSection title="最近事件" description="只保留值得扫一眼的系统变化。">
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
  { group: "积分", title: "积分规则", note: "调整规则与排行榜节奏", path: "/points", icon: Coin },
  { group: "风控", title: "违规处理", note: "查看待处理违规和封禁情况", path: "/violations", icon: CircleClose },
  { group: "内容", title: "发布任务", note: "新建提醒、公告和自动发布", path: "/posts", icon: Calendar },
  { group: "运营", title: "消息记录", note: "追踪机器人投递与互动表现", path: "/messages", icon: MessageBox },
  { group: "活动", title: "抽奖活动", note: "管理进行中与历史抽奖", path: "/lotteries", icon: Trophy },
  { group: "调度", title: "任务调度", note: "查看任务配置与执行状态", path: "/schedules", icon: Tickets },
];

const focusQueue = computed(() => {
  const items: Array<{ title: string; detail: string; tone: string; action: string; path: string }> = [];

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
  padding: 7px 11px;
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.03);
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.28fr) minmax(320px, 0.92fr);
  gap: 18px;
}

.dashboard-main,
.dashboard-side {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.queue-list,
.action-grid,
.activity-list {
  display: grid;
  gap: 12px;
}

.queue-item,
.action-card {
  width: 100%;
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 16px;
  background: linear-gradient(180deg, rgba(21, 31, 45, 0.94), rgba(17, 25, 37, 0.94));
  color: var(--app-text);
  text-align: left;
  cursor: pointer;
  box-shadow: var(--app-shadow-soft);
}

.queue-item {
  padding: 16px;
}

.queue-item[data-tone="success"] {
  border-color: rgba(114, 192, 145, 0.28);
}

.queue-item[data-tone="warning"] {
  border-color: rgba(216, 162, 95, 0.28);
}

.queue-item[data-tone="danger"] {
  border-color: rgba(210, 120, 120, 0.3);
}

.queue-head,
.activity-head,
.health-copy {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.queue-head strong,
.activity-head strong {
  font-size: 14px;
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
  margin: 8px 0 0;
  color: var(--app-muted);
  font-size: 12px;
  line-height: 1.6;
}

.action-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.action-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 126px;
  padding: 16px;
}

.action-title {
  display: flex;
  align-items: center;
  gap: 9px;
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
  gap: 10px;
}

.health-copy strong {
  display: block;
  margin-bottom: 4px;
  font-size: 13px;
}

.health-value {
  min-width: 42px;
  text-align: right;
}

.activity-item {
  padding: 14px 16px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 16px;
  background: linear-gradient(180deg, rgba(21, 31, 45, 0.94), rgba(17, 25, 37, 0.94));
}

.activity-item[data-tone="warning"] {
  border-color: rgba(216, 162, 95, 0.26);
}

.activity-item[data-tone="danger"] {
  border-color: rgba(210, 120, 120, 0.28);
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
