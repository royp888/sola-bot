<template>
  <div class="page-stack">
    <PageHeader eyebrow="内容资产" title="快捷键盘" description="定义可复用的内联按钮键盘，发帖时一键挂载到消息下方。">
      <template #actions>
        <el-button :icon="Refresh" :loading="loading" @click="loadTemplates">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">新建键盘</el-button>
      </template>
    </PageHeader>

    <div class="summary-grid">
      <div class="summary-card">
        <div class="summary-label">键盘模板</div>
        <div class="summary-value">{{ templates.length }}</div>
        <div class="summary-meta">当前可见的自定义按钮键盘</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">全局键盘</div>
        <div class="summary-value">{{ globalCount }}</div>
        <div class="summary-meta">所有管理员共享的通用键盘</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">个人键盘</div>
        <div class="summary-value">{{ personalCount }}</div>
        <div class="summary-meta">仅创建者可见的键盘</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">加载状态</div>
        <div class="summary-value">{{ nextCursor ? '还有更多' : '已到底' }}</div>
        <div class="summary-meta">{{ nextCursor ? '可以继续加载下一页结果' : '当前结果已经完整展示' }}</div>
      </div>
    </div>

    <PanelSection title="键盘列表" description="点击编辑可调整按钮行、按钮文案与跳转地址。">
      <template #actions>
        <div class="scope-toolbar panel-toolbar">
          <div class="scope-main scope-main-single">
            <div class="scope-field">
              <label>作用范围</label>
              <el-select v-model="filterScope" class="wide-control" clearable placeholder="全部范围" @change="loadTemplates">
                <el-option label="全局" value="global" />
                <el-option label="个人" value="personal" />
              </el-select>
            </div>
          </div>
          <div class="result-toolbar">
            <strong>{{ resultHeadline }}</strong>
            <div class="result-toolbar-meta">
              <span>全局 {{ globalCount }} 条</span>
              <span>个人 {{ personalCount }} 条</span>
            </div>
          </div>
        </div>
      </template>

      <el-table class="table-compact" :data="templates" size="small" stripe empty-text="暂无快捷键盘">
        <el-table-column prop="name" label="键盘名称" min-width="160" />
        <el-table-column prop="scope" label="作用范围" min-width="110">
          <template #default="{ row }">{{ scopeLabel(row.scope) }}</template>
        </el-table-column>
        <el-table-column label="按钮预览" min-width="300">
          <template #default="{ row }">
            <div class="keyboard-preview">
              <div v-for="(rowButtons, rowIndex) in parseRows(row.inline_keyboard_json)" :key="rowIndex" class="preview-row">
                <span v-for="(btn, btnIndex) in rowButtons" :key="btnIndex" class="preview-btn">{{ btn.text || '按钮' }}</span>
              </div>
              <span v-if="!parseRows(row.inline_keyboard_json).length" class="preview-empty">未配置按钮</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="is_active" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.is_active ? 'success' : 'info'" size="small">{{ row.is_active ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="removeTemplate(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!templates.length" class="empty-state template-empty">
        <strong>还没有快捷键盘</strong>
        <p>创建一组按钮，例如「官网」「客服」「群规则」，发帖时即可一键复用。</p>
      </div>

      <div v-if="nextCursor" class="load-more">
        <el-button :loading="loadingMore" @click="loadMoreTemplates">加载更多</el-button>
      </div>
    </PanelSection>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑快捷键盘' : '新建快捷键盘'" width="720px">
      <el-form label-position="top" class="template-form">
        <div class="form-section">
          <div class="form-section-title">
            <strong>基本信息</strong>
            <span>用名称和范围区分不同场景的按钮组。</span>
          </div>
          <div class="form-grid">
            <el-form-item label="键盘名称">
              <el-input v-model="form.name" placeholder="例如：官网导航、客服菜单" />
            </el-form-item>
            <el-form-item label="作用范围">
              <el-select v-model="form.scope" class="wide-control">
                <el-option label="全局（所有人可用）" value="global" />
                <el-option label="个人（仅自己可见）" value="personal" />
              </el-select>
            </el-form-item>
            <el-form-item label="启用状态">
              <el-switch v-model="form.is_active" active-text="启用" inactive-text="停用" />
            </el-form-item>
            <div class="form-span-2">
              <el-form-item label="描述">
                <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选，记录这组键盘的用途" />
              </el-form-item>
            </div>
          </div>
        </div>

        <div class="form-section">
          <div class="form-section-title">
            <strong>按钮配置</strong>
            <span>每行一个按钮组，按钮可设置跳转链接（url）或回调数据（callback_data）。</span>
          </div>
          <div class="keyboard-editor">
            <div v-for="(row, rowIndex) in rows" :key="rowIndex" class="kb-row">
              <div class="kb-row-head">
                <span>第 {{ rowIndex + 1 }} 行</span>
                <el-button size="small" text type="danger" @click="removeRow(rowIndex)">删除行</el-button>
              </div>
              <div class="kb-buttons">
                <div v-for="(btn, btnIndex) in row" :key="btnIndex" class="kb-btn">
                  <div class="kb-btn-head">
                    <span>按钮 {{ btnIndex + 1 }}</span>
                    <el-button size="small" text type="danger" @click="removeButton(rowIndex, btnIndex)">移除</el-button>
                  </div>
                  <el-input v-model="btn.text" placeholder="按钮文字（必填）" />
                  <el-input v-model="btn.url" placeholder="跳转链接 https://…" />
                  <el-input v-model="btn.callback_data" placeholder="回调数据 callback_data（可选）" />
                </div>
                <el-button size="small" class="kb-add-btn" @click="addButton(rowIndex)">＋ 添加按钮</el-button>
              </div>
            </div>
            <el-button size="small" @click="addRow">＋ 添加一行</el-button>
            <div v-if="!rows.length" class="empty-state kb-empty">
              <strong>还没有按钮</strong>
              <p>点击上方「添加一行」开始配置按钮。</p>
            </div>
          </div>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitTemplate">保存键盘</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Plus, Refresh } from "@element-plus/icons-vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import {
  createButtonTemplate,
  deleteButtonTemplate,
  fetchButtonTemplates,
  updateButtonTemplate,
} from "@/api/buttonTemplates";
import type {
  ButtonTemplatePayload,
  ButtonTemplateRecord,
  InlineKeyboardButton,
  InlineKeyboardRows,
} from "@/types/api";

const filterScope = ref("");
const loading = ref(false);
const loadingMore = ref(false);
const saving = ref(false);
const dialogVisible = ref(false);
const editing = ref<ButtonTemplateRecord>();
const templates = ref<ButtonTemplateRecord[]>([]);
const nextCursor = ref("");
const form = reactive<ButtonTemplatePayload>({
  name: "",
  scope: "global",
  inline_keyboard_json: "[]",
  description: "",
  is_active: true,
});
const rows = ref<InlineKeyboardRows>([]);

const globalCount = computed(() => templates.value.filter((item) => item.scope === "global").length);
const personalCount = computed(() => templates.value.filter((item) => item.scope !== "global").length);
const resultHeadline = computed(() => {
  if (!templates.value.length) return "当前范围内还没有快捷键盘";
  return `已找到 ${templates.value.length} 条快捷键盘`;
});

function parseRows(raw?: string): InlineKeyboardRows {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed
      .filter((row) => Array.isArray(row))
      .map((row) =>
        row
          .filter((btn) => btn && typeof btn === "object")
          .map((btn) => ({
            text: typeof btn.text === "string" ? btn.text : "",
            url: typeof btn.url === "string" ? btn.url : "",
            callback_data: typeof btn.callback_data === "string" ? btn.callback_data : "",
          })),
      );
  } catch {
    return [];
  }
}

