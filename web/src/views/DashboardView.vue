<template>
  <div class="page">
    <PageHeader
      eyebrow="Overview"
      title="运营概览"
      description="看板、任务、同步和健康状态都放在同一屏，先把日常操作跑顺。"
    >
      <template #actions>
        <el-button :icon="Refresh" @click="loadDashboard">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="router.push('/posts')">新建任务</el-button>
      </template>
    </PageHeader>

    <el-row :gutter="16">
      <el-col v-for="metric in summary.metrics" :key="metric.label" :xs="24" :sm="12" :lg="6">
        <StatTile v-bind="metric" />
      </el-col>
    </el-row>

    <PanelSection title="运营入口" description="常用功能入口已按积分、群管、规则和内容分组。">
      <div class="quick-grid">
        <button v-for="entry in quickEntries" :key="entry.path" class="quick-entry" @click="router.push(entry.path)">
          <span>{{ entry.group }}</span>
          <strong>{{ entry.title }}</strong>
        </button>
      </div>
    </PanelSection>

    <div class="stack">
      <PanelSection title="近期事件" description="采集到的关键操作与系统事件。">
        <el-timeline class="timeline">
          <el-timeline-item
            v-for="item in summary.activity"
            :key="item.title + item.time"
            :type="timelineType(item.status)"
            :timestamp="item.time"
          >
            <strong>{{ item.title }}</strong>
            <p>{{ item.detail }}</p>
          </el-timeline-item>
        </el-timeline>
      </PanelSection>

      <el-row :gutter="16">
        <el-col :xs="24" :lg="14">
          <PanelSection title="定时任务" description="队列和调度器一起保证发布稳定。">
            <el-table :data="summary.jobs" size="small" stripe>
              <el-table-column prop="title" label="任务" min-width="160" />
              <el-table-column prop="schedule" label="计划" min-width="120" />
              <el-table-column prop="nextRun" label="下次执行" min-width="120" />
              <el-table-column label="状态" width="120">
                <template #default="{ row }">
                  <el-tag :type="jobTag(row.status)" effect="dark">{{ jobLabel(row.status) }}</el-tag>
                </template>
              </el-table-column>
            </el-table>
          </PanelSection>
        </el-col>

        <el-col :xs="24" :lg="10">
          <PanelSection title="系统健康" description="接口、队列和命令触发的实时状态。">
            <div class="health-list">
              <div v-for="item in summary.health" :key="item.label" class="health-row">
                <div class="health-copy">
                  <strong>{{ item.label }}</strong>
                  <span>{{ item.note }}</span>
                </div>
                <el-progress :percentage="item.value" :stroke-width="10" />
              </div>
            </div>
          </PanelSection>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { Plus, Refresh } from "@element-plus/icons-vue";
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
  { group: "Points", title: "用户与积分", path: "/users" },
  { group: "Points", title: "积分配置", path: "/points/config" },
  { group: "Admin", title: "群组配置", path: "/admin/config" },
  { group: "Rules", title: "等级规则", path: "/levels" },
  { group: "Rules", title: "关键词规则", path: "/keywords" },
  { group: "Rules", title: "违规记录", path: "/violations" },
  { group: "Content", title: "定时发帖", path: "/posts" },
  { group: "Lottery", title: "抽奖管理", path: "/lottery" },
];

async function loadDashboard(): Promise<void> {
  try {
    const response = await fetchDashboardSummary();
    Object.assign(summary, response);
  } catch {
    Object.assign(summary, { metrics: [], activity: [], jobs: [], health: [] });
    ElMessage.error("运营概览接口不可用");
  }
}

function timelineType(status: DashboardSummary["activity"][number]["status"]): "success" | "warning" | "danger" | "info" {
  if (status === "success") return "success";
  if (status === "warning") return "warning";
  if (status === "danger") return "danger";
  return "info";
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

onMounted(loadDashboard);
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.stack {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(132px, 1fr));
  gap: 10px;
}

.quick-entry {
  display: flex;
  min-height: 76px;
  cursor: pointer;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 6px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  padding: 12px;
  color: var(--app-text);
  background: rgba(255, 255, 255, 0.03);
  text-align: left;
}

.quick-entry:hover {
  border-color: rgba(94, 205, 195, 0.52);
  background: rgba(94, 205, 195, 0.08);
}

.quick-entry span {
  color: var(--app-muted);
  font-size: 12px;
}

.timeline strong {
  display: block;
  margin-bottom: 4px;
}

.timeline p {
  margin: 0;
  color: var(--app-muted);
}

.health-list {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.health-row {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.health-copy {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}

.health-copy strong {
  font-size: 14px;
}

.health-copy span {
  color: var(--app-muted);
  font-size: 12px;
}

@media (max-width: 1180px) {
  .quick-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .quick-grid {
    grid-template-columns: 1fr;
  }
}
</style>
