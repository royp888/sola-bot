<template>
  <div class="page-stack chats-page">
    <PageHeader eyebrow="群组资产" title="群组 / 频道" description="把群组资产、同步状态和进入工作台的动作放在同一个界面里。">
      <template #meta>
        <span class="header-meta">共 {{ filteredChats.length }} 个资产</span>
      </template>
      <template #actions>
        <el-button :icon="Refresh" :loading="loading" @click="loadChats">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openBind">绑定聊天</el-button>
      </template>
    </PageHeader>

    <div class="summary-grid">
      <div class="summary-card">
        <div class="summary-label">资产总数</div>
        <div class="summary-value">{{ filteredChats.length }}</div>
        <div class="summary-meta">当前筛选范围内</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">已同步</div>
        <div class="summary-value">{{ syncCounts.synced }}</div>
        <div class="summary-meta">连接稳定且可进入运营流</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">待处理</div>
        <div class="summary-value">{{ syncCounts.pending }}</div>
        <div class="summary-meta">需要补同步或人工跟进</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">受阻</div>
        <div class="summary-value">{{ syncCounts.blocked }}</div>
        <div class="summary-meta">建议优先排查权限或连接状态</div>
      </div>
    </div>

    <PanelSection title="群组资产" description="点击群组名看详情，进入成员台继续做配置、日志和运营动作。">
      <div class="scope-toolbar">
        <div class="scope-main chats-scope">
          <div class="scope-field">
            <label>资产类型</label>
            <el-select v-model="kind" placeholder="类型">
              <el-option label="全部" value="all" />
              <el-option label="频道" value="channel" />
              <el-option label="群组" value="group" />
              <el-option label="超级群" value="supergroup" />
            </el-select>
          </div>
          <div class="scope-field">
            <label>当前结果</label>
            <div class="scope-static">{{ filteredChats.length }} 个资产</div>
          </div>
          <div class="scope-field">
            <label>使用提示</label>
            <div class="scope-static muted-copy">先看同步状态，再决定进入成员台、配置或日志。</div>
          </div>
        </div>
        <div class="scope-meta">
          <span>已同步 {{ syncCounts.synced }} / 待处理 {{ syncCounts.pending }} / 受阻 {{ syncCounts.blocked }}</span>
        </div>
      </div>

      <el-table :data="filteredChats" stripe empty-text="暂无已绑定聊天">
        <el-table-column prop="title" label="群组 / 频道" min-width="240">
          <template #default="{ row }">
            <button class="chat-button" type="button" @click="openDetails(row)">
              <strong>{{ row.title }}</strong>
              <span>{{ row.username || chatValue(row) }}</span>
            </button>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="120">
          <template #default="{ row }">
            <el-tag effect="plain" type="info">{{ kindLabel(chatKind(row)) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="permission" label="权限" min-width="160" />
        <el-table-column prop="members" label="成员" width="120" />
        <el-table-column label="同步" width="120">
          <template #default="{ row }">
            <el-tag :type="syncTag(row.syncStatus)" effect="plain">{{ syncLabel(row.syncStatus) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="owner" label="负责人" min-width="120" />
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="goUsers(row)">进入成员台</el-button>
            <el-button size="small" @click="goConfig(row)">配置</el-button>
            <el-button size="small" @click="goLogs(row)">日志</el-button>
            <el-button size="small" type="danger" :loading="unbindingId === (row.chat_id ?? row.id)" @click="submitUnbind(row)">解绑</el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <el-drawer v-model="detailVisible" title="群组详情" size="400px">
      <template v-if="currentChat">
        <div class="detail-stack">
          <div class="detail-hero">
            <strong>{{ currentChat.title }}</strong>
            <span>{{ currentChat.username || chatValue(currentChat) }}</span>
          </div>
          <div class="detail-grid">
            <div class="detail-card">
              <span>类型</span>
              <strong>{{ kindLabel(chatKind(currentChat)) }}</strong>
            </div>
            <div class="detail-card">
              <span>同步</span>
              <strong>{{ syncLabel(currentChat.syncStatus) }}</strong>
            </div>
            <div class="detail-card">
              <span>权限</span>
              <strong>{{ currentChat.permission || '未标注' }}</strong>
            </div>
            <div class="detail-card">
              <span>负责人</span>
              <strong>{{ currentChat.owner || currentChat.bound_by || '未标注' }}</strong>
            </div>
          </div>
          <div class="detail-actions">
            <el-button type="primary" @click="goUsers(currentChat)">进入成员台</el-button>
            <el-button @click="goConfig(currentChat)">群组设置</el-button>
            <el-button @click="goLogs(currentChat)">积分日志</el-button>
            <el-button type="danger" :loading="unbindingId === (currentChat.chat_id ?? currentChat.id)" @click="submitUnbind(currentChat)">解绑群组</el-button>
          </div>
          <div class="detail-note">
            {{ currentChat.description || '该资产暂无补充说明，可继续进入成员台或群组设置页完成操作。' }}
          </div>
        </div>
      </template>
    </el-drawer>

    <el-dialog v-model="bindVisible" title="绑定聊天" width="520px">
      <el-form label-position="top">
        <el-form-item label="群组 ID">
          <el-input v-model="bindForm.chat_id" placeholder="-100xxxxxxxxxx" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="类型">
              <el-select v-model="bindForm.chat_type" class="wide-control">
                <el-option label="群组" value="group" />
                <el-option label="超级群" value="supergroup" />
                <el-option label="频道" value="channel" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="负责人">
              <el-input v-model="bindForm.bound_by" placeholder="管理员" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="名称">
          <el-input v-model="bindForm.title" placeholder="群组或频道名称" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="bindForm.username" placeholder="@用户名，可选" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="bindForm.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="bindVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitBind">保存绑定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Plus, Refresh } from "@element-plus/icons-vue";
import { useRouter } from "vue-router";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { bindChat, fetchChats, unbindChat } from "@/api/chats";
import type { ChatRecord } from "@/types/api";
import { errorMessage } from "@/utils/helpers";

const router = useRouter();
const kind = ref<"all" | ChatRecord["kind"] | ChatRecord["chat_type"]>("all");
const chats = ref<ChatRecord[]>([]);
const loading = ref(false);
const saving = ref(false);
const bindVisible = ref(false);
const detailVisible = ref(false);
const currentChat = ref<ChatRecord>();
const unbindingId = ref<string | number>();
const bindForm = reactive({
  chat_id: "",
  chat_type: "supergroup",
  title: "",
  username: "",
  bound_by: "admin",
  description: "",
});

const filteredChats = computed(() => {
  if (kind.value === "all") {
    return chats.value;
  }

  return chats.value.filter((chat) => chatKind(chat) === kind.value);
});

const syncCounts = computed(() => {
  return filteredChats.value.reduce(
    (acc, chat) => {
      const key = String(chat.syncStatus ?? "blocked") as "synced" | "pending" | "blocked";
      acc[key] += 1;
      return acc;
    },
    { synced: 0, pending: 0, blocked: 0 },
  );
});

async function loadChats(): Promise<void> {
  loading.value = true;
  try {
    chats.value = await fetchChats();
  } catch (error) {
    chats.value = [];
    ElMessage.error(errorMessage(error));
  } finally {
    loading.value = false;
  }
}

function chatValue(chat: ChatRecord): string {
  return String(chat.chat_id ?? chat.id ?? "");
}

function chatKind(chat: ChatRecord): string {
  return String(chat.chat_type ?? chat.kind ?? "group");
}

function kindLabel(value: ChatRecord["kind"] | ChatRecord["chat_type"]): string {
  if (value === "channel") return "频道";
  if (value === "supergroup") return "超级群";
  if (value === "private") return "私聊";
  return "群组";
}

function syncTag(status: ChatRecord["syncStatus"]): "success" | "warning" | "danger" {
  if (status === "synced") return "success";
  if (status === "pending") return "warning";
  return "danger";
}

function syncLabel(status: ChatRecord["syncStatus"]): string {
  if (status === "synced") return "已同步";
  if (status === "pending") return "待处理";
  return "受阻";
}

function openBind(): void {
  Object.assign(bindForm, {
    chat_id: "",
    chat_type: "supergroup",
    title: "",
    username: "",
    bound_by: "admin",
    description: "",
  });
  bindVisible.value = true;
}

function openDetails(chat: ChatRecord): void {
  currentChat.value = chat;
  detailVisible.value = true;
}

async function submitBind(): Promise<void> {
  const chatID = Number(bindForm.chat_id);
  if (!Number.isFinite(chatID) || chatID === 0) {
    ElMessage.warning("请输入有效的群组 ID");
    return;
  }
  saving.value = true;
  try {
    await bindChat({
      chat_id: chatID,
      chat_type: bindForm.chat_type,
      title: bindForm.title.trim() || String(chatID),
      username: bindForm.username.trim().replace(/^@/, ""),
      bound_by: bindForm.bound_by.trim() || "admin",
      description: bindForm.description.trim(),
    });
    ElMessage.success("聊天已绑定");
    bindVisible.value = false;
    await loadChats();
  } catch (error) {
    ElMessage.error(errorMessage(error));
  } finally {
    saving.value = false;
  }
}

async function submitUnbind(chat: ChatRecord): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `确认解绑群组「${chat.title}」？解绑后该群组的所有配置数据将保留，但机器人将停止在该群响应。`,
      "确认解绑",
      {
        type: "warning",
        confirmButtonText: "确认解绑",
        cancelButtonText: "取消",
      },
    );
  } catch {
    return;
  }
  const chatId = chat.chat_id ?? chat.id;
  unbindingId.value = chatId;
  try {
    await unbindChat(chatId);
    ElMessage.success("群组已解绑");
    detailVisible.value = false;
    await loadChats();
  } catch (error) {
    ElMessage.error(errorMessage(error));
  } finally {
    unbindingId.value = undefined;
  }
}

function goUsers(chat: ChatRecord): void {
  void router.push({ path: "/users", query: { chat_id: chatValue(chat) } });
}

function goConfig(chat: ChatRecord): void {
  void router.push({ path: "/admin/config", query: { chat_id: chatValue(chat) } });
}

function goLogs(chat: ChatRecord): void {
  void router.push({ path: "/points/logs", query: { chat_id: chatValue(chat) } });
}

onMounted(loadChats);
</script>

<style scoped>
.header-meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
  border: 1px solid var(--app-border);
  border-radius: 999px;
  background: var(--app-surface);
}

.chats-scope {
  grid-template-columns: 180px 180px minmax(240px, 1fr);
}

.scope-static {
  display: flex;
  align-items: center;
  min-height: 32px;
  padding: 0 10px;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius);
  background: var(--app-surface-2);
  color: var(--app-text);
  font-size: 13px;
}

.chat-button {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  border: 0;
  padding: 0;
  background: transparent;
  color: var(--app-text);
  text-align: left;
  cursor: pointer;
}

.chat-button span,
.detail-hero span,
.detail-card span,
.detail-note {
  color: var(--app-muted);
  font-size: 12px;
}

.detail-stack {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-hero {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-hero strong {
  font-size: 18px;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.detail-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius);
  background: var(--app-surface-2);
}

.detail-card strong {
  font-size: 13px;
  line-height: 1.5;
}

.detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.wide-control {
  width: 100%;
}

@media (max-width: 1180px) {
  .chats-scope {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .detail-grid,
  .chats-scope {
    grid-template-columns: 1fr;
  }
}
</style>