function serializeRows(): string {
  const cleaned = rows.value
    .map((row) =>
      row
        .map((btn) => {
          const out: InlineKeyboardButton = { text: (btn.text || "").trim() };
          if (btn.url && btn.url.trim()) out.url = btn.url.trim();
          if (btn.callback_data && btn.callback_data.trim()) out.callback_data = btn.callback_data.trim();
          return out;
        })
        .filter((btn) => btn.text),
    )
    .filter((row) => row.length > 0);
  return JSON.stringify(cleaned);
}

function addRow(): void {
  rows.value.push([{ text: "", url: "", callback_data: "" }]);
}

function removeRow(index: number): void {
  rows.value.splice(index, 1);
}

function addButton(rowIndex: number): void {
  rows.value[rowIndex].push({ text: "", url: "", callback_data: "" });
}

function removeButton(rowIndex: number, btnIndex: number): void {
  rows.value[rowIndex].splice(btnIndex, 1);
  if (!rows.value[rowIndex].length) rows.value.splice(rowIndex, 1);
}

async function loadTemplates(): Promise<void> {
  loading.value = true;
  nextCursor.value = "";
  try {
    const response = await fetchButtonTemplates(filterScope.value || undefined);
    templates.value = response.items;
    nextCursor.value = response.next_cursor || "";
  } catch {
    templates.value = [];
    nextCursor.value = "";
    ElMessage.error("快捷键盘接口不可用");
  } finally {
    loading.value = false;
  }
}

