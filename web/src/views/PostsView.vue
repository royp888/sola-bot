<template>
  <div class="page-stack">
    <PageHeader eyebrow="Publishing" title="发布任务" description="维护一次性和 cron 定时发布任务。">
      <template #actions>
        <el-button :icon="Refresh" :loading="loading" @click="loadPosts">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">新建任务</el-button>
      </template>
    </PageHeader>

    <div class="summary-grid">
      <div class="summary-card">
        <div class="summary-label">任务总数</div>
        <div class="summary-value">{{ posts.length }}</div>
        <div class="summary-meta">当前列表已加载的发布任务</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">启用中</div>
        <div class="summary-value">{{ enabledCount }}</div>
        <div class="summary-meta">仍在调度中的任务</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">一次性任务</div>
        <div class="summary-value">{{ onceCount }}</div>
        <div class="summary-meta">按指定时间触发后停用</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">循环任务</div>
        <div class="summary-value">{{ recurringCount }}</div>
        <div class="summary-meta">Cron 或间隔调度任务</div>
      </div>
    </div>

    <PanelSection title="任务列表" description="对应 /api/posts、/api/posts/:id/toggle。">
      <template #actions>
        <div class="panel-toolbar">
          <div class="control-cluster filters">
            <ChatSelect v-model="selectedChatId" class="filter-control" />
            <el-select v-model="statusFilter" class="filter-control">
              <el-option label="全部状态" value="all" />
              <el-option label="启用中" value="enabled" />
              <el-option label="已停用" value="disabled" />
            </el-select>
            <el-select v-model="scheduleFilter" class="filter-control">
              <el-option label="全部类型" value="all" />
              <el-option label="一次性" value="once" />
              <el-option label="循环任务" value="recurring" />
            </el-select>
          </div>
          <div class="filter-summary">
            <span>当前显示 {{ filteredPosts.length }} / {{ posts.length }} 条任务</span>
          </div>
        </div>
      </template>

      <el-table :data="filteredPosts" stripe class="table-compact">
        <el-table-column prop="title" label="标题" min-width="160" />
        <el-table-column prop="chat_id" label="Chat" min-width="120" />
        <el-table-column prop="media_type" label="类型" width="110" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" effect="dark">{{ row.enabled ? "启用" : "停用" }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="计划" min-width="160">
          <template #default="{ row }">{{ scheduleLabel(row) }}</template>
        </el-table-column>
        <el-table-column label="发送后" min-width="150">
          <template #default="{ row }">
            <el-tag v-if="row.pin_after_send" size="small" effect="dark">置顶</el-tag>
            <el-tag v-if="row.auto_delete_seconds" size="small" type="warning" effect="dark">
              {{ row.auto_delete_seconds }}s 删除
            </el-tag>
            <span v-if="!row.pin_after_send && !row.auto_delete_seconds">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="last_run_at" label="上次执行" min-width="150" />
        <el-table-column label="启用" width="110">
          <template #default="{ row }">
            <el-switch :model-value="row.enabled" @change="togglePost(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" :loading="deletingId === row.id" @click="removePost(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <el-dialog v-model="dialogVisible" :title="editingPost ? '编辑提醒/发布任务' : '新建提醒/发布任务'" width="620px">
      <el-form label-position="top">
        <el-form-item label="Chat ID">
          <ChatSelect v-model="form.chat_id" @update:model-value="loadTemplatesForPost" />
        </el-form-item>
        <el-form-item label="从模板载入">
          <el-select v-model="selectedTemplateId" class="wide-control" clearable filterable @change="applyTemplate">
            <el-option
              v-for="template in templates"
              :key="template.id"
              :label="`${template.name} · ${template.chat_id || '全局'}`"
              :value="String(template.id)"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="标题">
          <el-input v-model="form.title" />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="form.content" type="textarea" :rows="4" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="媒体类型">
              <el-select v-model="form.media_type" class="wide-control">
                <el-option label="文字" value="text" />
                <el-option label="图片" value="photo" />
                <el-option label="视频" value="video" />
                <el-option label="文件" value="document" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="调度类型">
              <el-select v-model="scheduleMode" class="wide-control" @change="handleScheduleModeChange">
                <el-option
                  v-for="option in scheduleModeOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item v-if="form.media_type !== 'text'" label="媒体 URL">
          <el-input v-model="form.media_url" placeholder="https://example.com/file.jpg" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col v-if="scheduleMode === 'once'" :xs="24" :md="12">
            <el-form-item label="提醒时间">
              <el-date-picker
                v-model="form.run_once_at"
                class="wide-control"
                type="datetime"
                format="YYYY-MM-DD HH:mm"
                value-format="YYYY-MM-DD HH:mm:ss"
                placeholder="选择日期和时间"
              />
            </el-form-item>
          </el-col>
          <el-col v-if="['daily', 'weekly', 'monthly'].includes(scheduleMode)" :xs="24" :md="12">
            <el-form-item label="执行时间">
              <el-time-picker
                v-model="timeOfDay"
                class="wide-control"
                format="HH:mm"
                value-format="HH:mm"
                placeholder="选择时间"
              />
            </el-form-item>
          </el-col>
          <el-col v-if="scheduleMode === 'hourly'" :xs="24" :md="12">
            <el-form-item label="每小时第几分钟">
              <el-input-number v-model="minuteOfHour" class="wide-control" :min="0" :max="59" />
            </el-form-item>
          </el-col>
          <el-col v-if="scheduleMode === 'weekly'" :xs="24" :md="12">
            <el-form-item label="星期">
              <el-select v-model="weekday" class="wide-control">
                <el-option label="周一" :value="1" />
                <el-option label="周二" :value="2" />
                <el-option label="周三" :value="3" />
                <el-option label="周四" :value="4" />
                <el-option label="周五" :value="5" />
                <el-option label="周六" :value="6" />
                <el-option label="周日" :value="0" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col v-if="scheduleMode === 'monthly'" :xs="24" :md="12">
            <el-form-item label="每月日期">
              <el-input-number v-model="monthDay" class="wide-control" :min="1" :max="31" />
            </el-form-item>
          </el-col>
          <el-col v-if="scheduleMode === 'seconds'" :xs="24" :md="12">
            <el-form-item label="秒数间隔">
              <el-input-number v-model="intervalSeconds" class="wide-control" :min="1" :max="86400" />
            </el-form-item>
          </el-col>
          <el-col v-if="scheduleMode === 'minutes'" :xs="24" :md="12">
            <el-form-item label="分钟间隔">
              <el-input-number v-model="intervalMinutes" class="wide-control" :min="1" :max="1440" />
            </el-form-item>
          </el-col>
          <el-col v-if="scheduleMode === 'custom'" :xs="24" :md="12">
            <el-form-item label="Cron 表达式">
              <el-input v-model="form.cron_expr" placeholder="30 20 * * *" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="发送后置顶">
              <el-switch v-model="form.pin_after_send" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="自动删除秒数">
              <el-input-number v-model="form.auto_delete_seconds" class="wide-control" :min="0" :max="604800" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-alert class="preview" :type="scheduleCheck.type" :closable="false" show-icon :title="scheduleCheck.title" />
        <el-alert class="preview" type="info" :closable="false" show-icon :title="schedulePreview" />
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitPost">{{ editingPost ? "保存" : "创建" }}</el-button>
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
import {
  createScheduledPost,
  deleteScheduledPost,
  fetchScheduledPosts,
  toggleScheduledPost,
  updateScheduledPost,
} from "@/api/posts";
import { fetchTemplates } from "@/api/templates";
import type { ChatID, MessageTemplateRecord, ScheduledPostPayload, ScheduledPostRecord } from "@/types/api";

