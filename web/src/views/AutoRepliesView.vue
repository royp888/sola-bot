<template>
  <div class="page">
    <PageHeader eyebrow="自动化运营" title="自动回复" description="维护群内关键词自动回复，和关键词过滤规则相互独立。">
      <template #actions>
        <ChatSelect v-model="filters.chatId" @update:model-value="loadReplies" />
        <el-select v-model="filters.enabled" class="filter-select" clearable placeholder="状态" @change="loadReplies">
          <el-option label="启用" :value="true" />
          <el-option label="关闭" :value="false" />
        </el-select>
        <el-button :icon="Refresh" :loading="loading" @click="loadReplies">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">新建回复</el-button>
      </template>
    </PageHeader>

    <PanelSection title="回复列表" description="接口：GET/POST/PATCH/DELETE /api/auto-replies。">
      <el-alert v-if="error" class="alert" type="error" :closable="false" show-icon title="服务暂时不可用" />
      <el-table :data="replies" stripe v-loading="loading">
        <el-table-column prop="keyword" label="关键词" min-width="160" />
        <el-table-column prop="chat_id" label="群组" min-width="120" />
        <el-table-column prop="match_type" label="匹配" width="110">
          <template #default="{ row }">{{ matchTypeLabel(row.match_type) }}</template>
        </el-table-column>
        <el-table-column prop="reply_text" label="回复内容" min-width="260" show-overflow-tooltip />
        <el-table-column label="启用" width="110">
          <template #default="{ row }">
            <el-switch :model-value="row.enabled" :loading="togglingId === row.id" @change="toggleReply(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" :loading="deletingId === row.id" @click="removeReply(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <el-dialog v-model="dialogVisible" :title="editingReply ? '编辑自动回复' : '新建自动回复'" width="560px">
      <el-form label-position="top">
        <el-form-item label="群组 ID">
          <el-input v-if="editingReply" v-model="form.chat_id" disabled />
          <ChatSelect v-else v-model="form.chat_id" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="关键词">
              <el-input v-model="form.keyword" maxlength="128" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="匹配类型">
              <el-select v-model="form.match_type" class="wide-control">
                <el-option label="包含" value="contains" />
                <el-option label="精确" value="exact" />
                <el-option label="正则" value="regex" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="回复内容">
          <el-input v-model="form.reply_text" type="textarea" :rows="4" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitReply">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Plus, Refresh } from "@element-plus/icons-vue";
import ChatSelect from "@/components/ChatSelect.vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { createAutoReply, deleteAutoReply, fetchAutoReplies, updateAutoReply } from "@/api/autoReplies";
import type { AutoReplyPayload, AutoReplyRecord, ChatID } from "@/types/api";

const loading = ref(false);
const saving = ref(false);
const error = ref(false);
const dialogVisible = ref(false);
const replies = ref<AutoReplyRecord[]>([]);
const editingReply = ref<AutoReplyRecord>();
const togglingId = ref<ChatID>();
const deletingId = ref<ChatID>();
const filters = reactive<{ chatId: ChatID | ""; enabled: boolean | "" }>({ chatId: "", enabled: "" });
const form = reactive({
  chat_id: "",
  keyword: "",
  match_type: "contains",
  reply_text: "",
  enabled: true,
});

function parseNumericId(value: ChatID): number | undefined {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function matchTypeLabel(value: string): string {
  switch (value) {
    case "exact":
      return "精确";
    case "regex":
      return "正则";
    default:
      return "包含";
  }
}

function resetForm(): void {
  Object.assign(form, {
    chat_id: filters.chatId,
    keyword: "",
    match_type: "contains",
    reply_text: "",
    enabled: true,
  });
  editingReply.value = undefined;
}

async function loadReplies(): Promise<void> {
  loading.value = true;
  error.value = false;
  try {
    replies.value = await fetchAutoReplies({
      chatId: filters.chatId,
      enabled: filters.enabled === "" ? undefined : filters.enabled,
    });
  } catch {
    replies.value = [];
    error.value = true;
    ElMessage.error("服务暂时不可用");
  } finally {
    loading.value = false;
  }
}

function openCreate(): void {
  resetForm();
  dialogVisible.value = true;
}

function openEdit(row: AutoReplyRecord): void {
  editingReply.value = row;
  Object.assign(form, {
    chat_id: String(row.chat_id),
    keyword: row.keyword,
    match_type: row.match_type || "contains",
    reply_text: row.reply_text || "",
    enabled: row.enabled,
  });
  dialogVisible.value = true;
}

async function submitReply(): Promise<void> {
  const chatId = parseNumericId(form.chat_id);
  if (!editingReply.value && !chatId) {
    ElMessage.warning("请输入有效的群组 ID");
    return;
  }
  if (!form.keyword.trim()) {
    ElMessage.warning("请输入关键词");
    return;
  }
  if (!form.reply_text.trim()) {
    ElMessage.warning("请输入回复内容");
    return;
  }

  saving.value = true;
  try {
    const payload = {
      keyword: form.keyword.trim(),
      match_type: form.match_type,
      reply_text: form.reply_text.trim(),
      enabled: form.enabled,
    };
    if (editingReply.value) {
      await updateAutoReply(editingReply.value.id, payload);
    } else {
      await createAutoReply({ ...payload, chat_id: chatId } as AutoReplyPayload);
    }
    ElMessage.success("自动回复已保存");
    dialogVisible.value = false;
    await loadReplies();
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    saving.value = false;
  }
}

async function toggleReply(row: AutoReplyRecord): Promise<void> {
  togglingId.value = row.id;
  try {
    const updated = await updateAutoReply(row.id, { enabled: !row.enabled });
    Object.assign(row, updated);
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    togglingId.value = undefined;
  }
}

async function removeReply(row: AutoReplyRecord): Promise<void> {
  try {
    await ElMessageBox.confirm(`确认删除自动回复「${row.keyword}」？`, "删除自动回复", {
      type: "warning",
      confirmButtonText: "删除",
      cancelButtonText: "取消",
    });
  } catch {
    return;
  }
  deletingId.value = row.id;
  try {
    await deleteAutoReply(row.id);
    ElMessage.success("自动回复已删除");
    await loadReplies();
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    deletingId.value = undefined;
  }
}

onMounted(loadReplies);
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.filter-select {
  width: 140px;
}

.alert {
  margin-bottom: 12px;
}

.wide-control {
  width: 100%;
}

@media (max-width: 720px) {
  .filter-select {
    width: 100%;
  }
}
</style>
