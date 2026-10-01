<template>
  <div class="page">
    <PageHeader
      eyebrow="运营"
      title="自动转发"
      description="把源频道 / 群的内容搬到你自己的频道：支持历史批量补发 + 实时监听，大文件走服务端复制不落盘，可改写文案并自动挂购买按钮。"
    >
      <template #actions>
        <el-button :icon="Refresh" :loading="loading" @click="reload">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">新建转发规则</el-button>
      </template>
    </PageHeader>

    <div class="stat-grid">
      <StatTile label="规则总数" :value="String(stats.total)" tone="primary" description="源 → 目标 的转发规则" />
      <StatTile label="已启用" :value="String(stats.enabled)" tone="success" description="启用后实时监听才会生效" />
      <StatTile label="实时监听" :value="String(stats.realtime)" tone="primary" description="监听新贴文并立即转发" />
      <StatTile label="历史补发中" :value="String(stats.running)" tone="warning" description="Worker 正在逐条补发历史" />
      <StatTile label="已转发" :value="String(stats.forwarded)" tone="success" description="累计成功条数" />
      <StatTile label="失败 / 跳过" :value="`${stats.failed} / ${stats.skipped}`" tone="danger" description="失败与命中过滤的条数" />
    </div>

    <PanelSection title="转发规则" description="点击行可查看转发明细；历史补发可以随时暂停与重置。">
      <el-table
        v-loading="loading"
        :data="items"
        stripe
        empty-text="还没有转发规则"
        @row-click="openLogs"
      >
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="规则" min-width="180">
          <template #default="{ row }">
            <div class="cell-title">{{ row.name || `转发 #${row.id}` }}</div>
            <div class="cell-sub">{{ modeLabel(row.mode) }} · {{ row.realtime ? "实时" : "仅历史" }}</div>
          </template>
        </el-table-column>
        <el-table-column label="源 → 目标" min-width="260">
          <template #default="{ row }">
            <div class="cell-title">{{ row.source_title || row.source_chat_id }}</div>
            <div class="cell-sub">↓ {{ row.target_title || row.target_chat_id }}</div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.enabled ? 'success' : 'info'">
              {{ row.enabled ? "启用中" : "已停用" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="历史补发" width="230">
          <template #default="{ row }">
            <el-tag size="small" :type="historyTone(row.history_state)">
              {{ historyLabel(row.history_state) }}
            </el-tag>
            <div class="cell-sub">
              成功 {{ row.history_sent }} / 失败 {{ row.history_failed }} / 跳过
              {{ row.history_skipped }}
            </div>
            <div class="cell-sub mono">游标 #{{ row.history_cursor || row.history_from_id || "—" }}</div>
          </template>
        </el-table-column>
        <el-table-column label="购买按钮" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.buy_enabled" size="small" type="warning" effect="plain">已开启</el-tag>
            <span v-else class="cell-sub">—</span>
          </template>
        </el-table-column>
        <el-table-column label="累计" width="150">
          <template #default="{ row }">
            <div class="cell-sub">成功 {{ row.total_forwarded }}</div>
            <div class="cell-sub">失败 {{ row.total_failed }} · 跳过 {{ row.total_skipped }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="290" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.history_state !== 'running'"
              size="small"
              type="primary"
              link
              @click.stop="onStartHistory(row)"
            >开始补发</el-button>
            <el-button
              v-else
              size="small"
              type="warning"
              link
              @click.stop="onPauseHistory(row)"
            >暂停</el-button>
            <el-button size="small" link @click.stop="onResetHistory(row)">重置</el-button>
            <el-button size="small" link @click.stop="openPreview(row)">预览</el-button>
            <el-button size="small" link @click.stop="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" link @click.stop="onDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <!-- 新建 / 编辑 -->
    <el-drawer
      v-model="formVisible"
      :title="editingId ? `编辑转发规则 #${editingId}` : '新建转发规则'"
      size="680px"
    >
      <el-form label-position="top">
        <el-form-item label="规则名称（仅内部可见）">
          <el-input v-model="form.name" placeholder="例如：A 频道 → 我的带货频道" clearable />
        </el-form-item>

        <div class="form-row">
          <el-form-item label="源频道 / 群" style="flex: 1">
            <el-select
              v-model="form.source_chat_id"
              filterable
              clearable
              placeholder="选择机器人已托管的源"
              style="width: 100%"
            >
              <el-option
                v-for="chat in chatOptions"
                :key="`src-${chatKey(chat)}`"
                :label="chatLabel(chat)"
                :value="Number(chat.chat_id ?? chat.id)"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="目标频道 / 群" style="flex: 1">
            <el-select
              v-model="form.target_chat_id"
              filterable
              clearable
              placeholder="选择机器人是管理员的频道"
              style="width: 100%"
            >
              <el-option
                v-for="chat in chatOptions"
                :key="`dst-${chatKey(chat)}`"
                :label="chatLabel(chat)"
                :value="Number(chat.chat_id ?? chat.id)"
              />
            </el-select>
          </el-form-item>
        </div>
        <p class="field-hint">
          机器人需同时是源与目标的管理员；源频道需要能收到 <code>channel_post</code> 更新（重新保存说明即可加入）。
        </p>

        <div class="form-row">
          <el-form-item label="启用规则" style="flex: 1">
            <el-switch v-model="form.enabled" />
          </el-form-item>
          <el-form-item label="实时监听新贴文" style="flex: 1">
            <el-switch v-model="form.realtime" />
          </el-form-item>
        </div>

        <el-form-item label="搬运方式">
          <el-radio-group v-model="form.mode">
            <el-radio-button value="copy">复制（去来源，可改文案/按钮）</el-radio-button>
            <el-radio-button value="forward">原生转发（保留来源）</el-radio-button>
          </el-radio-group>
          <p class="field-hint">
            复制模式走 Telegram 服务端 <code>copyMessage</code>：2GB 以内的大文件不用下载再上传，同时能改文案、挂购买按钮。原生转发无法改文案。
          </p>
        </el-form-item>

        <el-form-item label="相册（多图/多视频）处理">
          <el-switch v-model="form.album_batch" active-text="合并成一组相册" inactive-text="拆成单条" />
          <p class="field-hint">
            合并：用 <code>copyMessages</code> 保留相册分组，但 Telegram 不支持给相册挂按钮，该组会跳过文案改写；
            拆成单条：每条都套用文案与购买按钮（带货推荐）。
          </p>
        </el-form-item>

        <el-divider content-position="left">历史批量补发</el-divider>
        <div class="form-row">
          <el-form-item label="起始消息 ID" style="flex: 1">
            <el-input-number v-model="form.history_from_id" :min="0" :step="1" style="width: 100%" />
          </el-form-item>
          <el-form-item label="结束消息 ID（0 = 追到最新）" style="flex: 1">
            <el-input-number v-model="form.history_to_id" :min="0" :step="1" style="width: 100%" />
          </el-form-item>
        </div>
        <div class="form-row">
          <el-form-item label="每条间隔（毫秒）" style="flex: 1">
            <el-input-number v-model="form.delay_ms" :min="0" :max="60000" :step="100" style="width: 100%" />
          </el-form-item>
          <el-form-item label="每轮条数" style="flex: 1">
            <el-input-number v-model="form.batch_size" :min="1" :max="200" style="width: 100%" />
          </el-form-item>
          <el-form-item label="连续空号判定结束" style="flex: 1">
            <el-input-number v-model="form.history_auto_stop_misses" :min="1" :max="2000" style="width: 100%" />
          </el-form-item>
        </div>
        <p class="field-hint">
          结束 ID 填 0 时会一直往后扫，连续遇到“消息不存在”达到阈值即认定已追到频道尾部并自动完成。
        </p>

        <el-divider content-position="left">过滤与去重</el-divider>
        <el-form-item label="跳过包含这些关键词的消息（逗号分隔）">
          <el-input v-model="form.skip_keywords" placeholder="广告, 加微信, 私聊" clearable />
        </el-form-item>
        <div class="switch-grid">
          <el-checkbox v-model="form.skip_forwarded">跳过带“转发自”的消息</el-checkbox>
          <el-checkbox v-model="form.skip_no_media">只转发带媒体的消息</el-checkbox>
          <el-checkbox v-model="form.skip_no_text">只转发带文案的消息</el-checkbox>
          <el-checkbox v-model="form.skip_duplicates">自动去重（推荐）</el-checkbox>
        </div>
        <div class="form-row">
          <el-form-item label="最小文件（MB，0=不限）" style="flex: 1">
            <el-input-number v-model="form.min_file_size_mb" :min="0" :max="2000" style="width: 100%" />
          </el-form-item>
          <el-form-item label="最大文件（MB，0=不限）" style="flex: 1">
            <el-input-number v-model="form.max_file_size_mb" :min="0" :max="2000" style="width: 100%" />
          </el-form-item>
          <el-form-item label="最短字数（0=不限）" style="flex: 1">
            <el-input-number v-model="form.min_text_length" :min="0" :max="10000" style="width: 100%" />
          </el-form-item>
        </div>

        <el-divider content-position="left">文案改写</el-divider>
        <el-form-item>
          <template #label>
            文案模板
            <span class="field-hint">
              可用占位符：{text} 原文 · {title} 首行 · {link} 原贴链接 · {source} 源名称 · {msg_id} · {buy_url}
            </span>
          </template>
          <el-input
            v-model="form.caption_template"
            type="textarea"
            :rows="5"
            placeholder="留空 = 保留原文，仅套用下面的前缀/后缀&#10;例如：&#10;{text}&#10;&#10;👇 点下面按钮直接下单"
          />
        </el-form-item>
        <div class="form-row">
          <el-form-item label="前缀" style="flex: 1">
            <el-input v-model="form.caption_prefix" type="textarea" :rows="3" placeholder="🔥 今日上新" />
          </el-form-item>
          <el-form-item label="后缀" style="flex: 1">
            <el-input v-model="form.caption_suffix" type="textarea" :rows="3" placeholder="（支持正品验货）" />
          </el-form-item>
        </div>
        <div class="switch-grid">
          <el-checkbox v-model="form.remove_links">删除链接</el-checkbox>
          <el-checkbox v-model="form.remove_mentions">删除 @用户名</el-checkbox>
          <el-checkbox v-model="form.remove_hashtags">删除话题标签</el-checkbox>
          <el-checkbox v-model="form.drop_source_line">删除“来源/Via”那一行</el-checkbox>
          <el-checkbox v-model="form.keep_original_text">模板没写 {text} 时仍保留原文</el-checkbox>
        </div>
        <el-form-item label="解析模式">
          <el-radio-group v-model="form.parse_mode">
            <el-radio-button value="">纯文本（最稳）</el-radio-button>
            <el-radio-button value="HTML">HTML</el-radio-button>
            <el-radio-button value="MarkdownV2">MarkdownV2</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="批量替换（JSON 数组，可选）">
          <el-input
            v-model="form.replacements_json"
            type="textarea"
            :rows="3"
            placeholder='[{"from":"某品牌","to":"我们家"}]'
          />
        </el-form-item>

        <el-divider content-position="left">购买按钮</el-divider>
        <div class="switch-grid">
          <el-checkbox v-model="form.buy_enabled">自动挂购买按钮</el-checkbox>
          <el-checkbox v-model="form.buy_button_on_media">媒体消息显示</el-checkbox>
          <el-checkbox v-model="form.buy_button_on_text">纯文本消息显示</el-checkbox>
        </div>
        <div class="form-row">
          <el-form-item label="按钮文字" style="flex: 1">
            <el-input v-model="form.buy_text" placeholder="🛒 立即购买" />
          </el-form-item>
          <el-form-item label="商品编号提取正则（可选）" style="flex: 1">
            <el-input v-model="form.product_id_regex" placeholder="(?i)(?:#|ID[:：]?\s*)(\d{2,12})" />
          </el-form-item>
        </div>
        <el-form-item>
          <template #label>
            购买链接模板
            <span class="field-hint">
              可用 {product_id} 自动抓取原文里的 #123 / ID:123；也可用 {slug}、{msg_id}、{chat_id}
            </span>
          </template>
          <el-input
            v-model="form.buy_url_template"
            placeholder="https://shop.example.com/product/{product_id}"
          />
        </el-form-item>
        <el-form-item label="附加按钮（JSON，可选）">
          <el-input
            v-model="form.extra_buttons_json"
            type="textarea"
            :rows="3"
            placeholder='[{"buttons":[{"text":"📦 物流查询","url":"https://t.me/xxx"}]}]'
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="formVisible = false">取消</el-button>
        <el-button v-if="editingId" :loading="previewing" @click="previewCurrent">预览文案</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-drawer>

    <!-- 预览 -->
    <el-dialog v-model="previewVisible" title="转发效果预览" width="620px">
      <el-form label-position="top">
        <el-form-item label="模拟原文（当前规则会按它渲染）">
          <el-input v-model="previewForm.text" type="textarea" :rows="5" />
        </el-form-item>
        <div class="switch-grid">
          <el-checkbox v-model="previewForm.has_media">媒体消息</el-checkbox>
          <el-checkbox v-model="previewForm.is_forwarded">来自转发</el-checkbox>
        </div>
        <div class="form-row">
          <el-form-item label="媒体类型" style="flex: 1">
            <el-select v-model="previewForm.media_type" style="width: 100%">
              <el-option label="纯文本" value="" />
              <el-option label="图片" value="photo" />
              <el-option label="视频" value="video" />
              <el-option label="文件" value="document" />
            </el-select>
          </el-form-item>
          <el-form-item label="文件大小（MB）" style="flex: 1">
            <el-input-number v-model="previewForm.file_size_mb" :min="0" :max="2000" style="width: 100%" />
          </el-form-item>
        </div>
        <el-button :loading="previewing" @click="runPreview">重新渲染</el-button>
      </el-form>

      <el-alert v-if="preview?.skipped" type="warning" :closable="false" class="preview-alert"
        :title="`这条消息会被过滤：${skipReasonLabel(preview.skip_reason)}`" />
      <el-alert
        v-for="(warning, index) in preview?.warnings ?? []"
        :key="`warn-${index}`"
        type="warning"
        :closable="false"
        class="preview-alert"
        :title="warning"
      />

      <div class="preview-card">
        <pre class="preview-caption">{{ preview?.caption || "（没有文字）" }}</pre>
        <div v-if="preview?.buttons?.length" class="preview-buttons">
          <el-button
            v-for="(row, rowIndex) in preview.buttons"
            :key="`row-${rowIndex}`"
            class="preview-button"
            plain
          >
            {{ row.buttons.map((button) => button.text).join(" | ") }}
          </el-button>
        </div>
      </div>

      <template #footer>
        <el-button @click="previewVisible = false">关闭</el-button>
        <el-button type="primary" :loading="sending" :disabled="!previewRuleId" @click="runTestSend">
          发一条测试到目标频道
        </el-button>
      </template>
    </el-dialog>

    <!-- 转发明细 -->
    <el-drawer v-model="logsVisible" :title="logsTitle" size="720px">
      <div class="logs-toolbar">
        <el-button :icon="Refresh" size="small" :loading="logsLoading" @click="loadLogs">刷新</el-button>
        <span class="cell-sub">最近 {{ logs.length }} 条</span>
      </div>
      <el-table v-loading="logsLoading" :data="logs" stripe empty-text="还没有转发记录">
        <el-table-column prop="source_message_id" label="源消息" width="110" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="logTone(row.status)">{{ logLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="类型 / 大小" width="130">
          <template #default="{ row }">
            <div class="cell-sub">{{ row.media_type || "text" }}</div>
            <div class="cell-sub">{{ formatSize(row.file_size) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="内容 / 错误" min-width="220">
          <template #default="{ row }">
            <div class="cell-sub">{{ truncate(row.text, 80) || "—" }}</div>
            <div v-if="row.error" class="cell-sub danger">{{ truncate(row.error, 120) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="目标消息" width="100">
          <template #default="{ row }">{{ row.target_message_id || "—" }}</template>
        </el-table-column>
        <el-table-column label="时间" width="160">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Plus, Refresh } from "@element-plus/icons-vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import StatTile from "@/components/StatTile.vue";
import { fetchChats } from "@/api/chats";
import type { ChatRecord } from "@/types/api";
import {
  createForward,
  deleteForward,
  fetchForwardLogs,
  fetchForwards,
  fetchForwardStats,
  pauseForwardHistory,
  previewForward,
  resetForwardHistory,
  startForwardHistory,
  testSendForward,
  updateForward,
} from "@/api/forwards";
import type {
  AutoForward,
  AutoForwardLog,
  AutoForwardPayload,
  AutoForwardPreview,
  AutoForwardStats,
} from "@/api/forwards";

const loading = ref(false);
const saving = ref(false);
const previewing = ref(false);
const sending = ref(false);
const logsLoading = ref(false);

const items = ref<AutoForward[]>([]);
const chats = ref<ChatRecord[]>([]);
const logs = ref<AutoForwardLog[]>([]);

const stats = reactive<AutoForwardStats>({
  total: 0,
  enabled: 0,
  realtime: 0,
  running: 0,
  forwarded: 0,
  failed: 0,
  skipped: 0,
});

const formVisible = ref(false);
const previewVisible = ref(false);
const logsVisible = ref(false);
const editingId = ref<number | null>(null);
const logsRuleId = ref<number | null>(null);
const previewRuleId = ref<number | null>(null);
const preview = ref<AutoForwardPreview | null>(null);

const form = reactive({
  name: "",
  enabled: true,
  source_chat_id: 0,
  target_chat_id: 0,
  mode: "copy" as "copy" | "forward",
  parse_mode: "",
  realtime: true,
  history_from_id: 0,
  history_to_id: 0,
  history_auto_stop_misses: 25,
  delay_ms: 800,
  batch_size: 20,
  album_batch: false,
  skip_keywords: "",
  skip_forwarded: false,
  skip_no_media: false,
  skip_no_text: false,
  skip_duplicates: true,
  min_file_size_mb: 0,
  max_file_size_mb: 0,
  min_text_length: 0,
  caption_template: "",
  caption_prefix: "",
  caption_suffix: "",
  remove_links: true,
  remove_mentions: true,
  remove_hashtags: false,
  drop_source_line: true,
  keep_original_text: true,
  replacements_json: "",
  buy_enabled: true,
  buy_text: "🛒 立即购买",
  buy_url_template: "",
  product_id_regex: "",
  buy_button_on_media: true,
  buy_button_on_text: true,
  extra_buttons_json: "",
});

const previewForm = reactive({
  text: "示例商品文案 #123\n原价 199，今日特价 99",
  has_media: true,
  media_type: "photo",
  file_size_mb: 2,
  is_forwarded: false,
});

const chatOptions = computed(() => chats.value.filter((chat) => chatKey(chat) !== ""));
const logsTitle = computed(() =>
  logsRuleId.value ? `转发明细 · 规则 #${logsRuleId.value}` : "转发明细",
);

let logTimer: number | undefined;

onMounted(async () => {
  await Promise.all([reload(), loadChats()]);
  logTimer = window.setInterval(() => {
    if (logsVisible.value) {
      void loadLogs(true);
    }
  }, 8000);
});

onUnmounted(() => {
  if (logTimer) {
    window.clearInterval(logTimer);
  }
});

async function loadChats(): Promise<void> {
  try {
    chats.value = await fetchChats();
  } catch {
    chats.value = [];
  }
}

async function reload(): Promise<void> {
  loading.value = true;
  try {
    const [list, statsPayload] = await Promise.all([fetchForwards(), fetchForwardStats()]);
    items.value = list;
    Object.assign(stats, statsPayload);
  } catch (error) {
    ElMessage.error(errorText(error, "加载转发规则失败"));
  } finally {
    loading.value = false;
  }
}

function openCreate(): void {
  editingId.value = null;
  Object.assign(form, {
    name: "",
    enabled: true,
    source_chat_id: 0,
    target_chat_id: 0,
    mode: "copy",
    parse_mode: "",
    realtime: true,
    history_from_id: 0,
    history_to_id: 0,
    history_auto_stop_misses: 25,
    delay_ms: 800,
    batch_size: 20,
    album_batch: false,
    skip_keywords: "",
    skip_forwarded: false,
    skip_no_media: false,
    skip_no_text: false,
    skip_duplicates: true,
    min_file_size_mb: 0,
    max_file_size_mb: 0,
    min_text_length: 0,
    caption_template: "",
    caption_prefix: "",
    caption_suffix: "",
    remove_links: true,
    remove_mentions: true,
    remove_hashtags: false,
    drop_source_line: true,
    keep_original_text: true,
    replacements_json: "",
    buy_enabled: true,
    buy_text: "🛒 立即购买",
    buy_url_template: "",
    product_id_regex: "",
    buy_button_on_media: true,
    buy_button_on_text: true,
    extra_buttons_json: "",
  });
  formVisible.value = true;
}

function openEdit(row: AutoForward): void {
  editingId.value = row.id;
  Object.assign(form, {
    name: row.name,
    enabled: row.enabled,
    source_chat_id: row.source_chat_id,
    target_chat_id: row.target_chat_id,
    mode: row.mode,
    parse_mode: row.parse_mode,
    realtime: row.realtime,
    history_from_id: row.history_from_id,
    history_to_id: row.history_to_id,
    history_auto_stop_misses: row.history_auto_stop_misses || 25,
    delay_ms: row.delay_ms || 800,
    batch_size: row.batch_size || 20,
    album_batch: row.album_batch,
    skip_keywords: row.skip_keywords,
    skip_forwarded: row.skip_forwarded,
    skip_no_media: row.skip_no_media,
    skip_no_text: row.skip_no_text,
    skip_duplicates: row.skip_duplicates,
    min_file_size_mb: row.min_file_size_mb,
    max_file_size_mb: row.max_file_size_mb,
    min_text_length: row.min_text_length,
    caption_template: row.caption_template,
    caption_prefix: row.caption_prefix,
    caption_suffix: row.caption_suffix,
    remove_links: row.remove_links,
    remove_mentions: row.remove_mentions,
    remove_hashtags: row.remove_hashtags,
    drop_source_line: row.drop_source_line,
    keep_original_text: row.keep_original_text,
    replacements_json: row.replacements_json,
    buy_enabled: row.buy_enabled,
    buy_text: row.buy_text || "🛒 立即购买",
    buy_url_template: row.buy_url_template,
    product_id_regex: row.product_id_regex,
    buy_button_on_media: row.buy_button_on_media,
    buy_button_on_text: row.buy_button_on_text,
    extra_buttons_json: row.extra_buttons_json,
  });
  formVisible.value = true;
}

function buildPayload(): AutoForwardPayload {
  const source = chats.value.find((chat) => Number(chat.chat_id ?? chat.id) === form.source_chat_id);
  const target = chats.value.find((chat) => Number(chat.chat_id ?? chat.id) === form.target_chat_id);
  return {
    name: form.name,
    enabled: form.enabled,
    source_chat_id: Number(form.source_chat_id) || 0,
    source_title: source?.title ?? "",
    target_chat_id: Number(form.target_chat_id) || 0,
    target_title: target?.title ?? "",
    mode: form.mode,
    parse_mode: form.parse_mode,
    realtime: form.realtime,
    history_from_id: Number(form.history_from_id) || 0,
    history_to_id: Number(form.history_to_id) || 0,
    history_auto_stop_misses: Number(form.history_auto_stop_misses) || 25,
    delay_ms: Number(form.delay_ms) || 0,
    batch_size: Number(form.batch_size) || 20,
    album_batch: form.album_batch,
    skip_keywords: form.skip_keywords,
    skip_forwarded: form.skip_forwarded,
    skip_no_media: form.skip_no_media,
    skip_no_text: form.skip_no_text,
    skip_duplicates: form.skip_duplicates,
    min_file_size_mb: Number(form.min_file_size_mb) || 0,
    max_file_size_mb: Number(form.max_file_size_mb) || 0,
    min_text_length: Number(form.min_text_length) || 0,
    caption_template: form.caption_template,
    caption_prefix: form.caption_prefix,
    caption_suffix: form.caption_suffix,
    remove_links: form.remove_links,
    remove_mentions: form.remove_mentions,
    remove_hashtags: form.remove_hashtags,
    drop_source_line: form.drop_source_line,
    keep_original_text: form.keep_original_text,
    replacements_json: form.replacements_json,
    buy_enabled: form.buy_enabled,
    buy_text: form.buy_text,
    buy_url_template: form.buy_url_template,
    product_id_regex: form.product_id_regex,
    buy_button_on_media: form.buy_button_on_media,
    buy_button_on_text: form.buy_button_on_text,
    extra_buttons_json: form.extra_buttons_json,
  };
}

async function save(): Promise<void> {
  if (!form.source_chat_id || !form.target_chat_id) {
    ElMessage.warning("请先选择源与目标频道");
    return;
  }
  if (form.source_chat_id === form.target_chat_id) {
    ElMessage.warning("源和目标的频道不能相同");
    return;
  }
  saving.value = true;
  try {
    const payload = buildPayload();
    if (editingId.value) {
      await updateForward(editingId.value, payload);
      ElMessage.success("规则已更新");
    } else {
      await createForward(payload);
      ElMessage.success("规则已创建");
    }
    formVisible.value = false;
    await reload();
  } catch (error) {
    ElMessage.error(errorText(error, "保存失败"));
  } finally {
    saving.value = false;
  }
}

async function onStartHistory(row: AutoForward): Promise<void> {
  try {
    await startForwardHistory(row.id);
    ElMessage.success("已开始历史补发，Worker 会按间隔逐条搬运");
    await reload();
  } catch (error) {
    ElMessage.error(errorText(error, "启动失败"));
  }
}

async function onPauseHistory(row: AutoForward): Promise<void> {
  try {
    await pauseForwardHistory(row.id);
    ElMessage.success("已暂停，游标已保存，可随时继续");
    await reload();
  } catch (error) {
    ElMessage.error(errorText(error, "暂停失败"));
  }
}

async function onResetHistory(row: AutoForward): Promise<void> {
  try {
    await ElMessageBox.confirm("重置会清空历史进度（不会删除已转发的消息），确定继续？", "重置历史", {
      type: "warning",
    });
  } catch {
    return;
  }
  try {
    await resetForwardHistory(row.id);
    ElMessage.success("进度已重置");
    await reload();
  } catch (error) {
    ElMessage.error(errorText(error, "重置失败"));
  }
}

async function onDelete(row: AutoForward): Promise<void> {
  try {
    await ElMessageBox.confirm(`确定删除规则「${row.name || row.id}」及其转发记录？`, "删除规则", {
      type: "warning",
    });
  } catch {
    return;
  }
  try {
    await deleteForward(row.id);
    ElMessage.success("已删除");
    await reload();
  } catch (error) {
    ElMessage.error(errorText(error, "删除失败"));
  }
}

function openPreview(row: AutoForward): void {
  previewRuleId.value = row.id;
  preview.value = null;
  previewVisible.value = true;
  if (row.id === editingId.value) {
    previewForm.text = previewForm.text || "示例商品文案 #123";
  }
  void runPreview();
}

async function previewCurrent(): Promise<void> {
  if (!editingId.value) return;
  previewRuleId.value = editingId.value;
  previewVisible.value = true;
  await runPreview();
}

async function runPreview(): Promise<void> {
  if (!previewRuleId.value) return;
  previewing.value = true;
  try {
    preview.value = await previewForward(previewRuleId.value, {
      text: previewForm.text,
      has_media: previewForm.has_media,
      media_type: previewForm.has_media ? previewForm.media_type : "",
      file_size: Number(previewForm.file_size_mb) * 1000 * 1000,
      is_forwarded: previewForm.is_forwarded,
      message_id: 12345,
    });
  } catch (error) {
    ElMessage.error(errorText(error, "预览失败"));
  } finally {
    previewing.value = false;
  }
}

async function runTestSend(): Promise<void> {
  if (!previewRuleId.value) return;
  sending.value = true;
  try {
    await testSendForward(previewRuleId.value, {
      text: previewForm.text,
      has_media: previewForm.has_media,
      media_type: previewForm.has_media ? previewForm.media_type : "",
      file_size: 0,
      is_forwarded: previewForm.is_forwarded,
      message_id: 12345,
    });
    ElMessage.success("测试消息已发到目标频道");
  } catch (error) {
    ElMessage.error(errorText(error, "试发失败，请确认机器人是目标频道管理员"));
  } finally {
    sending.value = false;
  }
}

function openLogs(row: AutoForward): void {
  logsRuleId.value = row.id;
  logsVisible.value = true;
  void loadLogs();
}

async function loadLogs(silent = false): Promise<void> {
  if (!logsRuleId.value) return;
  if (!silent) {
    logsLoading.value = true;
  }
  try {
    logs.value = await fetchForwardLogs(logsRuleId.value, 100);
  } catch (error) {
    if (!silent) {
      ElMessage.error(errorText(error, "加载明细失败"));
    }
  } finally {
    logsLoading.value = false;
  }
}

function chatKey(chat: ChatRecord): string {
  return String(chat.chat_id ?? chat.id ?? "");
}

function chatLabel(chat: ChatRecord): string {
  const id = chatKey(chat);
  const title = chat.title || chat.username || id;
  return `${title} · ${id}`;
}

function modeLabel(mode: string): string {
  return mode === "forward" ? "原生转发" : "复制并改写";
}

function historyLabel(state: string): string {
  return (
    {
      idle: "未开始",
      running: "补发中",
      paused: "已暂停",
      completed: "已完成",
      failed: "出错",
    }[state] ?? state
  );
}

function historyTone(state: string): "success" | "warning" | "danger" | "info" | "primary" {
  if (state === "completed") return "success";
  if (state === "running") return "primary";
  if (state === "paused") return "warning";
  if (state === "failed") return "danger";
  return "info";
}

function logLabel(status: string): string {
  return { sent: "成功", failed: "失败", skipped: "跳过" }[status] ?? status;
}

function logTone(status: string): "success" | "danger" | "warning" {
  if (status === "sent") return "success";
  if (status === "failed") return "danger";
  return "warning";
}

function skipReasonLabel(reason: string): string {
  const labels: Record<string, string> = {
    duplicate: "重复消息（已转发过）",
    forwarded: "带“转发自”来源",
    no_media: "没有媒体",
    no_text: "没有文案",
    text_too_short: "文案太短",
    file_too_small: "文件太小",
    file_too_large: "文件超过上限",
    empty_caption: "改写后文案为空",
    message_not_found: "消息不存在（历史空号）",
  };
  if (reason?.startsWith("keyword:")) {
    return `命中关键词 ${reason.slice("keyword:".length)}`;
  }
  return labels[reason] ?? reason ?? "";
}

function formatTime(value?: string | null): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function truncate(value: string, max: number): string {
  if (!value) return "";
  return value.length > max ? `${value.slice(0, max)}…` : value;
}

function formatSize(bytes: number): string {
  if (!bytes) return "—";
  if (bytes >= 1000 * 1000 * 1000) return `${(bytes / 1e9).toFixed(2)} GB`;
  if (bytes >= 1000 * 1000) return `${(bytes / 1e6).toFixed(1)} MB`;
  if (bytes >= 1000) return `${(bytes / 1e3).toFixed(0)} KB`;
  return `${bytes} B`;
}

function errorText(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) {
    return `${fallback}：${error.message}`;
  }
  const payload = (error as { payload?: { error?: string } })?.payload;
  if (payload?.error) {
    return `${fallback}：${payload.error}`;
  }
  return fallback;
}
</script>

<style scoped>
.form-row {
  display: flex;
  gap: 12px;
}

.switch-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 20px;
  margin-bottom: 16px;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.danger {
  color: var(--el-color-danger);
}

.preview-alert {
  margin-bottom: 10px;
}

.preview-card {
  margin-top: 12px;
  padding: 14px;
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
}

.preview-caption {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
  font-size: 14px;
  line-height: 1.6;
}

.preview-buttons {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 12px;
}

.preview-button {
  width: 100%;
}

.logs-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.field-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-weight: 400;
}

.field-hint code {
  padding: 0 4px;
  border-radius: 4px;
  background: var(--el-fill-color);
}

.switch-grid :deep(.el-checkbox) {
  margin-right: 0;
}
</style>
