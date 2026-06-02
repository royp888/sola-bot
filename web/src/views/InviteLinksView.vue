<template>
  <div class="page-stack">
    <PageHeader eyebrow="Growth" title="邀请链接追踪" description="创建专属邀请链接，并统计不同渠道带来的新增成员。">
      <template #actions>
        <el-button :icon="Refresh" :loading="loading" @click="loadLinks">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">创建链接</el-button>
      </template>
    </PageHeader>

    <div class="summary-grid">
      <div class="summary-card">
        <div class="summary-label">链接总数</div>
        <div class="summary-value">{{ links.length }}</div>
        <div class="summary-meta">当前列表已加载的邀请链接</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">累计加入</div>
        <div class="summary-value">{{ totalJoinCount }}</div>
        <div class="summary-meta">基于当前筛选范围统计</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">待审批链接</div>
        <div class="summary-value">{{ joinRequestCount }}</div>
        <div class="summary-meta">开启入群审批的渠道</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">平均转化</div>
        <div class="summary-value">{{ averageJoinCount }}</div>
        <div class="summary-meta">每条链接平均加入人数</div>
      </div>
    </div>

    <PanelSection title="邀请链接" description="需要 Bot 在群内拥有创建邀请链接权限。">
      <template #actions>
        <div class="panel-toolbar">
          <div class="control-cluster filters">
            <ChatSelect v-model="selectedChatId" class="filter-control" @update:model-value="loadLinks" />
            <el-select v-model="approvalFilter" class="filter-control">
              <el-option label="全部审批状态" value="all" />
              <el-option label="需审批" value="approval" />
              <el-option label="直接加入" value="instant" />
            </el-select>
          </div>
          <div class="filter-summary">
            <span>当前显示 {{ filteredLinks.length }} / {{ links.length }} 条链接</span>
          </div>
        </div>
      </template>

      <el-table :data="filteredLinks" stripe class="table-compact">
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="chat_id" label="Chat" min-width="120" />
        <el-table-column prop="invite_link" label="链接" min-width="260" show-overflow-tooltip />
        <el-table-column prop="join_count" label="加入数" width="100" sortable />
        <el-table-column label="审批" width="100">
          <template #default="{ row }">{{ row.creates_join_request ? "开启" : "关闭" }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="removeLink(row)">撤销</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="nextCursor" class="load-more">
        <el-button :loading="loadingMore" @click="loadMoreLinks">加载更多</el-button>
      </div>
    </PanelSection>

    <el-dialog v-model="dialogVisible" title="创建邀请链接" width="460px">
      <el-form label-position="top">
        <el-form-item label="Chat ID">
          <ChatSelect v-model="form.chat_id" />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="渠道 / 活动名称" />
        </el-form-item>
        <el-form-item label="入群审批">
          <el-switch v-model="form.creates_join_request" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitLink">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Plus, Refresh } from "@element-plus/icons-vue";
import ChatSelect from "@/components/ChatSelect.vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { createInviteLink, deleteInviteLink, fetchInviteLinks } from "@/api/inviteLinks";
import type { ChatID, InviteLinkPayload, InviteLinkRecord } from "@/types/api";

const selectedChatId = ref<ChatID | "">("");
const loading = ref(false);
const loadingMore = ref(false);
const saving = ref(false);
const dialogVisible = ref(false);
const links = ref<InviteLinkRecord[]>([]);
const nextCursor = ref("");
const approvalFilter = ref<"all" | "approval" | "instant">("all");
const form = reactive<InviteLinkPayload>({
  chat_id: "",
  name: "",
  creates_join_request: false,
});

const filteredLinks = computed(() => {
  return links.value.filter((item) => {
    if (approvalFilter.value === "approval" && !item.creates_join_request) {
      return false;
    }
    if (approvalFilter.value === "instant" && item.creates_join_request) {
      return false;
    }
    return true;
  });
});

const totalJoinCount = computed(() => filteredLinks.value.reduce((sum, item) => sum + Number(item.join_count || 0), 0));
const joinRequestCount = computed(() => filteredLinks.value.filter((item) => item.creates_join_request).length);
const averageJoinCount = computed(() => {
  if (!filteredLinks.value.length) return "0";
  return (totalJoinCount.value / filteredLinks.value.length).toFixed(totalJoinCount.value % filteredLinks.value.length === 0 ? 0 : 1);
});

async function loadLinks(): Promise<void> {
  loading.value = true;
  nextCursor.value = "";
  try {
    const response = await fetchInviteLinks(selectedChatId.value || undefined);
    links.value = response.items;
    nextCursor.value = response.next_cursor || "";
  } catch {
    links.value = [];
    nextCursor.value = "";
    ElMessage.error("邀请链接接口不可用");
  } finally {
    loading.value = false;
  }
}

async function loadMoreLinks(): Promise<void> {
  if (!nextCursor.value) return;
  loadingMore.value = true;
  try {
    const response = await fetchInviteLinks(selectedChatId.value || undefined, nextCursor.value);
    links.value = links.value.concat(response.items);
    nextCursor.value = response.next_cursor || "";
  } catch {
    ElMessage.error("更多邀请链接加载失败");
  } finally {
    loadingMore.value = false;
  }
}

function openCreate(): void {
  Object.assign(form, { chat_id: selectedChatId.value, name: "", creates_join_request: false });
  dialogVisible.value = true;
}

async function submitLink(): Promise<void> {
  if (!Number(form.chat_id)) {
    ElMessage.warning("请选择或输入 Chat ID");
    return;
  }
  saving.value = true;
  try {
    await createInviteLink({ ...form, chat_id: Number(form.chat_id) });
    ElMessage.success("邀请链接已创建");
    dialogVisible.value = false;
    await loadLinks();
  } catch {
    ElMessage.error("创建失败，请确认 Bot 权限");
  } finally {
    saving.value = false;
  }
}

async function removeLink(row: InviteLinkRecord): Promise<void> {
  try {
    await ElMessageBox.confirm(`确认撤销邀请链接「${row.name || row.id}」？`, "撤销链接", { type: "warning" });
  } catch {
    return;
  }
  await deleteInviteLink(row.id);
  ElMessage.success("邀请链接已撤销");
  await loadLinks();
}

onMounted(loadLinks);
</script>

<style scoped>
.panel-toolbar {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: min(100%, 920px);
}

.load-more {
  display: flex;
  justify-content: center;
  margin-top: 16px;
}

.filters :deep(.chat-select) {
  width: 100%;
}
</style>
