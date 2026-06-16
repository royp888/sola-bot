<template>
  <div class="page-stack">
    <PageHeader eyebrow="内容资产" title="内容模板" description="先明确作用范围，再维护可复用的文案与媒体模板。">
      <template #actions>
        <el-button :icon="Refresh" :loading="loading" @click="loadTemplates">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">新建模板</el-button>
      </template>
    </PageHeader>

    <div class="summary-grid">
      <div class="summary-card">
        <div class="summary-label">当前模板</div>
        <div class="summary-value">{{ templates.length }}</div>
        <div class="summary-meta">{{ selectedChatId ? '当前群组与全局模板合并展示' : '当前显示全部可见模板' }}</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">全局模板</div>
        <div class="summary-value">{{ globalCount }}</div>
        <div class="summary-meta">适用于多个群组的通用内容</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">群组专属</div>
        <div class="summary-value">{{ scopedCount }}</div>
        <div class="summary-meta">只在具体群组内使用的模板</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">加载状态</div>
        <div class="summary-value">{{ nextCursor ? '还有更多' : '已到底' }}</div>
        <div class="summary-meta">{{ nextCursor ? '可以继续加载下一页结果' : '当前结果已经完整展示' }}</div>
      </div>
    </div>

    <PanelSection title="模板列表" description="选定群组后，会同时显示该群专属模板和全局模板。">
      <template #actions>
        <div class="scope-toolbar panel-toolbar">
          <div class="scope-main scope-main-single">
            <div class="scope-field">
              <label>目标群组</label>
              <ChatSelect v-model="selectedChatId" @update:model-value="loadTemplates" />
            </div>
          </div>
          <div class="result-toolbar">
            <strong>{{ resultHeadline }}</strong>
            <div class="result-toolbar-meta">
              <span>{{ selectedChatId ? `当前群组 ${selectedChatId}` : '当前查看全部可见模板' }}</span>
              <span>全局 {{ globalCount }} 条</span>
              <span>群组专属 {{ scopedCount }} 条</span>
            </div>
          </div>
        </div>
      </template>

      <el-table class="table-compact" :data="templates" size="small" stripe empty-text="暂无模板">
        <el-table-column prop="name" label="模板名称" min-width="180" />
        <el-table-column prop="chat_id" label="作用范围" min-width="150">
          <template #default="{ row }">{{ row.chat_id ? `群组 ${row.chat_id}` : '全局模板' }}</template>
        </el-table-column>
        <el-table-column prop="media_type" label="内容类型" width="110">
          <template #default="{ row }">{{ mediaTypeLabel(row.media_type) }}</template>
        </el-table-column>
        <el-table-column prop="content" label="内容预览" min-width="280" show-overflow-tooltip />
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="removeTemplate(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!templates.length" class="empty-state template-empty">
        <strong>{{ emptyTitle }}</strong>
        <p>{{ emptyDescription }}</p>
      </div>

      <div v-if="nextCursor" class="load-more">
        <el-button :loading="loadingMore" @click="loadMoreTemplates">加载更多</el-button>
      </div>
    </PanelSection>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑模板' : '新建模板'" width="680px">
      <el-form label-position="top" class="template-form">
        <div class="form-section">
          <div class="form-section-title">
            <strong>作用范围</strong>
            <span>先决定模板是给单个群组使用，还是作为全局模板复用。</span>
          </div>
          <div class="form-grid">
            <div class="form-span-2 template-scope-toggle">
              <el-checkbox v-model="globalTemplate">保存为全局模板</el-checkbox>
            </div>
            <div v-if="!globalTemplate" class="form-span-2">
              <el-form-item label="目标群组">
                <ChatSelect v-model="formChatId" />
              </el-form-item>
            </div>
          </div>
        </div>

        <div class="form-section">
          <div class="form-section-title">
            <strong>模板内容</strong>
            <span>让名称、正文和媒体类型表达清楚使用场景。</span>
          </div>
          <div class="form-grid">
            <el-form-item label="模板名称">
              <el-input v-model="form.name" placeholder="例如：开奖提醒、群欢迎语" />
            </el-form-item>
            <el-form-item label="内容类型">
              <el-select v-model="form.media_type" class="wide-control">
                <el-option label="文字" value="text" />
                <el-option label="图片" value="photo" />
                <el-option label="视频" value="video" />
              </el-select>
            </el-form-item>
            <div class="form-span-2">
              <el-form-item label="正文内容">
                <el-input v-model="form.content" type="textarea" :rows="5" placeholder="输入模板正文，可用于公告、欢迎语或活动通知" />
              </el-form-item>
            </div>
            <div v-if="form.media_type !== 'text'" class="form-span-2">
              <el-form-item label="媒体地址">
                <el-input v-model="form.media_url" placeholder="https://example.com/file.jpg" />
              </el-form-item>
            </div>
            <el-form-item class="form-span-2" label="解析模式">
              <el-select v-model="form.parse_mode" class="wide-control" clearable placeholder="默认无特殊解析">
                <el-option label="无" value="" />
                <el-option label="HTML" value="HTML" />
                <el-option label="Markdown" value="Markdown" />
              </el-select>
            </el-form-item>
          </div>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitTemplate">保存模板</el-button>
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
import { createTemplate, deleteTemplate, fetchTemplates, updateTemplate } from "@/api/templates";
import type { ChatID, MessageTemplatePayload, MessageTemplateRecord } from "@/types/api";

