<script setup lang="ts">
import { ref, watch, onMounted } from "vue";
import { request } from "@/api/http";
import { useChatStore } from "@/mini/stores/chat";
import type { LotteryRecord } from "@/types/api";

const { selectedChat } = useChatStore();

const lotteries = ref<LotteryRecord[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

function getChatId(): string | null {
  const chat = selectedChat.value;
  if (!chat) return null;
  return String(chat.chat_id ?? (chat as any).id ?? "");
}

async function loadLotteries(): Promise<void> {
  const chatId = getChatId();
  if (!chatId) {
    lotteries.value = [];
    return;
  }

  loading.value = true;
  error.value = null;

  try {
    const resp = await request<LotteryRecord[]>(
      `/lottery?chat_id=${encodeURIComponent(chatId)}&status=active`
    );
    lotteries.value = Array.isArray(resp) ? resp : [];
  } catch (e: any) {
    error.value = e?.message || "加载抽奖列表失败";
    lotteries.value = [];
  } finally {
    loading.value = false;
  }
}

function formatTime(iso: string | null | undefined): string {
  if (!iso) return "-";
  try {
    const d = new Date(iso);
    const now = Date.now();
    const diff = d.getTime() - now;
    if (diff < 0) return "已结束";
    const hours = Math.floor(diff / 3600000);
    const mins = Math.floor((diff % 3600000) / 60000);
    if (hours > 24) {
      const days = Math.floor(hours / 24);
      return `剩余 ${days} 天`;
    }
    return `剩余 ${hours} 小时 ${mins} 分钟`;
  } catch {
    return iso;
  }
}

function getParticipants(lottery: LotteryRecord): number {
  return lottery.participants ?? lottery.entry_count ?? 0;
}

function getWinnerCount(lottery: LotteryRecord): number {
  return lottery.winner_count_done ?? lottery.winner_count ?? 0;
}

function openAdmin(): void {
  window.open("/", "_blank");
}

watch(selectedChat, () => {
  loadLotteries();
});

onMounted(() => {
  if (selectedChat.value) {
    loadLotteries();
  }
});
</script>

<template>
  <div class="scrollable">
    <div v-if="!selectedChat" class="empty">
      <div class="empty-icon">🎁</div>
      <p>请先选择一个群组</p>
    </div>

    <div v-else-if="loading" class="spinner"></div>

    <div v-else-if="error" class="error">{{ error }}</div>

    <div v-else-if="lotteries.length > 0">
      <div
        v-for="lottery in lotteries"
        :key="String(lottery.id)"
        class="lottery-card"
      >
        <div class="lottery-card-header">
          <div class="lottery-card-title">{{ lottery.title }}</div>
          <span class="badge" :class="lottery.status === 'active' ? 'badge-success' : 'badge-warning'">
            {{ lottery.status === 'active' ? '进行中' : lottery.status }}
          </span>
        </div>
        <div class="lottery-card-prize">🎁 {{ lottery.prize }}</div>
        <div class="lottery-card-meta">
          <span>👥 {{ getParticipants(lottery) }} / {{ lottery.max_participants }}</span>
          <span>🏆 {{ getWinnerCount(lottery) }} 名</span>
          <span>⏱ {{ formatTime(lottery.end_at) }}</span>
        </div>
      </div>
    </div>

    <div v-else class="empty">
      <div class="empty-icon">🎁</div>
      <p>暂无进行中的抽奖</p>
    </div>

    <button class="btn btn-secondary btn-block" style="margin-top: 16px;" @click="openAdmin">
      前往后台管理抽奖
    </button>
  </div>
</template>
