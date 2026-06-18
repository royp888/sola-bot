<template>
  <div class="page-stack">
    <PageHeader eyebrow="内容发布" title="发布任务" description="支持仅文字、图片 + 文字、视频 + 文字的北京时间定时发送。">
      <template #meta>
        <span class="page-meta-chip">页面内时间均为北京时间（东八区）</span>
      </template>
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
        <div class="summary-meta">Cron 表达式或按间隔执行的任务</div>
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

      <div class="table-wrap">
        <el-table :data="filteredPosts" stripe class="table-compact" empty-text="暂无符合条件的发布任务">
        <el-table-column prop="title" label="标题" min-width="160" />
        <el-table-column prop="chat_id" label="目标群组" min-width="140" />
        <el-table-column label="发送形式" min-width="130">
          <template #default="{ row }">{{ mediaTypeLabel(row.media_type) }}</template>
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
        <el-table-column label="最近发送" min-width="180">
          <template #default="{ row }">{{ row.last_run_at ? formatDateTime(row.last_run_at) : "尚未发送" }}</template>
        </el-table-column>
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
      </div>
    </PanelSection>

    <el-dialog v-model="dialogVisible" :title="editingPost ? '编辑发布任务' : '新建发布任务'" width="680px" class="posts-dialog">
      <el-form label-position="top" class="post-form">
        <el-alert class="preview" type="info" :closable="false" show-icon title="图片和视频任务支持附带文字说明；页面内时间均按北京时间（东八区）处理。" />
        <el-form-item label="目标群组">
          <ChatSelect v-model="form.chat_id" @update:model-value="loadTemplatesForPost" />
        </el-form-item>
        <el-form-item label="从模板快速填充">
          <el-select v-model="selectedTemplateId" class="wide-control" clearable filterable @change="applyTemplate">
            <el-option
              v-for="template in templates"
              :key="template.id"
              :label="templateLabel(template)"
              :value="String(template.id)"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="任务名称">
          <el-input v-model="form.title" />
        </el-form-item>
        <el-form-item :label="bodyLabel">
          <el-input v-model="form.content" type="textarea" :rows="5" :placeholder="bodyPlaceholder" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="媒体类型">
              <el-select v-model="form.media_type" class="wide-control" @change="handleMediaTypeChange">
                <el-option label="仅文字" value="text" />
                <el-option label="图片 + 文字" value="photo" />
                <el-option label="视频 + 文字" value="video" />
                <el-option label="文件 + 文字" value="document" />
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
        <el-form-item v-if="form.media_type !== 'text'" label="媒体文件">
          <MediaSourceField
            :model-value="form.media_url || ''"
            @update:model-value="form.media_url = $event"
            :media-type="form.media_type"
            :media-file="mediaFile"
            :existing-file-name="existingInlineMediaName"
            :existing-mime="existingInlineMediaMime"
            placeholder="可粘贴图片/视频链接，或直接上传文件"
            @update:media-file="handleMediaFileUpdate"
          />
        </el-form-item>
        <el-row :gutter="12">
          <el-col v-if="scheduleMode === 'once'" :xs="24" :md="12">
            <el-form-item label="发送时间（北京时间）">
              <el-date-picker
                v-model="form.run_once_at"
                class="wide-control"
                type="datetime"
                format="YYYY-MM-DD HH:mm"
                value-format="YYYY-MM-DD HH:mm:ss"
                placeholder="选择发送时间（北京时间）"
              />
            </el-form-item>
          </el-col>
          <el-col v-if="['daily', 'weekly', 'monthly'].includes(scheduleMode)" :xs="24" :md="12">
            <el-form-item label="发送时刻（北京时间）">
              <el-time-picker
                v-model="timeOfDay"
                class="wide-control"
                format="HH:mm"
                value-format="HH:mm"
                placeholder="选择时刻（北京时间）"
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
            <el-form-item label="自定义发送规则（Cron 表达式）">
              <el-input v-model="form.cron_expr" placeholder="30 20 * * *" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="保存后立即启用">
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
        <el-button type="primary" :loading="saving" @click="submitPost">{{ editingPost ? "保存修改" : "创建任务" }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import MediaSourceField, { type InlineMediaFileValue } from "@/components/MediaSourceField.vue";
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
import { formatChinaInputDateTime, parseChinaLocalDateTimeToISO } from "@/utils/datetime";
import { parseNumericId, formatDateTime, errorMessage } from "@/utils/helpers";

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
const mediaFile = ref<InlineMediaFileValue | null>(null);
const existingInlineMediaName = ref("");
const existingInlineMediaMime = ref("");
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
  media_url: "",
  media_name: "",
  media_mime: "",
  media_data_base64: "",
  clear_inline_media: false,
  cron_expr: "",
  run_once_at: "",
  enabled: true,
  pin_after_send: false,
  auto_delete_seconds: 0,
});