const selectedChatId = ref<ChatID | "">("");
const formChatId = ref<ChatID | "">("");
const globalTemplate = ref(false);
const loading = ref(false);
const loadingMore = ref(false);
const saving = ref(false);
const dialogVisible = ref(false);
const editing = ref<MessageTemplateRecord>();
const templates = ref<MessageTemplateRecord[]>([]);
const nextCursor = ref("");
const form = reactive<MessageTemplatePayload>({
  name: "",
  content: "",
  media_type: "text",
  media_url: "",
  parse_mode: "",
});

const globalCount = computed(() => templates.value.filter((item) => !item.chat_id).length);
const scopedCount = computed(() => templates.value.filter((item) => item.chat_id).length);
const resultHeadline = computed(() => {
  if (!templates.value.length) return "当前范围内还没有模板";
  return `已找到 ${templates.value.length} 条可复用模板`;
});
const emptyTitle = computed(() => (selectedChatId.value ? "这个群组还没有模板" : "当前还没有可用模板"));
const emptyDescription = computed(() =>
  selectedChatId.value
    ? "可以为这个群组创建专属模板，也可以直接使用全局模板。"
    : "建议先沉淀欢迎语、活动通知或常用公告模板，后续创建任务会更快。",
);

async function loadTemplates(): Promise<void> {
  loading.value = true;
  nextCursor.value = "";
  try {
    const response = await fetchTemplates(selectedChatId.value || undefined);
    templates.value = response.items;
    nextCursor.value = response.next_cursor || "";
  } catch {
    templates.value = [];
    nextCursor.value = "";
    ElMessage.error("模板接口不可用");
  } finally {
    loading.value = false;
  }
}

async function loadMoreTemplates(): Promise<void> {
  if (!nextCursor.value) return;
  loadingMore.value = true;
  try {
    const response = await fetchTemplates(selectedChatId.value || undefined, nextCursor.value);
    templates.value = templates.value.concat(response.items);
    nextCursor.value = response.next_cursor || "";
  } catch {
    ElMessage.error("更多模板加载失败");
  } finally {
    loadingMore.value = false;
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
  try {
    await deleteTemplate(row.id);
    ElMessage.success("模板已删除");
    await loadTemplates();
  } catch {
    ElMessage.error("删除失败");
  }
}

function mediaTypeLabel(value?: string): string {
  return {
    text: "文字",
    photo: "图片",
    video: "视频",
  }[value || "text"] || (value || "未知");
}

onMounted(loadTemplates);
</script>

<style scoped>
.panel-toolbar {
  width: min(100%, 920px);
}

.scope-main-single {
  grid-template-columns: minmax(0, 420px);
}

.wide-control {
  width: 100%;
}

.template-empty {
  padding-top: 18px;
}

.template-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.template-scope-toggle {
  display: flex;
  align-items: center;
}

.load-more {
  display: flex;
  justify-content: center;
  margin-top: 18px;
}
</style>
