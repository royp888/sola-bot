<template>
  <div class="page">
    <PageHeader
      eyebrow="数据统计"
      title="统计分析"
      description="把命令、来源和活跃度拆开看，方便判断运营动作是否有效。"
    >
      <template #actions>
        <el-select v-model="range" class="select" @change="loadStats">
          <el-option label="近 7 天" value="7d" />
          <el-option label="近 30 天" value="30d" />
          <el-option label="近 90 天" value="90d" />
        </el-select>
        <el-button :icon="Refresh" @click="loadStats">刷新</el-button>
      </template>
    </PageHeader>

    <el-row :gutter="16">
      <el-col v-for="metric in summary.metrics" :key="metric.label" :xs="24" :sm="12" :lg="6">
        <StatTile v-bind="metric" />
      </el-col>
    </el-row>

    <el-row :gutter="16" class="stack">
      <el-col :xs="24" :lg="14">
        <PanelSection title="活跃趋势" description="消息、命令和新成员形成的综合活跃指数。">
          <div ref="activityChartRef" class="chart chart-large" />
        </PanelSection>
      </el-col>

      <el-col :xs="24" :lg="10">
        <PanelSection title="流量来源" description="命令和入口来源的组成。">
          <div ref="sourceChartRef" class="chart" />
        </PanelSection>
      </el-col>
    </el-row>

    <PanelSection title="积分排行用户" description="当前周期积分贡献最高的用户。">
      <div ref="pointsChartRef" class="chart chart-wide" />
    </PanelSection>

    <PanelSection title="积分排行用户明细" description="用户在当前周期内贡献的积分占比。">
      <el-table :data="summary.topPointsUsers" stripe>
        <el-table-column prop="rank" label="排名" width="90" />
        <el-table-column prop="label" label="用户" min-width="140" />
        <el-table-column prop="points" label="积分" min-width="140" />
        <el-table-column label="占比" min-width="140">
          <template #default="{ row }">
            <el-progress :percentage="row.share" :stroke-width="10" />
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { Refresh } from "@element-plus/icons-vue";
import { BarChart, LineChart, PieChart } from "echarts/charts";
import {
  GridComponent,
  LegendComponent,
  TooltipComponent,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";
import { use, init, type ECharts, type ComposeOption } from "echarts/core";
import type {
  BarSeriesOption,
  LineSeriesOption,
  PieSeriesOption,
} from "echarts/charts";
import type {
  GridComponentOption,
  LegendComponentOption,
  TooltipComponentOption,
} from "echarts/components";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import StatTile from "@/components/StatTile.vue";
import { fetchStats } from "@/api/stats";
import type { StatsOverview } from "@/types/api";

use([BarChart, LineChart, PieChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer]);

type ChartOption = ComposeOption<
  | BarSeriesOption
  | LineSeriesOption
  | PieSeriesOption
  | GridComponentOption
  | LegendComponentOption
  | TooltipComponentOption
>;

const range = ref("7d");
const activityChartRef = ref<HTMLDivElement>();
const sourceChartRef = ref<HTMLDivElement>();
const pointsChartRef = ref<HTMLDivElement>();
const summary = reactive<StatsOverview>({
  metrics: [],
  series: [],
  topPointsUsers: [],
  sources: [],
});
let activityChart: ECharts | undefined;
let sourceChart: ECharts | undefined;
let pointsChart: ECharts | undefined;

async function loadStats(): Promise<void> {
  try {
    const response = await fetchStats(range.value);
    Object.assign(summary, response);
    await nextTick();
    renderCharts();
  } catch {
    Object.assign(summary, { metrics: [], series: [], topPointsUsers: [], sources: [] });
    await nextTick();
    renderCharts();
    ElMessage.error("统计接口不可用");
  }
}

function renderCharts(): void {
  activityChart = ensureChart(activityChartRef.value, activityChart);
  sourceChart = ensureChart(sourceChartRef.value, sourceChart);
  pointsChart = ensureChart(pointsChartRef.value, pointsChart);

  const activityOption: ChartOption = {
    color: ["#5ecdc3"],
    grid: { left: 36, right: 18, top: 24, bottom: 28 },
    tooltip: { trigger: "axis" },
    xAxis: { type: "category", data: summary.series.map((item) => item.label), axisLine: { lineStyle: { color: "#2f3f46" } } },
    yAxis: { type: "value", max: 100, splitLine: { lineStyle: { color: "rgba(255,255,255,0.07)" } } },
    series: [
      {
        name: "活跃指数",
        type: "line",
        smooth: true,
        areaStyle: { color: "rgba(94,205,195,0.16)" },
        data: summary.series.map((item) => item.value),
      },
    ],
  };

  const sourceOption: ChartOption = {
    color: summary.sources.map((item) => item.color),
    tooltip: { trigger: "item" },
    legend: { show: false },
    series: [
      {
        name: "来源",
        type: "pie",
        radius: ["48%", "72%"],
        label: { color: "#d7e3e5" },
        data: summary.sources.map((item) => ({ name: item.label, value: item.value })),
      },
    ],
  };

  const pointsOption: ChartOption = {
    color: ["#f0b35d"],
    grid: { left: 42, right: 18, top: 24, bottom: 36 },
    tooltip: { trigger: "axis" },
    xAxis: {
      type: "category",
      data: summary.topPointsUsers.map((item) => item.label),
      axisLabel: { interval: 0, rotate: 20 },
      axisLine: { lineStyle: { color: "#2f3f46" } },
    },
    yAxis: { type: "value", splitLine: { lineStyle: { color: "rgba(255,255,255,0.07)" } } },
    series: [{ name: "积分", type: "bar", barMaxWidth: 28, data: summary.topPointsUsers.map((item) => item.points) }],
  };

  activityChart?.setOption(activityOption);
  sourceChart?.setOption(sourceOption);
  pointsChart?.setOption(pointsOption);
}

function ensureChart(el: HTMLDivElement | undefined, current: ECharts | undefined): ECharts | undefined {
  if (!el) return current;
  return current ?? init(el, "dark");
}

function resizeCharts(): void {
  activityChart?.resize();
  sourceChart?.resize();
  pointsChart?.resize();
}

onMounted(() => {
  void loadStats();
  window.addEventListener("resize", resizeCharts);
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", resizeCharts);
  activityChart?.dispose();
  sourceChart?.dispose();
  pointsChart?.dispose();
});
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.stack {
  margin-top: 0;
}

.select {
  width: 160px;
}

.chart {
  min-height: 280px;
  width: 100%;
}

.chart-large {
  min-height: 320px;
}

.chart-wide {
  min-height: 340px;
}

@media (max-width: 720px) {
  .select {
    width: 100%;
  }

  .chart,
  .chart-large,
  .chart-wide {
    min-height: 260px;
  }
}
</style>