async function loadMoreTemplates(): Promise<void> {
  if (!nextCursor.value) return;
  loadingMore.value = true;
  try {
    const response = await fetchButtonTemplates(filterScope.value || undefined, undefined, nextCursor.value);
    templates.value = templates.value.concat(response.items);
    nextCursor.value = response.next_cursor || "";
  } catch {
    ElMessage.error("更多快捷键盘加载失败");
  } finally {
    loadingMore.value = false;
  }
}

function openCreate(): void {
  editing.value = undefined;
  Object.assign(form, { name: "", scope: "global", description: "", is_active: true });
  rows.value = [];
  dialogVisible.value = true;
}

function openEdit(row: ButtonTemplateRecord): void {
  editing.value = row;
  Object.assign(form, {
    name: row.name,
    scope: row.scope || "global",
    description: row.description || "",
    is_active: row.is_active,
  });
  rows.value = parseRows(row.inline_keyboard_json);
  dialogVisible.value = true;
}

async function submitTemplate(): Promise<void> {
  if (!form.name.trim()) {
    ElMessage.warning("请填写键盘名称");
    return;
  }
  const keyboardJson = serializeRows();
  if (!rows.value.length || !keyboardJson || keyboardJson === "[]") {
    ElMessage.warning("请至少配置一个按钮");
    return;
  }
  const payload: ButtonTemplatePayload = {
    name: form.name.trim(),
    scope: form.scope || "global",
    inline_keyboard_json: keyboardJson,
    description: form.description || undefined,
    is_active: form.is_active,
  };
  saving.value = true;
  try {
    if (editing.value) {
      await updateButtonTemplate(editing.value.id, payload);
    } else {
      await createButtonTemplate(payload);
    }
    ElMessage.success("快捷键盘已保存");
    dialogVisible.value = false;
    await loadTemplates();
  } catch {
    ElMessage.error("快捷键盘保存失败");
  } finally {
    saving.value = false;
  }
}

async function removeTemplate(row: ButtonTemplateRecord): Promise<void> {
  try {
    await ElMessageBox.confirm(`确认删除快捷键盘「${row.name}」？`, "删除快捷键盘", { type: "warning" });
  } catch {
    return;
  }
  try {
    await deleteButtonTemplate(row.id);
    ElMessage.success("快捷键盘已删除");
    await loadTemplates();
  } catch {
    ElMessage.error("删除失败");
  }
}

function scopeLabel(value?: string): string {
  return value === "personal" ? "个人" : "全局";
}

onMounted(loadTemplates);
</script>

<style scoped>
.panel-toolbar {
  width: min(100%, 920px);
}

.scope-main-single {
  grid-template-columns: minmax(0, 320px);
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

.load-more {
  display: flex;
  justify-content: center;
  margin-top: 18px;
}

.keyboard-preview {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.preview-row {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.preview-btn {
  display: inline-flex;
  align-items: center;
  padding: 3px 10px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  font-size: 12px;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
}

.preview-empty {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.keyboard-editor {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.kb-row {
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.kb-row-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.kb-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: flex-start;
}

.kb-btn {
  flex: 1 1 200px;
  min-width: 200px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: var(--el-fill-color-blank);
}

.kb-btn-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.kb-add-btn {
  align-self: center;
}

.kb-empty {
  padding: 14px 0;
}
</style>
