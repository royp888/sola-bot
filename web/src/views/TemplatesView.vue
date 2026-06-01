<template>
  <div class="page">
    <PageHeader eyebrow="Content" title="消息模板库" description="维护可复用的文字、图片和视频发布模板。">
      <template #actions>
        <ChatSelect v-model="selectedChatId" />
        <el-button :icon="Refresh" :loading="loading" @click="loadTemplates">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">新建模板</el-button>
      </template>
    </PageHeader>

    <PanelSection title="模板列表" description="Chat ID 为空时为全局模板，选择群组时会同时显示全局模板和本群模板。">
      <el-table :data="templates" stripe>
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="chat_id" label="Chat" min-width="120">
          <template #default="{ row }">{{ row.chat_id || "全局" }}</template>
        </el-table-column>
        <el-table-column prop="media_type" label="类型" width="100" />
        <el-table-column prop="content" label="内容" min-width="240" show-overflow-tooltip />
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="removeTemplate(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑模板' : '新建模板'" width="560px">
      <el-form label-position="top">
        <el-form-item label="作用群组">
          <ChatSelect v-model="formChatId" />
        </el-form-item>
        <el-checkbox v-model="globalTemplate">保存为全局模板</el-checkbox>
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="媒体类型">
          <el-select v-model="form.media_type" class="wide-control">
            <el-option label="文字" value="text" />
            <el-option label="图片" value="photo" />
            <el-option label="视频" value="video" />
          </el-select>
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="form.content" type="textarea" :rows="4" />
        </el-form-item>
        <el-form-item v-if="form.media_type !== 'text'" label="媒体 URL">
          <el-input v-model="form.media_url" />
        </el-form-item>
        <el-form-item label="解析模式">
          <el-select v-model="form.parse_mode" class="wide-control" clearable>
            <el-option label="无" value="" />
            <el-option label="HTML" value="HTML" />
            <el-option label="Markdown" value="Markdown" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitTemplate">保存</el-button>
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
import { createTemplate, deleteTemplate, fetchTemplates, updateTemplate } from "@/api/templates";
import type { ChatID, MessageTemplatePayload, MessageTemplateRecord } from "@/types/api";

const selectedChatId = ref<ChatID | "">("");
const formChatId = ref<ChatID | "">("");
const globalTemplate = ref(false);
const loading = ref(false);
const saving = ref(false);
const dialogVisible = ref(false);
const editing = ref<MessageTemplateRecord>();
const templates = ref<MessageTemplateRecord[]>([]);
const form = reactive<MessageTemplatePayload>({
  name: "",
  content: "",
  media_type: "text",
  media_url: "",
  parse_mode: "",
});

async function loadTemplates(): Promise<void> {
  loading.value = true;
  try {
    templates.value = await fetchTemplates(selectedChatId.value || undefined);
  } catch {
    templates.value = [];
    ElMessage.error("模板接口不可用");
  } finally {
    loading.value = false;
  }
}

function openCreate(): void {
  editing.value = undefined;
  globalTemplate.value = false;
  formChatId.value = selectedChatId.value;
  Object.assign(form, { name: "", content: "", media_type: "text", media_url: "", parse_mode: "" });
  dialogVisible.value = true;
}

function openEdit(row: MessageTemplateRecord): void {
  editing.value = row;
  globalTemplate.value = !row.chat_id;
  formChatId.value = row.chat_id || "";
  Object.assign(form, {
    name: row.name,
    content: row.content,
    media_type: row.media_type || "text",
    media_url: row.media_url || "",
    parse_mode: row.parse_mode || "",
  });
  dialogVisible.value = true;
}

async function submitTemplate(): Promise<void> {
  if (!form.name.trim()) {
    ElMessage.warning("请填写模板名称");
    return;
  }
  const payload: MessageTemplatePayload = {
    ...form,
    chat_id: globalTemplate.value ? null : Number(formChatId.value),
  };
  if (!globalTemplate.value && !payload.chat_id) {
    ElMessage.warning("请选择群组或勾选全局模板");
    return;
  }
  saving.value = true;
  try {
    if (editing.value) {
      await updateTemplate(editing.value.id, payload);
    } else {
      await createTemplate(payload);
    }
    ElMessage.success("模板已保存");
    dialogVisible.value = false;
    await loadTemplates();
  } catch {
    ElMessage.error("模板保存失败");
  } finally {
    saving.value = false;
  }
}

async function removeTemplate(row: MessageTemplateRecord): Promise<void> {
  try {
    await ElMessageBox.confirm(`确认删除模板「${row.name}」？`, "删除模板", { type: "warning" });
  } catch {
    return;
  }
  await deleteTemplate(row.id);
  ElMessage.success("模板已删除");
  await loadTemplates();
}

onMounted(loadTemplates);
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.wide-control {
  width: 100%;
}
</style>