const schedulePreview = computed(() => {
  const fields = scheduleFields(false);
  if (!fields) return "请选择执行计划";
  return scheduleLabel({ ...form, id: "", created_at: "", ...fields } as ScheduledPostRecord);
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
const bodyLabel = computed(() => (form.media_type === "text" ? "正文内容" : "配文内容"));
const bodyPlaceholder = computed(() => (form.media_type === "text" ? "输入要发送的文字内容" : "输入要随媒体一起发送的文字内容"));

function optionalText(value?: string | null): string | undefined {
  const text = value?.trim();
  return text || undefined;
}

function toRFC3339(value?: string | null): string | undefined {
  return parseChinaLocalDateTimeToISO(value);
}

function templateLabel(template: MessageTemplateRecord): string {
  return template.name + " · " + (template.chat_id || "全局");
}

function mediaTypeLabel(value?: string): string {
  return { text: "仅文字", photo: "图片 + 文字", video: "视频 + 文字", document: "文件 + 文字" }[value || "text"] || (value || "未知类型");
}

function handleMediaFileUpdate(value: InlineMediaFileValue | null): void {
  mediaFile.value = value;
  if (value) {
    form.media_name = value.name;
    form.media_mime = value.mime_type;
    form.media_data_base64 = value.data_base64;
    form.clear_inline_media = false;
    existingInlineMediaName.value = value.name;
    existingInlineMediaMime.value = value.mime_type;
    form.media_url = "";
    return;
  }
  form.media_name = "";
  form.media_mime = "";
  form.media_data_base64 = "";
  form.clear_inline_media = true;
  existingInlineMediaName.value = "";
  existingInlineMediaMime.value = "";
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
          ElMessage.warning("请先选择发送时间");
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
          ElMessage.warning("请输入自定义发送规则（Cron 表达式）");
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
    const title = scheduleMode.value === "custom" ? "请输入自定义发送规则（Cron 表达式）" : "请先选择发送时间";
    if (showWarning) {
      ElMessage.warning(title);
    }
    return { valid: false, type: "error", title };
  }

  if (fields.run_once_at) {
    const runAt = new Date(fields.run_once_at);
    if (Number.isNaN(runAt.getTime()) || !runAt.getTime()) {
      const title = "发送时间格式无效";
      if (showWarning) ElMessage.warning(title);
      return { valid: false, type: "error", title };
    }
    if (!runAt.getTime() || runAt <= new Date()) {
      const title = "发送时间需晚于当前时间";
      if (showWarning) ElMessage.warning(title);
      return { valid: false, type: "error", title };
    }
    return { valid: true, type: "success", title: "将在北京时间发送 1 次，发送后自动结束" };
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

  return { valid: true, type: "success", title: "发送计划已设置，保存后将按北京时间自动执行" };
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
    return "Cron 表达式需包含 5 段，例如 30 20 * * *";
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
  // Support step syntax: */5 or 1-5/2
  const [base, step] = part.split("/");
  if (step !== undefined) {
    const parsedStep = Number(step);
    if (!Number.isInteger(parsedStep) || parsedStep < 1) {
      return `${range.name}字段步长必须为正整数`;
    }
    if (base === "*") return undefined;
  }
  const effectivePart = step !== undefined ? base : part;
  const values = effectivePart.split(",");
  for (const value of values) {
    const [start, end] = value.split("-");
    const parsedStart = Number(start);
    const parsedEnd = end == null ? parsedStart : Number(end);
    if (!Number.isInteger(parsedStart) || !Number.isInteger(parsedEnd)) {
      return `${range.name}字段仅支持 *、数字、范围、步长（*/n）或逗号分隔`;
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
    ElMessage.warning("请先选择要发送到的群组");
    return undefined;
  }
  const check = validateSchedule();
  if (!check.valid) return undefined;
  const schedule = scheduleFields();
  if (!schedule) return undefined;
  const mediaUrl = optionalText(form.media_url);
  const inlineMediaData = optionalText(form.media_data_base64);
  const inlineMediaName = optionalText(form.media_name);
  const inlineMediaMime = optionalText(form.media_mime);
  const hasInlineMedia = Boolean(inlineMediaData && inlineMediaName && inlineMediaMime);
  const keepExistingInlineMedia = Boolean(editingPost.value?.has_inline_media && !form.clear_inline_media && !mediaUrl && !hasInlineMedia);
  if (form.media_type !== "text" && !mediaUrl && !hasInlineMedia && !keepExistingInlineMedia) {
    ElMessage.warning("图片、视频或文件任务需要填写媒体链接，或上传一个本地文件");
    return undefined;
  }
  if (form.media_type === "text" && !form.content.trim() && !form.title.trim()) {
    ElMessage.warning("文字任务需要填写标题或内容");
    return undefined;
  }
  const shouldClearInlineMedia = form.media_type === "text" || (Boolean(mediaUrl) && !hasInlineMedia) || Boolean(form.clear_inline_media);
  return {
    chat_id: chatId,
    title: form.title.trim(),
    content: form.content.trim(),
    media_type: form.media_type,
    media_url: mediaUrl,
    media_name: hasInlineMedia ? inlineMediaName : undefined,
    media_mime: hasInlineMedia ? inlineMediaMime : undefined,
    media_data_base64: hasInlineMedia ? inlineMediaData : undefined,
    clear_inline_media: shouldClearInlineMedia ? true : undefined,
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
    media_name: "",
    media_mime: "",
    media_data_base64: "",
    clear_inline_media: false,
    cron_expr: "",
    run_once_at: "",
    enabled: true,
    pin_after_send: false,
    auto_delete_seconds: 0,
  });
  mediaFile.value = null;
  existingInlineMediaName.value = "";
  existingInlineMediaMime.value = "";
  selectedTemplateId.value = "";
  scheduleMode.value = "once";
  void loadTemplatesForPost();
  dialogVisible.value = true;
}

function handleMediaTypeChange(): void {
  if (form.media_type === "text") {
    form.media_url = "";
    form.media_name = "";
    form.media_mime = "";
    form.media_data_base64 = "";
    form.clear_inline_media = true;
    mediaFile.value = null;
    existingInlineMediaName.value = "";
    existingInlineMediaMime.value = "";
  } else if (!editingPost.value?.has_inline_media) {
    form.clear_inline_media = false;
  }
}

function handleScheduleModeChange(): void {
  if (scheduleMode.value !== "custom") {
    form.cron_expr = "";
  }
  if (scheduleMode.value !== "once") {
    form.run_once_at = "";
  }
  if (form.media_type === "text") {
    form.media_url = "";
    form.media_name = "";
    form.media_mime = "";
    form.media_data_base64 = "";
    form.clear_inline_media = true;
    mediaFile.value = null;
    existingInlineMediaName.value = "";
    existingInlineMediaMime.value = "";
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
    media_name: row.media_name || "",
    media_mime: row.media_mime || "",
    media_data_base64: "",
    clear_inline_media: false,
    cron_expr: row.cron_expr || "",
    run_once_at: row.run_once_at || "",
    enabled: row.enabled,
    pin_after_send: Boolean(row.pin_after_send),
    auto_delete_seconds: row.auto_delete_seconds || 0,
  });
  mediaFile.value = null;
  existingInlineMediaName.value = row.media_name || "";
  existingInlineMediaMime.value = row.media_mime || "";
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
  form.media_name = "";
  form.media_mime = "";
  form.media_data_base64 = "";
  form.clear_inline_media = false;
  mediaFile.value = null;
  existingInlineMediaName.value = "";
  existingInlineMediaMime.value = "";
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
  try {
    await ElMessageBox.confirm(`确认删除任务「${row.title || row.id}」？`, "删除发布任务", {
      type: "warning",
      confirmButtonText: "删除",
      cancelButtonText: "取消",
    });
  } catch {
    return;
  }
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

function scheduleLabel(row: Pick<ScheduledPostRecord, "cron_expr" | "run_once_at">): string {
  if (row.run_once_at) {
    return `一次性 ${formatDateTime(row.run_once_at)}`;
  }
  const expr = row.cron_expr?.trim();
  if (!expr) return "-";
  if (expr.startsWith("@every ")) {
    return "循环任务 · 每 " + expr.replace("@every ", "");
  }
  const parts = expr.split(/\s+/);
  if (parts.length === 5) {
    const [minute, hour, day, month, weekdayPart] = parts;
    if (day === "*" && month === "*" && weekdayPart === "*") {
      if (hour === "*") return `每小时第 ${minute} 分钟`;
      return `每天 ${padTime(hour)}:${padTime(minute)}`;
    }
    if (day === "*" && month === "*" && weekdayPart !== "*") {
      return "循环任务 · 每周" + weekdayLabel(weekdayPart) + " " + padTime(hour) + ":" + padTime(minute) + "（北京时间）";
    }
    if (day !== "*" && month === "*" && weekdayPart === "*") {
      return "循环任务 · 每月 " + day + " 日 " + padTime(hour) + ":" + padTime(minute) + "（北京时间）";
    }
  }
  return expr;
}

function applyScheduleFromPost(row: Pick<ScheduledPostRecord, "cron_expr" | "run_once_at">): void {
  if (row.run_once_at) {
    scheduleMode.value = "once";
    form.run_once_at = formatChinaInputDateTime(row.run_once_at);
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
.page-meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 5px 10px;
  border: 1px solid var(--app-border);
  border-radius: 999px;
  background: var(--app-tint-light);
}

.posts-dialog :deep(.el-dialog) {
  max-width: calc(100vw - 24px);
}

.post-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

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
