<script setup lang="ts">
import { ref, watch, onMounted } from "vue";
import { request } from "@/api/http";
import { useChatStore } from "@/mini/stores/chat";

const { selectedChat } = useChatStore();

interface DashboardData {
  chat_name?: string;
  member_count?: number;
  today_points?: number;
  violation_count?: number;
  verify_passed?: number;
}

const data = ref<DashboardData | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);

async function loadDashboard(): Promise<void> {
  const chat = selectedChat.value;
  if (!chat) {
    data.value = null;
    return;
  }

  const chatId = chat.chat_id ?? chat.id;
  loading.value = true;
  error.value = null;

  try {
    const resp = await request<any>(`/dashboard/summary?chat_id=${encodeURIComponent(String(chatId))}`);
    // Normalize response: could be a flat object or nested
    data.value = {
      chat_name: resp.chat_name ?? resp.title ?? chat.title ?? String(chatId),
      member_count: resp.member_count ?? resp.members ?? 0,
      today_points: resp.today_points ?? resp.points_today ?? 0,
      violation_count: resp.violation_count ?? resp.violations ?? 0,
      verify_passed: resp.verify_passed ?? resp.verifications ?? 0,
    };
  } catch (e: any) {
    error.value = e?.message || "加载仪表盘数据失败";
    data.value = null;
  } finally {
    loading.value = false;
  }
}

watch(selectedChat, () => {
  loadDashboard();
});

onMounted(() => {
  if (selectedChat.value) {
    loadDashboard();
  }
});
</script>

<template>
  <div class="scrollable">
    <div v-if="!selectedChat" class="empty">
      <div class="empty-icon">📊</div>
      <p>请先选择一个群组</p>
    </div>

    <div v-else-if="loading" class="spinner"></div>

    <div v-else-if="error" class="error">{{ error }}</div>

    <div v-else-if="data">
      <div class="card">
        <div class="card-title">当前群组</div>
        <div class="card-value">{{ data.chat_name }}</div>
      </div>

      <div class="card-grid">
        <div class="card">
          <div class="card-title">成员数</div>
          <div class="card-value">{{ data.member_count }}</div>
        </div>
        <div class="card">
          <div class="card-title">今日积分</div>
          <div class="card-value">{{ data.today_points }}</div>
        </div>
        <div class="card">
          <div class="card-title">违规数</div>
          <div class="card-value">{{ data.violation_count }}</div>
        </div>
        <div class="card">
          <div class="card-title">验证通过</div>
          <div class="card-value">{{ data.verify_passed }}</div>
        </div>
      </div>
    </div>

    <div v-else class="empty">
      <div class="empty-icon">📊</div>
      <p>暂无数据</p>
    </div>
  </div>
</template>