const loading = ref(false);
const saving = ref(false);
const dialogVisible = ref(false);
const posts = ref<ScheduledPostRecord[]>([]);
const selectedChatId = ref<ChatID | "">("");
const statusFilter = ref<"all" | "enabled" | "disabled">("all");
const scheduleFilter = ref<"all" | "once" | "recurring">("all");
const templates = ref<MessageTemplateRecord[]>([]);
const deletingId = ref<ChatID>();
const editingPost = ref<ScheduledPostRecord>();
const selectedTemplateId = ref("");
type ScheduleMode = "once" | "daily" | "hourly" | "weekly" | "monthly" | "seconds" | "minutes" | "custom";
type ScheduleCheck = { valid: boolean; type: "success" | "warning" | "info" | "error"; title: string };

const scheduleMode = ref<ScheduleMode>("once");
const scheduleModeOptions: Array<{ label: string; value: ScheduleMode }> = [
  { label: "一次性提醒", value: "once" },
  { label: "每天固定时间", value: "daily" },
  { label: "每小时", value: "hourly" },
  { label: "每周", value: "weekly" },
  { label: "每月", value: "monthly" },
  { label: "每 N 秒", value: "seconds" },
  { label: "每 N 分钟", value: "minutes" },
  { label: "Cron 高级模式", value: "custom" },
];
const timeOfDay = ref("20:30");
const minuteOfHour = ref(0);
const weekday = ref(1);
const monthDay = ref(1);
const intervalSeconds = ref(30);
const intervalMinutes = ref(5);
const form = reactive<ScheduledPostPayload>({
  chat_id: "",
  title: "",
  content: "",
  media_type: "text",
  cron_expr: "",
  run_once_at: "",
  enabled: true,
  pin_after_send: false,
  auto_delete_seconds: 0,
});

