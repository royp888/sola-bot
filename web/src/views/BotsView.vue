<template>
  <div class="page">
    <PageHeader
      eyebrow="机器人管理"
      title="机器人管理"
      description="管理多个 Telegram 机器人 的在线状态、绑定关系和语言配置，心跳时间按北京时间显示。"
    >
      <template #actions>
        <el-input v-model="keyword" class="search" placeholder="搜索机器人">
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button :icon="Refresh" @click="loadBots">刷新</el-button>
        <el-button type="primary" :icon="Plus" disabled>新增机器人</el-button>
      </template>
    </PageHeader>

    <PanelSection title="机器人列表" description="来自后端 /bots 的实时清单。">
      <div class="table-wrap">
      <el-table :data="filteredBots" stripe>
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="username" label="用户名" min-width="180" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="botTag(row.status)" effect="dark">{{ botStatus(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="boundChats" label="绑定聊天" width="110" />
        <el-table-column label="心跳" min-width="180">
          <template #default="{ row }">{{ formatDateTime(row.lastHeartbeat) }}</template>
        </el-table-column>
        <el-table-column prop="language" label="语言" width="120" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button text type="primary" @click="showDetail(row)">详情</el-button>
            <el-button text disabled>同步</el-button>
          </template>
        </el-table-column>
      </el-table>
      </div>
    </PanelSection>

    <PanelSection title="Bot 全局配置" description="接口：GET/PUT /api/v1/bot/config。修改后立即生效。">
      <template #actions>
        <el-button :icon="Refresh" :loading="configLoading" @click="loadBotConfig">刷新</el-button>
        <el-button type="primary" :loading="configSaving" @click="saveBotConfig">保存</el-button>
      </template>
      <el-form v-loading="configLoading" label-position="top" class="config-form">
        <div class="switch-row">
          <div>
            <strong>Bot 总开关</strong>
            <span>关闭后机器人停止响应所有群组消息</span>
          </div>
          <el-switch v-model="config.enabled" />
        </div>
        <div class="switch-row">
          <div>
            <strong>积分系统</strong>
            <span>全局开关，关闭后所有群组积分功能停用</span>
          </div>
          <el-switch v-model="config.enable_points" />
        </div>
        <div class="switch-row">
          <div>
            <strong>统计追踪</strong>
            <span>开启后记录消息量、活跃度等统计数据</span>
          </div>
          <el-switch v-model="config.enable_stats_tracking" />
        </div>
        <div class="switch-row">
          <div>
            <strong>允许转发消息计分</strong>
            <span>关闭后转发的消息不计入积分</span>
          </div>
          <el-switch v-model="config.allow_forwarded_posts" />
        </div>
        <div class="switch-row">
          <div>
            <strong>自动删除消息</strong>
            <span>开启后 Bot 发送的消息会在指定时间后自动删除</span>
          </div>
          <el-switch v-model="config.auto_delete_enabled" />
        </div>
        <el-form-item v-if="config.auto_delete_enabled" label="自动删除延迟（秒）">
          <el-input-number v-model="config.auto_delete_after_secs" :min="10" :max="86400" controls-position="right" style="width:200px" />
        </el-form-item>
        <el-form-item label="默认语言">
          <el-select v-model="config.default_language" style="width:200px">
            <el-option label="中文" value="zh" />
            <el-option label="English" value="en" />
          </el-select>
        </el-form-item>
        <el-form-item label="时区">
          <el-input v-model="config.time_zone" placeholder="Asia/Shanghai" style="width:200px" />
        </el-form-item>
      </el-form>
    </PanelSection>

    <el-dialog v-model="detailVisible" title="机器人详情" width="520px">
      <el-descriptions v-if="currentBot" :column="1" border>
        <el-descriptions-item label="名称">{{ currentBot.name }}</el-descriptions-item>
        <el-descriptions-item label="用户名">{{ currentBot.username }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ botStatus(currentBot.status) }}</el-descriptions-item>
        <el-descriptions-item label="绑定聊天">{{ currentBot.boundChats }}</el-descriptions-item>
        <el-descriptions-item label="语言">{{ currentBot.language }}</el-descriptions-item>
        <el-descriptions-item label="心跳">{{ formatDateTime(currentBot.lastHeartbeat) }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { Plus, Refresh, Search } from "@element-plus/icons-vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { fetchBots } from "@/api/bots";
import { fetchBotConfig, updateBotConfig } from "@/api/botConfig";
import type { BotConfig } from "@/api/botConfig";
import type { BotRecord } from "@/types/api";
import { formatDateTime } from "@/utils/helpers";

const keyword = ref("");
const bots = ref<BotRecord[]>([]);
const detailVisible = ref(false);
const currentBot = ref<BotRecord>();
const configLoading = ref(false);
const configSaving = ref(false);
const config = reactive<BotConfig>({
  enabled: true,
  default_language: "zh",
  time_zone: "Asia/Shanghai",
  auto_delete_enabled: false,
  auto_delete_after_secs: 0,
  allow_forwarded_posts: true,
  enable_stats_tracking: true,
  enable_points: true,
});

const filteredBots = computed(() => {
  const value = keyword.value.trim().toLowerCase();
  if (!value) {
    return bots.value;
  }

  return bots.value.filter((bot) => {
    return [bot.name, bot.username, bot.language].some((field) =>
      field.toLowerCase().includes(value),
    );
  });
});

async function loadBots(): Promise<void> {
  try {
    bots.value = await fetchBots();
  } catch {
    bots.value = [];
    ElMessage.error("机器人列表暂时不可用");
  }
}

function botTag(status: BotRecord["status"]): "success" | "warning" | "danger" {
  if (status === "online") return "success";
  if (status === "degraded") return "warning";
  return "danger";
}

function botStatus(status: BotRecord["status"]): string {
  if (status === "online") return "在线";
  if (status === "degraded") return "降级";
  return "离线";
}

function showDetail(bot: BotRecord): void {
  currentBot.value = bot;
  detailVisible.value = true;
}

async function loadBotConfig(): Promise<void> {
  configLoading.value = true;
  try {
    const result = await fetchBotConfig();
    Object.assign(config, result);
  } catch {
    ElMessage.error("获取 Bot 配置失败");
  } finally {
    configLoading.value = false;
  }
}

async function saveBotConfig(): Promise<void> {
  configSaving.value = true;
  try {
    const result = await updateBotConfig({ ...config });
    Object.assign(config, result);
    ElMessage.success("Bot 全局配置已保存");
  } catch {
    ElMessage.error("保存失败，请重试");
  } finally {
    configSaving.value = false;
  }
}

onMounted(loadBots);
onMounted(loadBotConfig);
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.search {
  width: 220px;
}

.config-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px solid var(--app-border);
}

.switch-row strong {
  display: block;
  margin-bottom: 4px;
}

.switch-row span {
  color: var(--app-muted);
  font-size: 13px;
}

@media (max-width: 720px) {
  .search {
    width: 100%;
  }

  .switch-row {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
