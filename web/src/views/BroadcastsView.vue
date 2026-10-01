<template>
  <div class="page">
    <PageHeader
      eyebrow="运营"
      title="群发"
      description="一条内容同时群发给 Bot 用户、已托管群组，或两者。支持图文、按钮与定时发送。"
    >
      <template #actions>
        <el-button :icon="Refresh" :loading="loading" @click="reload">刷新</el-button>
        <el-button type="primary" :icon="Promotion" @click="openCreate">新建群发</el-button>
      </template>
    </PageHeader>

    <div class="stat-grid">
      <StatTile label="全部任务" :value="String(stats.total)" tone="primary" description="累计创建的群发数" />
      <StatTile label="待发送" :value="String(stats.queued)" tone="warning" description="已入队，等待 Worker 执行" />
      <StatTile label="发送中" :value="String(stats.sending)" tone="primary" description="正在逐个投递" />
      <StatTile label="已完成" :value="String(stats.completed)" tone="success" description="全部目标已处理" />
      <StatTile label="失败" :value="String(stats.failed)" tone="danger" description="无一成功或任务出错" />
    </div>

    <PanelSection title="群发任务" description="点击任务行可查看每个目标的投递明细。">
      <el-table
        v-loading="loading"
        :data="items"
        stripe
        empty-text="还没有群发任务"
        @row-click="openDetail"
      >
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="标题 / 内容" min-width="220">
          <template #default="{ row }">
            <div class="cell-title">{{ row.title || `群发 #${row.id}` }}</div>
            <div class="cell-sub">{{ truncate(row.content, 60) || "（仅媒体）" }}</div>
          </template>
        </el-table-column>
        <el-table-column label="目标" width="120">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ targetLabel(row.target_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="statusTone(row.status)">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="200">
          <template #default="{ row }">
            <el-progress
              :percentage="progressOf(row)"
              :status="row.status === 'failed' ? 'exception' : undefined"
              :stroke-width="10"
            />
            <div class="cell-sub">
              成功 {{ row.sent_count }} / 失败 {{ row.failed_count }} / 共
              {{ row.total_targets || "—" }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="canSend(row)"
              size="small"
              type="primary"
              link
              @click.stop="onSend(row)"
            >发送</el-button>
            <el-button
              v-if="canCancel(row)"
              size="small"
              type="warning"
              link
              @click.stop="onCancel(row)"
            >取消</el-button>
            <el-button
              v-if="row.status === 'failed' || row.status === 'completed'"
              size="small"
              link
              @click.stop="onSend(row)"
            >重发</el-button>
            <el-button
              v-if="row.status !== 'sending'"
              size="small"
              type="danger"
              link
              @click.stop="onDelete(row)"
            >删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <!-- 新建 / 编辑 -->
    <el-drawer v-model="formVisible" :title="editingId ? `编辑群发 #${editingId}` : '新建群发'" size="620px">
      <el-form label-position="top">
        <el-form-item label="任务名称（仅内部可见）">
          <el-input v-model="form.title" placeholder="例如：618 活动预热" clearable />
        </el-form-item>

        <el-form-item label="发送目标">
          <el-radio-group v-model="form.target_type">
            <el-radio-button value="users">Bot 用户</el-radio-button>
            <el-radio-button value="groups">群组 / 频道</el-radio-button>
            <el-radio-button value="both">两者都要</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="form.target_type !== 'users'">
          <template #label>
            限定群组
            <span class="field-hint">不选 = 所有已托管且启用的群组</span>
          </template>
          <el-select
            v-model="form.chat_ids"
            multiple
            filterable
            clearable
            collapse-tags
            collapse-tags-tooltip
            placeholder="留空表示全选"
            style="width: 100%"
          >
            <el-option
              v-for="chat in chatOptions"
              :key="String(chat.chat_id ?? chat.id)"
              :label="chatLabel(chat)"
              :value="Number(chat.chat_id ?? chat.id)"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="正文内容">
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="5"
            placeholder="支持 HTML / MarkdownV2 标记"
          />
        </el-form-item>

        <el-form-item label="解析模式">
          <el-radio-group v-model="form.parse_mode">
            <el-radio-button value="HTML">HTML</el-radio-button>
            <el-radio-button value="MarkdownV2">MarkdownV2</el-radio-button>
            <el-radio-button value="">纯文本</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="媒体类型">
          <el-radio-group v-model="form.media_type">
            <el-radio-button value="">无</el-radio-button>
            <el-radio-button value="photo">图片</el-radio-button>
            <el-radio-button value="video">视频</el-radio-button>
            <el-radio-button value="document">文件</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="form.media_type">
          <MediaSourceField
            v-model="form.media_url"
            v-model:media-file="form.media_file"
            :media-type="mediaSourceType"
          />
        </el-form-item>

        <el-form-item>
          <template #label>
            图文按钮
            <span class="field-hint">每行最多 8 个按钮</span>
          </template>
          <div class="keyboard-editor">
            <div v-for="(row, rowIndex) in form.keyboard" :key="rowIndex" class="keyboard-row">
              <div v-for="(button, buttonIndex) in row.buttons" :key="buttonIndex" class="keyboard-button">
                <el-input v-model="button.text" placeholder="按钮文字" size="small" />
                <el-input v-model="button.url" placeholder="https://链接" size="small" />
                <el-button text type="danger" :icon="Delete" @click="removeButton(rowIndex, buttonIndex)" />
              </div>
              <div class="keyboard-row-actions">
                <el-button size="small" :icon="Plus" @click="addButton(rowIndex)">加按钮</el-button>
                <el-button size="small" text type="danger" @click="removeRow(rowIndex)">删除本行</el-button>
              </div>
            </div>
            <el-button size="small" :icon="Plus" @click="addRow">新增一行按钮</el-button>
          </div>
        </el-form-item>

        <el-form-item label="发送时间">
          <el-radio-group v-model="scheduleMode">
            <el-radio-button value="now">立即发送</el-radio-button>
            <el-radio-button value="later">定时发送</el-radio-button>
            <el-radio-button value="draft">仅保存草稿</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="scheduleMode === 'later'">
          <el-date-picker
            v-model="form.scheduled_at"
            type="datetime"
            placeholder="选择发送时间"
            style="width: 100%"
          />
        </el-form-item>

        <el-alert v-if="preview" type="info" :closable="false" show-icon style="margin-bottom: 16px">
          <template #title>
            预计触达 {{ preview.total }} 个目标（用户 {{ preview.users }} · 群组 {{ preview.chats }}）
          </template>
        </el-alert>
      </el-form>

      <template #footer>
        <div class="drawer-footer">
          <el-button :loading="previewing" @click="runPreview">预估触达</el-button>
          <el-button @click="formVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="save">保存</el-button>
        </div>
      </template>
    </el-drawer>

    <!-- 投递明细 -->
    <el-drawer v-model="detailVisible" :title="detailTitle" size="620px">
      <el-descriptions :column="1" border style="margin-bottom: 16px">
        <el-descriptions-item label="状态">
          <el-tag size="small" :type="statusTone(detail?.status ?? 'draft')">
            {{ statusLabel(detail?.status ?? "draft") }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="进度">
          成功 {{ detail?.sent_count ?? 0 }} / 失败 {{ detail?.failed_count ?? 0 }} / 共
          {{ detail?.total_targets ?? 0 }}
        </el-descriptions-item>
        <el-descriptions-item v-if="detail?.last_error" label="错误">
          <span class="error-text">{{ detail.last_error }}</span>
        </el-descriptions-item>
      </el-descriptions>

      <el-table v-loading="detailLoading" :data="deliveries" stripe max-height="520">
        <el-table-column label="目标" min-width="180">
          <template #default="{ row }">
            <div class="cell-title">{{ row.target_name || "—" }}</div>
            <div class="cell-sub">{{ row.target_id }}</div>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="90">
          <template #default="{ row }">
            {{ row.target_type === "users" ? "用户" : "群组" }}
          </template>
        </el-table-column>
        <el-table-column label="结果" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'sent' ? 'success' : row.status === 'failed' ? 'danger' : 'info'">
              {{ row.status === "sent" ? "成功" : row.status === "failed" ? "失败" : "待发" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="错误" min-width="200">
          <template #default="{ row }">
            <span class="error-text">{{ truncate(row.error, 80) }}</span>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Delete, Plus, Promotion, Refresh } from "@element-plus/icons-vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import StatTile from "@/components/StatTile.vue";
import MediaSourceField from "@/components/MediaSourceField.vue";
import type { InlineMediaFileValue } from "@/components/MediaSourceField.vue";
import { fetchChats } from "@/api/chats";
import type { ChatRecord } from "@/types/api";
import {
  cancelBroadcast,
  createBroadcast,
  deleteBroadcast,
  fetchBroadcastDeliveries,
  fetchBroadcasts,
  fetchBroadcastStats,
  previewBroadcastTargets,
  sendBroadcast,
} from "@/api/broadcasts";
import type {
  Broadcast,
  BroadcastCreatePayload,
  BroadcastDelivery,
  BroadcastStats,
  BroadcastStatus,
  BroadcastTargetPreview,
  BroadcastTargetType,
  InlineKeyboardRow,
} from "@/api/broadcasts";

const loading = ref(false);
const saving = ref(false);
const previewing = ref(false);
const detailLoading = ref(false);

const items = ref<Broadcast[]>([]);
const chats = ref<ChatRecord[]>([]);
const deliveries = ref<BroadcastDelivery[]>([]);
const detail = ref<Broadcast | null>(null);

const stats = reactive<BroadcastStats>({
  total: 0,
  draft: 0,
  queued: 0,
  sending: 0,
  completed: 0,
  failed: 0,
});

const formVisible = ref(false);
const detailVisible = ref(false);
const editingId = ref<number | null>(null);
const scheduleMode = ref<"now" | "later" | "draft">("now");
const preview = ref<BroadcastTargetPreview | null>(null);

const form = reactive<{
  title: string;
  content: string;
  target_type: BroadcastTargetType;
  chat_ids: number[];
  parse_mode: string;
  media_type: string;
  media_url: string;
  media_file: InlineMediaFileValue | null;
  keyboard: InlineKeyboardRow[];
  scheduled_at: Date | null;
}>({
  title: "",
  content: "",
  target_type: "users",
  chat_ids: [],
  parse_mode: "HTML",
  media_type: "",
  media_url: "",
  media_file: null,
  keyboard: [],
  scheduled_at: null,
});

const chatOptions = computed(() => chats.value);

const mediaSourceType = computed<"text" | "photo" | "video" | "document">(() => {
  const value = form.media_type;
  if (value === "photo" || value === "video" || value === "document") return value;
  return "text";
});

const detailTitle = computed(() =>
  detail.value ? `群发 #${detail.value.id} 投递明细` : "投递明细",
);

let poller: number | undefined;

function chatLabel(chat: ChatRecord): string {
  const id = String(chat.chat_id ?? chat.id ?? "");
  const title = chat.title || chat.username || id;
  return `${title} · ${id}`;
}

function targetLabel(value: string): string {
  return { users: "用户", groups: "群组", both: "两者" }[value] ?? value;
}

function statusLabel(value: string): string {
  return (
    {
      draft: "草稿",
      queued: "待发送",
      sending: "发送中",
      completed: "已完成",
      failed: "失败",
      canceled: "已取消",
    }[value] ?? value
  );
}

function statusTone(value: string): "success" | "warning" | "danger" | "info" | "primary" {
  if (value === "completed") return "success";
  if (value === "failed") return "danger";
  if (value === "sending") return "primary";
  if (value === "queued") return "warning";
  return "info";
}

function truncate(value: string, max: number): string {
  if (!value) return "";
  return value.length > max ? `${value.slice(0, max)}…` : value;
}

function formatTime(value?: string | null): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function progressOf(row: Broadcast): number {
  if (!row.total_targets) {
    return row.status === "completed" ? 100 : 0;
  }
  const done = row.sent_count + row.failed_count;
  return Math.min(100, Math.round((done / row.total_targets) * 100));
}

function canSend(row: Broadcast): boolean {
  return row.status === "draft" || row.status === "queued" || row.status === "canceled";
}

function canCancel(row: Broadcast): boolean {
  return row.status === "draft" || row.status === "queued";
}

async function reload(): Promise<void> {
  loading.value = true;
  try {
    const [list, statPayload] = await Promise.all([
      fetchBroadcasts(),
      fetchBroadcastStats(),
    ]);
    items.value = list;
    Object.assign(stats, statPayload);
  } catch (error) {
    ElMessage.error(errorMessage(error, "加载群发列表失败"));
  } finally {
    loading.value = false;
  }
}

async function loadChats(): Promise<void> {
  try {
    chats.value = await fetchChats();
  } catch {
    chats.value = [];
  }
}

function resetForm(): void {
  form.title = "";
  form.content = "";
  form.target_type = "users";
  form.chat_ids = [];
  form.parse_mode = "HTML";
  form.media_type = "";
  form.media_url = "";
  form.media_file = null;
  form.keyboard = [];
  form.scheduled_at = null;
  scheduleMode.value = "now";
  preview.value = null;
}

function openCreate(): void {
  editingId.value = null;
  resetForm();
  formVisible.value = true;
}

function buildPayload(): BroadcastCreatePayload {
  const payload: BroadcastCreatePayload = {
    title: form.title.trim(),
    content: form.content,
    target_type: form.target_type,
    chat_ids: form.target_type === "users" ? [] : form.chat_ids,
    parse_mode: form.parse_mode,
    media_type: form.media_type,
    media_url: form.media_url,
  };

  if (form.media_file) {
    payload.media_name = form.media_file.name;
    payload.media_mime = form.media_file.mime_type;
    payload.media_data_base64 = form.media_file.data_base64;
  }

  const keyboard = form.keyboard
    .map((row) => ({
      buttons: row.buttons
        .filter((button) => button.text.trim())
        .map((button) => ({
          text: button.text.trim(),
          url: button.url?.trim() || undefined,
        })),
    }))
    .filter((row) => row.buttons.length > 0);

  payload.inline_keyboard = keyboard;

  if (scheduleMode.value === "now") {
    payload.send_now = true;
  } else if (scheduleMode.value === "later" && form.scheduled_at) {
    payload.scheduled_at = form.scheduled_at.toISOString();
  }

  return payload;
}

async function runPreview(): Promise<void> {
  previewing.value = true;
  try {
    preview.value = await previewBroadcastTargets(buildPayload());
  } catch (error) {
    ElMessage.error(errorMessage(error, "预估失败"));
  } finally {
    previewing.value = false;
  }
}

async function save(): Promise<void> {
  saving.value = true;
  try {
    await createBroadcast(buildPayload());
    ElMessage.success(scheduleMode.value === "now" ? "已加入发送队列" : "群发任务已保存");
    formVisible.value = false;
    await reload();
  } catch (error) {
    ElMessage.error(errorMessage(error, "保存失败"));
  } finally {
    saving.value = false;
  }
}

async function onSend(row: Broadcast): Promise<void> {
  try {
    await sendBroadcast(row.id);
    ElMessage.success(`群发 #${row.id} 已加入发送队列`);
    await reload();
  } catch (error) {
    ElMessage.error(errorMessage(error, "发送失败"));
  }
}

async function onCancel(row: Broadcast): Promise<void> {
  try {
    await cancelBroadcast(row.id);
    ElMessage.success(`群发 #${row.id} 已取消`);
    await reload();
  } catch (error) {
    ElMessage.error(errorMessage(error, "取消失败"));
  }
}

async function onDelete(row: Broadcast): Promise<void> {
  try {
    await ElMessageBox.confirm(`确定删除群发 #${row.id}？`, "删除确认", { type: "warning" });
  } catch {
    return;
  }
  try {
    await deleteBroadcast(row.id);
    ElMessage.success("已删除");
    await reload();
  } catch (error) {
    ElMessage.error(errorMessage(error, "删除失败"));
  }
}

async function openDetail(row: Broadcast): Promise<void> {
  detail.value = row;
  detailVisible.value = true;
  detailLoading.value = true;
  try {
    deliveries.value = await fetchBroadcastDeliveries(row.id);
  } catch (error) {
    ElMessage.error(errorMessage(error, "加载投递明细失败"));
    deliveries.value = [];
  } finally {
    detailLoading.value = false;
  }
}

function addRow(): void {
  form.keyboard.push({ buttons: [{ text: "", url: "" }] });
}

function addButton(rowIndex: number): void {
  if (form.keyboard[rowIndex].buttons.length >= 8) {
    ElMessage.warning("每行最多 8 个按钮");
    return;
  }
  form.keyboard[rowIndex].buttons.push({ text: "", url: "" });
}

function removeButton(rowIndex: number, buttonIndex: number): void {
  form.keyboard[rowIndex].buttons.splice(buttonIndex, 1);
  if (form.keyboard[rowIndex].buttons.length === 0) {
    form.keyboard.splice(rowIndex, 1);
  }
}

function removeRow(rowIndex: number): void {
  form.keyboard.splice(rowIndex, 1);
}

function errorMessage(error: unknown, fallback: string): string {
  const payload = (error as { payload?: unknown })?.payload;
  if (typeof payload === "string" && payload.trim()) return payload;
  if (payload && typeof payload === "object") {
    const record = payload as Record<string, unknown>;
    const message = record.error ?? record.message;
    if (typeof message === "string" && message.trim()) return message;
  }
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

onMounted(async () => {
  await Promise.all([reload(), loadChats()]);
  poller = window.setInterval(() => {
    if (items.value.some((item) => item.status === "sending" || item.status === "queued")) {
      void reload();
    }
  }, 5000);
});

onUnmounted(() => {
  if (poller) window.clearInterval(poller);
});
</script>

<style scoped>
.stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.cell-title {
  font-weight: 600;
}

.cell-sub {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.field-hint {
  margin-left: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.keyboard-editor {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
}

.keyboard-row {
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.keyboard-button {
  display: grid;
  grid-template-columns: 1fr 1.4fr auto;
  gap: 8px;
  align-items: center;
}

.keyboard-row-actions {
  display: flex;
  gap: 8px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.error-text {
  color: var(--el-color-danger);
  font-size: 12px;
  word-break: break-all;
}
</style>