const schedulePreview = computed(() => {
  const fields = scheduleFields(false);
  if (!fields) return "请选择执行计划";
  const raw = fields.run_once_at || fields.cron_expr || "-";
  return `计划：${scheduleLabel({ ...form, id: "", created_at: "", ...fields } as ScheduledPostRecord)}，实际提交：${raw}`;
});

const scheduleCheck = computed<ScheduleCheck>(() => validateSchedule(false));
const filteredPosts = computed(() => {
  return posts.value.filter((item) => {
    if (selectedChatId.value && String(item.chat_id) !== String(selectedChatId.value)) {
      return false;
    }
    if (statusFilter.value === "enabled" && !item.enabled) {
      return false;
    }
    if (statusFilter.value === "disabled" && item.enabled) {
      return false;
    }
    if (scheduleFilter.value === "once" && !item.run_once_at) {
      return false;
    }
    if (scheduleFilter.value === "recurring" && item.run_once_at) {
      return false;
    }
    return true;
  });
});
const enabledCount = computed(() => posts.value.filter((item) => item.enabled).length);
const onceCount = computed(() => posts.value.filter((item) => Boolean(item.run_once_at)).length);
const recurringCount = computed(() => posts.value.filter((item) => !item.run_once_at).length);

function parseNumericId(value: ChatID): number | undefined {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function optionalText(value?: string | null): string | undefined {
  const text = value?.trim();
  return text || undefined;
}

function toRFC3339(value?: string | null): string | undefined {
  const text = optionalText(value);
  if (!text) return undefined;
  const normalized = text.includes("T") ? text : text.replace(" ", "T");
  const date = new Date(normalized);
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}

function cronFromTime(parts: { day?: number; weekday?: number } = {}): string {
  const [hour = "20", minute = "30"] = timeOfDay.value.split(":");
  const day = parts.day ?? "*";
  const weekdayPart = parts.weekday ?? "*";
  return `${Number(minute)} ${Number(hour)} ${day} * ${weekdayPart}`;
}

function scheduleFields(showWarning = true): Pick<ScheduledPostPayload, "cron_expr" | "run_once_at"> | undefined {
  switch (scheduleMode.value) {
    case "once": {
      const runAt = toRFC3339(form.run_once_at);
      if (!runAt) {
        if (showWarning) {
          ElMessage.warning("请选择提醒时间");
        }
        return undefined;
      }
      return { run_once_at: runAt, cron_expr: undefined };
    }
    case "daily":
      return { cron_expr: cronFromTime(), run_once_at: undefined };
    case "hourly":
      return { cron_expr: `${minuteOfHour.value} * * * *`, run_once_at: undefined };
    case "weekly":
      return { cron_expr: cronFromTime({ weekday: weekday.value }), run_once_at: undefined };
    case "monthly":
      return { cron_expr: cronFromTime({ day: monthDay.value }), run_once_at: undefined };
    case "seconds":
      return { cron_expr: `@every ${intervalSeconds.value}s`, run_once_at: undefined };
    case "minutes":
      return { cron_expr: `@every ${intervalMinutes.value}m`, run_once_at: undefined };
    case "custom":
      if (!optionalText(form.cron_expr)) {
        if (showWarning) {
          ElMessage.warning("请输入 Cron 表达式");
        }
        return undefined;
      }
      return { cron_expr: optionalText(form.cron_expr), run_once_at: undefined };
    default:
      return undefined;
  }
}

function validateSchedule(showWarning = true): ScheduleCheck {
  const fields = scheduleFields(false);
  if (!fields) {
    const title = scheduleMode.value === "custom" ? "请输入 Cron 表达式" : "请选择完整的执行时间";
    if (showWarning) {
      ElMessage.warning(title);
    }
    return { valid: false, type: "error", title };
  }

  if (fields.run_once_at) {
    const runAt = new Date(fields.run_once_at);
    if (Number.isNaN(runAt.getTime()) || !runAt.getTime()) {
      const title = "提醒时间格式无效";
      if (showWarning) ElMessage.warning(title);
      return { valid: false, type: "error", title };
    }
    if (!runAt.getTime() || runAt <= new Date()) {
      const title = "提醒时间需要晚于当前时间";
      if (showWarning) ElMessage.warning(title);
      return { valid: false, type: "error", title };
    }
    return { valid: true, type: "success", title: "一次性任务会在所选时间触发，执行后自动停用" };
  }

  const cronExpr = optionalText(fields.cron_expr);
  if (!cronExpr) {
    const title = "请选择完整的执行计划";
    if (showWarning) ElMessage.warning(title);
    return { valid: false, type: "error", title };
  }

  const cronError = cronValidationMessage(cronExpr);
  if (cronError) {
    if (showWarning) ElMessage.warning(cronError);
    return { valid: false, type: "error", title: cronError };
  }

  if (scheduleMode.value === "seconds" && intervalSeconds.value < 60) {
    return {
      valid: true,
      type: "warning",
      title: "秒级任务可提交；worker 每分钟兜底扫描，低于 60 秒的执行精度取决于调度器运行状态",
    };
  }

  return { valid: true, type: "success", title: "调度表达式有效，保存后将进入后台调度队列" };
}

function cronValidationMessage(expr: string): string | undefined {
  if (expr.startsWith("@every ")) {
    const duration = expr.replace("@every ", "").trim();
    if (!/^\d+[smh]$/.test(duration)) {
      return "间隔表达式仅支持 @every 30s、@every 5m、@every 1h 这类格式";
    }
    return undefined;
  }

  const parts = expr.trim().split(/\s+/);
  if (parts.length !== 5) {
    return "Cron 表达式需要 5 段，例如 30 20 * * *";
  }

  const ranges = [
    { min: 0, max: 59, name: "分钟" },
    { min: 0, max: 23, name: "小时" },
    { min: 1, max: 31, name: "日期" },
    { min: 1, max: 12, name: "月份" },
    { min: 0, max: 7, name: "星期" },
  ];
  for (let index = 0; index < parts.length; index += 1) {
    const error = cronPartError(parts[index], ranges[index]);
    if (error) return error;
  }
  return undefined;
}

function cronPartError(part: string, range: { min: number; max: number; name: string }): string | undefined {
  if (part === "*") return undefined;
  const values = part.split(",");
  for (const value of values) {
    const [start, end] = value.split("-");
    const parsedStart = Number(start);
    const parsedEnd = end == null ? parsedStart : Number(end);
    if (!Number.isInteger(parsedStart) || !Number.isInteger(parsedEnd)) {
      return `${range.name}字段仅支持 *、数字、范围或逗号分隔`;
    }
    if (parsedStart < range.min || parsedEnd > range.max || parsedStart > parsedEnd) {
      return `${range.name}字段超出范围`;
    }
  }
  return undefined;
}

function buildPayload(): ScheduledPostPayload | undefined {
  const chatId = parseNumericId(form.chat_id);
  if (!chatId) {
    ElMessage.warning("请输入有效的 Chat ID");
    return undefined;
  }
  const check = validateSchedule();
  if (!check.valid) return undefined;
  const schedule = scheduleFields();
  if (!schedule) return undefined;
  if (form.media_type !== "text" && !optionalText(form.media_url)) {
    ElMessage.warning("媒体任务需要填写媒体 URL");
    return undefined;
  }
  if (form.media_type === "text" && !form.content.trim() && !form.title.trim()) {
    ElMessage.warning("文字任务需要填写标题或内容");
    return undefined;
  }
  return {
    chat_id: chatId,
    title: form.title.trim(),
    content: form.content.trim(),
    media_type: form.media_type,
    media_url: optionalText(form.media_url),
    cron_expr: schedule.cron_expr,
    run_once_at: schedule.run_once_at,
    enabled: form.enabled,
    pin_after_send: Boolean(form.pin_after_send),
    auto_delete_seconds: Number(form.auto_delete_seconds || 0),
  };
}

function openCreate(): void {
  editingPost.value = undefined;
  Object.assign(form, {
    chat_id: "",
    title: "",
    content: "",
    media_type: "text",
    media_url: "",
    cron_expr: "",
    run_once_at: "",
    enabled: true,
    pin_after_send: false,
    auto_delete_seconds: 0,
  });
  selectedTemplateId.value = "";
  scheduleMode.value = "once";
  void loadTemplatesForPost();
  dialogVisible.value = true;
}

function handleScheduleModeChange(): void {
  if (scheduleMode.value !== "custom") {
    form.cron_expr = "";
  }
  if (scheduleMode.value !== "once") {
    form.run_once_at = "";
  }
}

function openEdit(row: ScheduledPostRecord): void {
  editingPost.value = row;
  Object.assign(form, {
    chat_id: row.chat_id,
    title: row.title || "",
    content: row.content || "",
    media_type: row.media_type || "text",
    media_url: row.media_url || "",
    cron_expr: row.cron_expr || "",
    run_once_at: row.run_once_at || "",
    enabled: row.enabled,
    pin_after_send: Boolean(row.pin_after_send),
    auto_delete_seconds: row.auto_delete_seconds || 0,
  });
  selectedTemplateId.value = "";
  applyScheduleFromPost(row);
  void loadTemplatesForPost();
  dialogVisible.value = true;
}

async function loadTemplatesForPost(): Promise<void> {
  try {
    templates.value = (await fetchTemplates(form.chat_id || undefined)).items;
  } catch {
    templates.value = [];
  }
}

function applyTemplate(): void {
  const template = templates.value.find((item) => String(item.id) === String(selectedTemplateId.value));
  if (!template) return;
  form.content = template.content || "";
  form.media_type = template.media_type || "text";
  form.media_url = template.media_url || "";
  if (!form.title) {
    form.title = template.name;
  }
}

async function loadPosts(): Promise<void> {
  loading.value = true;
  try {
    posts.value = await fetchScheduledPosts();
  } catch (error) {
    posts.value = [];
    ElMessage.error(errorMessage(error));
  } finally {
    loading.value = false;
  }
}

async function togglePost(row: ScheduledPostRecord): Promise<void> {
  try {
    const updated = await toggleScheduledPost(row.id, !row.enabled);
    Object.assign(row, updated);
  } catch (error) {
    ElMessage.error(errorMessage(error));
  }
}

async function submitPost(): Promise<void> {
  const payload = buildPayload();
  if (!payload) return;
  saving.value = true;
  try {
    if (editingPost.value) {
      await updateScheduledPost(editingPost.value.id, payload);
      ElMessage.success("任务已保存");
    } else {
      await createScheduledPost(payload);
      ElMessage.success("任务已创建");
    }
    dialogVisible.value = false;
    await loadPosts();
  } catch (error) {
    ElMessage.error(errorMessage(error));
  } finally {
    saving.value = false;
  }
}

async function removePost(row: ScheduledPostRecord): Promise<void> {
  await ElMessageBox.confirm(`确认删除任务「${row.title || row.id}」？`, "删除定时任务", {
    type: "warning",
    confirmButtonText: "删除",
    cancelButtonText: "取消",
  });
  deletingId.value = row.id;
  try {
    await deleteScheduledPost(row.id);
    ElMessage.success("任务已删除");
    await loadPosts();
  } catch (error) {
    ElMessage.error(errorMessage(error));
  } finally {
    deletingId.value = undefined;
  }
}

function errorMessage(error: unknown): string {
  const payload = (error as { payload?: { error?: string } })?.payload;
  const status = (error as { status?: number })?.status;
  return payload?.error || (status ? `接口返回 ${status}` : "接口不可用");
}

function scheduleLabel(row: Pick<ScheduledPostRecord, "cron_expr" | "run_once_at">): string {
  if (row.run_once_at) {
    return `一次性 ${formatDateTime(row.run_once_at)}`;
  }
  const expr = row.cron_expr?.trim();
  if (!expr) return "-";
  if (expr.startsWith("@every ")) {
    return `每 ${expr.replace("@every ", "")}`;
  }
  const parts = expr.split(/\s+/);
  if (parts.length === 5) {
    const [minute, hour, day, month, weekdayPart] = parts;
    if (day === "*" && month === "*" && weekdayPart === "*") {
      if (hour === "*") return `每小时第 ${minute} 分钟`;
      return `每天 ${padTime(hour)}:${padTime(minute)}`;
    }
    if (day === "*" && month === "*" && weekdayPart !== "*") {
      return `每周${weekdayLabel(weekdayPart)} ${padTime(hour)}:${padTime(minute)}`;
    }
    if (day !== "*" && month === "*" && weekdayPart === "*") {
      return `每月 ${day} 日 ${padTime(hour)}:${padTime(minute)}`;
    }
  }
  return expr;
}

function applyScheduleFromPost(row: Pick<ScheduledPostRecord, "cron_expr" | "run_once_at">): void {
  if (row.run_once_at) {
    scheduleMode.value = "once";
    form.run_once_at = formatPickerValue(row.run_once_at);
    return;
  }
  const expr = row.cron_expr?.trim();
  if (!expr) {
    scheduleMode.value = "once";
    return;
  }
  if (expr.startsWith("@every ")) {
    const value = expr.replace("@every ", "");
    const parsed = Number(value.slice(0, -1));
    if (value.endsWith("s") && Number.isFinite(parsed)) {
      scheduleMode.value = "seconds";
      intervalSeconds.value = parsed;
      return;
    }
    if (value.endsWith("m") && Number.isFinite(parsed)) {
      scheduleMode.value = "minutes";
      intervalMinutes.value = parsed;
      return;
    }
  }
  const parts = expr.split(/\s+/);
  if (parts.length === 5) {
    const [minute, hour, day, month, weekdayPart] = parts;
    timeOfDay.value = `${padTime(hour)}:${padTime(minute)}`;
    if (day === "*" && month === "*" && weekdayPart === "*") {
      if (hour === "*") {
        scheduleMode.value = "hourly";
        minuteOfHour.value = Number(minute) || 0;
      } else {
        scheduleMode.value = "daily";
      }
      return;
    }
    if (day === "*" && month === "*" && weekdayPart !== "*") {
      scheduleMode.value = "weekly";
      weekday.value = Number(weekdayPart) || 0;
      return;
    }
    if (day !== "*" && month === "*" && weekdayPart === "*") {
      scheduleMode.value = "monthly";
      monthDay.value = Number(day) || 1;
      return;
    }
  }
  scheduleMode.value = "custom";
  form.cron_expr = expr;
}

function formatPickerValue(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  const pad = (item: number) => String(item).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:00`;
}

function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function padTime(value: string): string {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? String(parsed).padStart(2, "0") : value;
}

function weekdayLabel(value: string): string {
  return { "0": "日", "1": "一", "2": "二", "3": "三", "4": "四", "5": "五", "6": "六" }[value] ?? value;
}

onMounted(loadPosts);
</script>

<style scoped>
.panel-toolbar {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: min(100%, 960px);
}

.wide-control {
  width: 100%;
}

.preview {
  margin-top: 4px;
}

.filters :deep(.chat-select) {
  width: 100%;
}
</style>

