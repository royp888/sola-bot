<template>
  <div class="page">
    <PageHeader eyebrow="群组管理" title="封禁与警告" description="围绕群组执行封禁、解封和警告追踪，时间按北京时间显示。">
      <template #actions>
        <ChatSelect v-model="selectedChatId" @update:model-value="loadBans" />
        <el-button :icon="Refresh" :loading="loading" @click="loadBans">刷新</el-button>
      </template>
    </PageHeader>

    <el-row :gutter="16">
      <el-col :xs="24" :lg="16">
        <PanelSection title="封禁记录" description="对应 /api/admin/bans/:chatID 与 /api/admin/ban。">
          <div class="table-wrap">
          <el-empty v-if="!selectedChatId" description="请先在右上角选择群组" :image-size="80" />
          <el-table v-else :data="bans" stripe v-loading="loading">
            <el-table-column label="用户" min-width="140">
              <template #default="{ row }">{{ row.username || row.user_id }}</template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="200" />
            <el-table-column label="封禁时间" min-width="180">
              <template #default="{ row }">{{ formatDateTime(row.banned_at) }}</template>
            </el-table-column>
            <el-table-column label="解封时间" min-width="180">
              <template #default="{ row }">{{ formatDateTime(row.unbanned_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="110" fixed="right">
              <template #default="{ row }">
                <el-button size="small" :loading="deletingId === row.user_id" @click="submitUnban(row)">
                  解封
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          </div>
        </PanelSection>
      </el-col>

      <el-col :xs="24" :lg="8">
        <PanelSection title="后台操作" description="提供封禁 / 解封的表单骨架。">
          <el-form label-position="top">
            <el-form-item label="成员">
              <UserSelect v-model="banForm.user_id" :chat-id="selectedChatId" />
            </el-form-item>
            <el-form-item label="原因">
              <el-input v-model="banForm.reason" type="textarea" :rows="3" />
            </el-form-item>
            <el-button type="danger" :loading="saving" @click="submitBan">提交封禁</el-button>
          </el-form>
        </PanelSection>

        <PanelSection title="警告记录" description="选择用户后查看 warn_records。">
          <el-form label-position="top">
            <el-form-item label="成员">
              <UserSelect v-model="warnUserId" :chat-id="selectedChatId" />
            </el-form-item>
            <el-button :loading="warnLoading" @click="loadWarns">查看警告</el-button>
          </el-form>
          <div class="table-wrap warn-table">
          <el-table :data="warns" stripe>
            <el-table-column prop="reason" label="原因" min-width="140" />
            <el-table-column label="时间" min-width="180">
              <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.cleared ? 'info' : 'warning'" effect="dark">
                  {{ row.cleared ? "已清除" : "有效" }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
          </div>
        </PanelSection>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Refresh } from "@element-plus/icons-vue";
import ChatSelect from "@/components/ChatSelect.vue";
import UserSelect from "@/components/UserSelect.vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { createBan, deleteBan, fetchBans, fetchWarns } from "@/api/admin";
import type { BanRecord, ChatID, WarnRecord } from "@/types/api";
import { formatDateTime } from "@/utils/helpers";

const selectedChatId = ref("");
const loading = ref(false);
const saving = ref(false);
const warnLoading = ref(false);
const bans = ref<BanRecord[]>([]);
const warns = ref<WarnRecord[]>([]);
const deletingId = ref<ChatID>();
const warnUserId = ref("");
const banForm = reactive({ user_id: "", reason: "" });

async function loadBans(): Promise<void> {
  if (!selectedChatId.value) return;
  warns.value = [];
  warnUserId.value = "";
  loading.value = true;
  try {
    bans.value = await fetchBans(selectedChatId.value);
  } catch {
    bans.value = [];
    ElMessage.error("服务暂时不可用");
  } finally {
    loading.value = false;
  }
}

async function loadWarns(): Promise<void> {
  if (!selectedChatId.value || !warnUserId.value) {
    ElMessage.warning("请先选择群和用户");
    return;
  }
  warnLoading.value = true;
  try {
    warns.value = await fetchWarns(selectedChatId.value, warnUserId.value);
  } catch {
    warns.value = [];
    ElMessage.error("服务暂时不可用");
  } finally {
    warnLoading.value = false;
  }
}

async function submitBan(): Promise<void> {
  if (!selectedChatId.value || !banForm.user_id) {
    ElMessage.warning("请先选择群组和成员");
    return;
  }
  try {
    await ElMessageBox.confirm(
      `确认封禁用户 ${banForm.user_id}？此操作需手动解封。`,
      "确认封禁",
      {
        type: "warning",
        confirmButtonText: "确认封禁",
        cancelButtonText: "取消",
      },
    );
  } catch {
    return;
  }
  saving.value = true;
  try {
    await createBan({
      chat_id: selectedChatId.value,
      user_id: banForm.user_id,
      reason: banForm.reason,
    });
    ElMessage.success("封禁请求已提交");
    banForm.user_id = "";
    banForm.reason = "";
    await loadBans();
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    saving.value = false;
  }
}

async function submitUnban(row: BanRecord): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `确认解封用户 ${row.username || row.user_id}？`,
      "确认解封",
      {
        type: "warning",
        confirmButtonText: "确认解封",
        cancelButtonText: "取消",
      },
    );
  } catch {
    return;
  }
  deletingId.value = row.user_id;
  try {
    await deleteBan(row.chat_id, row.user_id);
    ElMessage.success("解封请求已提交");
    await loadBans();
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    deletingId.value = undefined;
  }
}

onMounted(loadBans);
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.warn-table {
  margin-top: 12px;
}

@media (max-width: 720px) {
}
</style>

