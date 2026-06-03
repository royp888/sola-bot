<template>
  <div class="page-stack dashboard-page">
    <PageHeader eyebrow="今日总览" title="运营总览" description="先处理异常，再看关键指标、发布节奏和系统状态。">
      <template #meta>
        <span class="header-meta">{{ focusSummary }}</span>
      </template>
      <template #actions>
        <el-button :icon="Refresh" :loading="dashboardState === 'loading'" @click="loadDashboard">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="router.push('/posts')">新建任务</el-button>
      </template>
    </PageHeader>

    <div v-if="dashboardState === 'error' && !hasSummaryContent" class="state-block is-error state-block-large">
      <strong class="state-title">运营总览暂时不可用</strong>
      <p class="state-description">概览数据拉取失败，请稍后重试，或先进入成员、任务和规则页面继续处理。</p>
      <el-button :icon="Refresh" @click="loadDashboard">重新加载</el-button>
    </div>

    <div v-else-if="dashboardState === 'loading' && !hasSummaryContent" class="dashboard-skeleton">
      <div class="metric-grid">
        <el-skeleton v-for="item in 4" :key="item" animated class="skeleton-card">
          <template #template>
            <el-skeleton-item variant="rect" style="width: 100%; height: 132px" />
          </template>
        </el-skeleton>
      </div>
      <div class="dashboard-grid">
        <el-skeleton animated class="skeleton-panel">
          <template #template>
            <el-skeleton-item variant="rect" style="width: 100%; height: 320px" />
          </template>
        </el-skeleton>
        <el-skeleton animated class="skeleton-panel">
          <template #template>
            <el-skeleton-item variant="rect" style="width: 100%; height: 320px" />
          </template>
        </el-skeleton>
      </div>
    </div>

    <template v-else>
      <PanelSection title="今日待处理" description="这里只保留需要你现在跟进的事项，方便直接决定下一步动作。">
        <div class="queue-list">
          <button
            v-for="item in focusQueue"
            :key="item.title + item.detail"
            class="queue-item"
            :data-tone="item.tone"
            @click="router.push(item.path)"
          >
            <div class="queue-head">
              <strong>{{ item.title }}</strong>
              <span>{{ item.action }}</span>
            </div>
            <p>{{ item.detail }}</p>
          </button>
        </div>
      </PanelSection>

      <section class="metric-section">
        <div class="metric-section-head">
          <div class="metric-section-copy">
            <span class="metric-kicker">今日关键指标</span>
            <strong>这些数字决定今天是继续推进，还是优先排查。</strong>
          </div>
        </div>
        <div class="metric-grid">
          <StatTile
            v-for="metric in primaryMetrics"
            :key="metric.label"
            :label="metric.label"
            :value="metric.value"
            :delta="metric.delta"
            :description="metric.description"
            :badge-text="metric.badgeText"
            :value-hint="metric.valueHint"
            :tone="metric.tone"
            :clickable="Boolean(metric.path)"
            @select="navigateTo(metric.path)"
          />
        </div>
      </section>

      <div class="dashboard-grid">
        <div class="dashboard-main">
          <PanelSection title="发布任务" description="这里显示未来即将执行、已暂停或执行失败的任务，方便你继续处理。">
            <div class="table-shell">
              <el-table v-if="jobRows.length" :data="jobRows" size="small" class="job-table">
                <el-table-column prop="title" label="任务" min-width="180" />
                <el-table-column prop="schedule" label="执行计划" min-width="140" />
                <el-table-column prop="nextRun" label="下次执行" min-width="160" />
                <el-table-column label="状态" width="128">
                  <template #default="{ row }">
                    <el-tag :type="jobTag(row.status)" effect="plain">{{ jobLabel(row.status) }}</el-tag>
                  </template>
                </el-table-column>
              </el-table>
              <div v-else class="state-block is-empty state-block-inline state-block-compact">
                <strong class="state-title">暂无发布任务</strong>
                <p class="state-description">当前没有排队中的发布任务。新建任务后，这里会显示执行计划、状态和下次执行时间。</p>
              </div>
            </div>
          </PanelSection>
        </div>

        <div class="dashboard-side">
          <PanelSection title="系统健康" description="接口、同步和任务链路的即时状态，用来判断今天是否需要排障。">
            <div v-if="secondaryMetrics.length" class="system-strip">
              <button
                v-for="metric in secondaryMetrics"
                :key="metric.label"
                class="system-chip"
                :data-tone="metric.tone"
                @click="navigateTo(metric.path)"
              >
                <span>{{ metric.label }}</span>
                <strong>{{ metric.value }}</strong>
              </button>
            </div>
            <div v-if="healthRows.length" class="health-list">
              <div v-for="item in healthRows" :key="item.label" class="health-row">
                <div class="health-copy">
                  <div>
                    <strong>{{ displayHealthLabel(item.label) }}</strong>
                    <span>{{ item.note }}</span>
                  </div>
                  <span class="health-value">{{ item.value }}%</span>
                </div>
                <el-progress :percentage="item.value" :stroke-width="8" :status="progressStatus(item.value)" />
              </div>
            </div>
            <div v-else class="state-block is-empty state-block-inline state-block-compact">
              <strong class="state-title">系统状态尚未回传</strong>
              <p class="state-description">健康指标稍后会显示在这里，用来判断接口、同步和任务链路是否稳定。</p>
            </div>
          </PanelSection>

          <PanelSection title="最近事件" description="最近 24 小时的失败、告警和状态变化会集中显示在这里。">
            <div v-if="recentEvents.length" class="activity-list">
              <div v-for="item in recentEvents" :key="item.title + item.time" class="activity-item" :data-tone="item.status">
                <div class="activity-head">
                  <strong>{{ item.title }}</strong>
                  <span class="activity-time">{{ item.time }}</span>
                </div>
                <p>{{ item.detail }}</p>
              </div>
            </div>
            <div v-else class="state-block is-empty state-block-inline state-block-compact">
              <strong class="state-title">最近没有新的异常事件</strong>
              <p class="state-description">当任务失败、状态变化或系统告警出现时，最新记录会展示在这里。</p>
            </div>
          </PanelSection>

        </div>
      </div>

      <PanelSection title="常用入口" description="按高频运营动作整理，方便你快速进入常用页面。">
        <div class="action-list">
          <button v-for="entry in quickEntries" :key="entry.path" class="action-card" @click="router.push(entry.path)">
            <div class="action-card-main">
              <span class="action-group">{{ entry.group }}</span>
              <div class="action-title">
                <el-icon><component :is="entry.icon" /></el-icon>
                <strong>{{ entry.title }}</strong>
              </div>
            </div>
            <p>{{ entry.note }}</p>
          </button>
        </div>
      </PanelSection>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import {
  Calendar,
  ChatDotRound,
  CircleClose,
  Coin,
  DataAnalysis,
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
import type { DashboardSummary, OverviewMetric } from "@/types/api";

type DashboardState = "loading" | "error" | "ready";
type MetricCategory = "business" | "system";

interface MetricCardModel {
  label: string;
  value: string;
  delta: string;
  tone: OverviewMetric["tone"];
  description: string;
  badgeText: string;
  valueHint: string;
  path: string;
  category: MetricCategory;
}

const router = useRouter();
const dashboardState = ref<DashboardState>("loading");
const summary = reactive<DashboardSummary>({
  metrics: [],
  activity: [],
  jobs: [],
  health: [],
});

const quickEntries = [
  { group: "成员", title: "成员管理", note: "筛选成员并执行批量处理", path: "/users", icon: UserFilled },
  { group: "积分", title: "积分规则", note: "调整奖励、扣分和排行榜策略", path: "/points/config", icon: Coin },
  { group: "群组", title: "群组设置", note: "管理接入、权限和管理员", path: "/admin/config", icon: ChatDotRound },
  { group: "内容", title: "发布任务", note: "安排图文、视频与定时发布", path: "/posts", icon: Calendar },
  { group: "活动", title: "活动抽奖", note: "查看进行中与历史抽奖活动", path: "/lottery", icon: Trophy },
  { group: "分析", title: "数据分析", note: "查看趋势、活跃度和积分表现", path: "/stats", icon: DataAnalysis },
];

const hasSummaryContent = computed(
  () => summary.metrics.length > 0 || summary.activity.length > 0 || summary.jobs.length > 0 || summary.health.length > 0,
);

const metricCards = computed<MetricCardModel[]>(() => summary.metrics.map((metric, index) => buildMetricCard(metric, index)));

const primaryMetrics = computed(() => {
  const business = metricCards.value.filter((metric) => metric.category === "business");
  return (business.length ? business : metricCards.value).slice(0, 4);
});

const secondaryMetrics = computed(() => {
  const system = metricCards.value.filter((metric) => metric.category === "system");
  return system.slice(0, 3);
});

const jobRows = computed(() => summary.jobs.slice(0, 6));
const recentEvents = computed(() => summary.activity.slice(0, 4));
const healthRows = computed(() => summary.health.slice(0, 4));

const focusQueue = computed(() => {
  const items: Array<{ title: string; detail: string; tone: string; action: string; path: string }> = [];

  summary.jobs
    .filter((job) => job.status !== "live")
    .slice(0, 2)
    .forEach((job) => {
      items.push({
        title: `${job.title}${job.status === "failed" ? " 执行失败" : " 已暂停"}`,
        detail: `执行计划 ${job.schedule}，下次执行 ${job.nextRun}`,
        tone: job.status === "failed" ? "danger" : "warning",
        action: "去查看任务",
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
        action: item.status === "danger" ? "去处理异常" : "查看事件",
        path: item.status === "danger" ? "/violations" : "/stats",
      });
    });

  summary.health
    .filter((item) => item.value < 80)
    .slice(0, 1)
    .forEach((item) => {
      items.push({
        title: `${displayHealthLabel(item.label)}需要关注`,
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
      action: "查看数据分析",
      path: "/stats",
    });
  }

  return items.slice(0, 3);
});

const focusSummary = computed(() => {
  if (dashboardState.value === "loading" && !hasSummaryContent.value) return "正在更新概览";
  const risky = focusQueue.value.filter((item) => item.tone !== "success").length;
  const weakHealth = summary.health.filter((item) => item.value < 80).length;
  if (risky > 0 || weakHealth > 0) return `${risky} 项待处理，${weakHealth} 个健康项需关注`;
  return "当前运行平稳，可继续推进日常运营";
});

function buildMetricCard(metric: OverviewMetric, index: number): MetricCardModel {
  const rawLabel = metric.label.trim();
  const numericValue = parseMetricNumber(metric.value);

  const presets = [
    {
      match: /活跃用户|成员活跃/,
      label: "活跃用户",
      description: "近 24 小时产生互动或积分变化的成员数量。",
      zeroHint: "当前还没有活跃成员，说明今天尚未出现互动或奖励变化。",
      category: "business",
      path: "/users",
    },
    {
      match: /违规|风险/,
      label: "待处理违规",
      description: "需要尽快查看和处理的风险记录。",
      zeroHint: "当前没有待处理违规，这属于正常状态。",
      category: "business",
      path: "/violations",
    },
    {
      match: /抽奖|活动/,
      label: "进行中活动",
      description: "正在进行或等待开奖的活动数量。",
      zeroHint: "当前没有进行中的活动，发布新活动后这里会更新。",
      category: "business",
      path: "/lottery",
    },
    {
      match: /等级/,
      label: "积分等级配置",
      description: "当前积分成长体系、门槛与头衔配置。",
      zeroHint: "还没配置积分等级，建议先完善成长门槛和头衔显示。",
      category: "business",
      path: "/levels",
    },
    {
      match: /任务|定时/,
      label: "发布任务",
      description: "排队中、运行中或暂停中的发布任务数量。",
      zeroHint: "当前没有待执行任务，说明今日发布计划尚未排期。",
      category: "system",
      path: "/posts",
    },
    {
      match: /扫描|调度/,
      label: "定时任务巡检",
      description: "定时任务巡检与调度链路的最近状态。",
      zeroHint: "当前巡检正常，没有发现任务积压或异常。",
      category: "system",
      path: "/stats",
    },
    {
      match: /消息|投递/,
      label: "消息发送情况",
      description: "机器人消息发送与互动链路的近期表现。",
      zeroHint: "当前没有新的发送记录，可结合数据分析继续查看。",
      category: "system",
      path: "/stats",
    },
  ] as const;

  const matched = presets.find((item) => item.match.test(rawLabel));
  const category: MetricCategory = matched?.category ?? (index < 4 ? "business" : "system");
  const label = matched?.label ?? rawLabel;
  const description = matched?.description ?? (category === "business" ? "用于判断当前社群运营状态。" : "用于判断当前任务与系统状态。");
  const badgeText = metric.delta || (category === "business" ? "当前范围" : "运行状态");
  const zeroHint = matched?.zeroHint ?? (category === "business" ? "当前暂无相关业务数据。" : "当前暂无相关运行数据。");
  const valueHint = numericValue === 0 ? zeroHint : "点击进入详情页继续查看。";

  return {
    label,
    value: metric.value,
    delta: metric.delta,
    tone: metric.tone,
    description,
    badgeText,
    valueHint,
    path: matched?.path ?? "/stats",
    category,
  };
}

function parseMetricNumber(value: string): number | null {
  const normalized = value.replace(/,/g, "").match(/-?\d+(\.\d+)?/);
  if (!normalized) return null;
  const parsed = Number(normalized[0]);
  return Number.isFinite(parsed) ? parsed : null;
}

function displayHealthLabel(label: string): string {
  if (/扫描|调度/.test(label)) return "定时任务巡检结果";
  return label;
}

async function loadDashboard(): Promise<void> {
  dashboardState.value = "loading";
  try {
    const response = await fetchDashboardSummary();
    Object.assign(summary, response);
    dashboardState.value = "ready";
  } catch {
    Object.assign(summary, { metrics: [], activity: [], jobs: [], health: [] });
    dashboardState.value = "error";
    ElMessage.error("运营总览加载失败");
  }
}

function navigateTo(path?: string): void {
  if (path) {
    router.push(path);
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
  return "执行失败";
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
  color: var(--app-muted-strong);
}

.dashboard-skeleton {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.skeleton-card,
.skeleton-panel {
  width: 100%;
}

.metric-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.metric-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.metric-section-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.metric-kicker {
  color: var(--app-muted);
  font-size: 12px;
  font-weight: 600;
}

.metric-section-copy strong {
  font-size: 15px;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(300px, 0.9fr);
  gap: 18px;
  align-items: start;
}

.dashboard-main,
.dashboard-side {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.queue-list,
.activity-list,
.action-list {
  display: grid;
  gap: 10px;
}

.queue-item,
.activity-item,
.action-card,
.system-chip {
  width: 100%;
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.02);
  color: var(--app-text);
  text-align: left;
  transition: border-color 0.18s ease, background 0.18s ease, transform 0.18s ease;
}

.queue-item,
.action-card,
.system-chip {
  cursor: pointer;
}

.queue-item:hover,
.action-card:hover,
.system-chip:hover {
  border-color: rgba(132, 170, 255, 0.12);
  background: rgba(255, 255, 255, 0.03);
  transform: translateY(-1px);
}

.queue-item {
  padding: 14px 15px;
}

.queue-item[data-tone="success"] {
  border-color: rgba(114, 192, 145, 0.12);
}

.queue-item[data-tone="warning"] {
  border-color: rgba(216, 162, 95, 0.14);
}

.queue-item[data-tone="danger"] {
  border-color: rgba(210, 120, 120, 0.14);
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
.activity-time,
.action-group,
.health-copy span {
  color: var(--app-muted);
  font-size: 12px;
}

.queue-item p,
.activity-item p,
.action-card p {
  margin: 8px 0 0;
  color: var(--app-muted-strong);
  font-size: 12px;
  line-height: 1.6;
}

.table-shell {
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 12px;
  background: rgba(10, 15, 22, 0.42);
}

.job-table {
  width: 100%;
}

.system-strip {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 14px;
}

.system-chip {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px;
}

.system-chip span {
  color: var(--app-muted);
  font-size: 12px;
}

.system-chip strong {
  font-size: 18px;
  font-weight: 700;
}

.system-chip[data-tone="success"] {
  border-color: rgba(114, 192, 145, 0.12);
}

.system-chip[data-tone="warning"] {
  border-color: rgba(216, 162, 95, 0.14);
}

.system-chip[data-tone="danger"] {
  border-color: rgba(210, 120, 120, 0.14);
}

.health-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
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
  padding: 14px 15px;
}

.activity-item[data-tone="warning"] {
  border-color: rgba(216, 162, 95, 0.14);
}

.activity-item[data-tone="danger"] {
  border-color: rgba(210, 120, 120, 0.14);
}

.action-list {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.action-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  min-height: 136px;
  padding: 14px;
}

.action-card-main {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.action-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.action-title strong {
  font-size: 14px;
}

.action-card p {
  margin: 0;
  max-width: none;
  text-align: left;
}

.state-block-large {
  align-items: flex-start;
  padding: 18px;
}

.state-block-inline {
  margin: 10px;
}

.state-block-compact {
  min-height: 180px;
  justify-content: center;
}

@media (max-width: 1180px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }

  .action-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .metric-section-copy strong {
    font-size: 14px;
  }

  .queue-head,
  .activity-head,
  .health-copy,
  .action-card {
    flex-direction: column;
    align-items: flex-start;
  }

  .system-strip,
  .action-list {
    grid-template-columns: 1fr;
  }
}
</style>
