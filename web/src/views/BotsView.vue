<template>
  <div class="page">
    <PageHeader
      eyebrow="Bots"
      title="机器人管理"
      description="管理多个 TG Bot 的在线状态、绑定关系和语言配置。"
    >
      <template #actions>
        <el-input v-model="keyword" class="search" placeholder="搜索 Bot">
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button :icon="Refresh" @click="loadBots">刷新</el-button>
        <el-button type="primary" :icon="Plus" disabled>新增 Bot</el-button>
      </template>
    </PageHeader>

    <PanelSection title="Bot 列表" description="来自后端 /bots 的实时清单。">
      <el-table :data="filteredBots" stripe>
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="username" label="Username" min-width="180" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="botTag(row.status)" effect="dark">{{ botStatus(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="boundChats" label="绑定聊天" width="110" />
        <el-table-column prop="lastHeartbeat" label="心跳" min-width="120" />
        <el-table-column prop="language" label="语言" width="120" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button text type="primary" @click="showDetail(row)">详情</el-button>
            <el-button text disabled>同步</el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <el-dialog v-model="detailVisible" title="Bot 详情" width="520px">
      <el-descriptions v-if="currentBot" :column="1" border>
        <el-descriptions-item label="名称">{{ currentBot.name }}</el-descriptions-item>
        <el-descriptions-item label="用户名">{{ currentBot.username }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ botStatus(currentBot.status) }}</el-descriptions-item>
        <el-descriptions-item label="绑定聊天">{{ currentBot.boundChats }}</el-descriptions-item>
        <el-descriptions-item label="语言">{{ currentBot.language }}</el-descriptions-item>
        <el-descriptions-item label="心跳">{{ currentBot.lastHeartbeat }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import { Plus, Refresh, Search } from "@element-plus/icons-vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { fetchBots } from "@/api/bots";
import type { BotRecord } from "@/types/api";

const keyword = ref("");
const bots = ref<BotRecord[]>([]);
const detailVisible = ref(false);
const currentBot = ref<BotRecord>();

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
    ElMessage.error("Bot 列表接口不可用");
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

onMounted(loadBots);
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

@media (max-width: 720px) {
  .search {
    width: 100%;
  }
}
</style>
