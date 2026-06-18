<template>
  <div class="page">
    <PageHeader eyebrow="关键词运营" title="关键词规则" description="维护关键词匹配、处理动作和群组作用范围。">
      <template #actions>
        <ChatSelect v-model="filters.chatId" @update:model-value="loadKeywords" />
        <el-select v-model="filters.action" class="filter-select" clearable placeholder="动作" @change="loadKeywords">
          <el-option label="记录" value="log" />
          <el-option label="警告" value="warn" />
          <el-option label="删除" value="delete" />
          <el-option label="封禁" value="ban" />
        </el-select>
        <el-button :icon="Refresh" :loading="loading" @click="loadKeywords">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">新建规则</el-button>
      </template>
    </PageHeader>

    <PanelSection title="关键词列表" description="接口：GET/POST/PATCH/DELETE /api/keywords。">
      <el-alert v-if="error" class="alert" type="error" :closable="false" show-icon title="服务暂时不可用" />
      <el-table :data="keywords" stripe v-loading="loading">
        <el-table-column prop="pattern" label="关键词" min-width="160" />
        <el-table-column prop="chat_id" label="群组" min-width="120" />
        <el-table-column prop="match_type" label="匹配" width="110" />
        <el-table-column prop="scope" label="范围" min-width="120" />
        <el-table-column prop="action" label="动作" min-width="120" />
        <el-table-column prop="reply_text" label="回复" min-width="180" />
        <el-table-column label="启用" width="110">
          <template #default="{ row }">
            <el-switch :model-value="row.enabled" :loading="togglingId === row.id" @change="toggleKeyword(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" :loading="deletingId === row.id" @click="removeKeyword(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <el-dialog v-model="dialogVisible" :title="editingKeyword ? '编辑关键词' : '新建关键词'" width="560px">
      <el-form label-position="top">
        <el-form-item label="群组 ID">
          <el-input v-if="editingKeyword" v-model="form.chat_id" disabled />
          <ChatSelect v-else v-model="form.chat_id" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="关键词">
              <el-input v-model="form.pattern" maxlength="128" />
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
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="范围">
              <el-input v-model="form.scope" placeholder="全局 / 群组 / 私聊" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="动作">
              <el-select v-model="form.action" class="wide-control">
                <el-option label="记录" value="log" />
                <el-option label="警告" value="warn" />
                <el-option label="删除" value="delete" />
                <el-option label="封禁" value="ban" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="回复文本">
          <el-input v-model="form.reply_text" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitKeyword">保存</el-button>
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
import { createKeyword, deleteKeyword, fetchKeywords, updateKeyword } from "@/api/keywords";
import type { ChatID, KeywordPayload, KeywordRecord } from "@/types/api";
import { parseNumericId } from "@/utils/helpers";

const loading = ref(false);
const saving = ref(false);
const error = ref(false);
const dialogVisible = ref(false);
const keywords = ref<KeywordRecord[]>([]);
const editingKeyword = ref<KeywordRecord>();
const togglingId = ref<ChatID>();
const deletingId = ref<ChatID>();
const filters = reactive({ chatId: "", action: "" });
const form = reactive({
  chat_id: "",
  pattern: "",
  match_type: "contains",
  action: "warn",
  scope: "chat",
  reply_text: "",
  enabled: true,
});

function resetForm(): void {
  Object.assign(form, {
    chat_id: filters.chatId,
    pattern: "",
    match_type: "contains",
    action: "warn",
    scope: "chat",
    reply_text: "",
    enabled: true,
  });
  editingKeyword.value = undefined;
}

async function loadKeywords(): Promise<void> {
  loading.value = true;
  error.value = false;
  try {
    keywords.value = await fetchKeywords({
      chatId: filters.chatId,
      action: filters.action,
    });
  } catch {
    keywords.value = [];
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

function openEdit(row: KeywordRecord): void {
  editingKeyword.value = row;
  Object.assign(form, {
    chat_id: String(row.chat_id),
    pattern: row.pattern,
    match_type: row.match_type || "contains",
    action: row.action,
    scope: row.scope || "chat",
    reply_text: row.reply_text || "",
    enabled: row.enabled,
  });
  dialogVisible.value = true;
}

async function submitKeyword(): Promise<void> {
  const chatId = parseNumericId(form.chat_id);
  if (!editingKeyword.value && !chatId) {
    ElMessage.warning("请输入有效的群组 ID");
    return;
  }
  if (!form.pattern.trim()) {
    ElMessage.warning("请输入关键词");
    return;
  }

  saving.value = true;
  try {
    const payload = {
      pattern: form.pattern.trim(),
      match_type: form.match_type,
      action: form.action,
      scope: form.scope.trim() || undefined,
      reply_text: form.reply_text.trim() || undefined,
      enabled: form.enabled,
    };
    if (editingKeyword.value) {
      await updateKeyword(editingKeyword.value.id, payload);
    } else {
      await createKeyword({ ...payload, chat_id: chatId } as KeywordPayload);
    }
    ElMessage.success("关键词规则已保存");
    dialogVisible.value = false;
    await loadKeywords();
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    saving.value = false;
  }
}

async function toggleKeyword(row: KeywordRecord): Promise<void> {
  togglingId.value = row.id;
  try {
    const updated = await updateKeyword(row.id, { enabled: !row.enabled });
    Object.assign(row, updated);
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    togglingId.value = undefined;
  }
}

async function removeKeyword(row: KeywordRecord): Promise<void> {
  try {
    await ElMessageBox.confirm(`确认删除关键词「${row.pattern}」？`, "删除关键词", {
      type: "warning",
      confirmButtonText: "删除",
      cancelButtonText: "取消",
    });
  } catch {
    return;
  }
  deletingId.value = row.id;
  try {
    await deleteKeyword(row.id);
    ElMessage.success("关键词规则已删除");
    await loadKeywords();
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    deletingId.value = undefined;
  }
}

onMounted(loadKeywords);
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.filter-select {
  width: 180px;
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
