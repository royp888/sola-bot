<template>
  <div class="page">
    <PageHeader
      eyebrow="Chats"
      title="频道 / 群组"
      description="统一管理频道和群组的权限、同步和成员状态。"
    >
      <template #actions>
        <el-select v-model="kind" class="select" placeholder="类型">
          <el-option label="全部" value="all" />
          <el-option label="频道" value="channel" />
          <el-option label="群组" value="group" />
        </el-select>
        <el-button :icon="Refresh" :loading="loading" @click="loadChats">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openBind">绑定聊天</el-button>
      </template>
    </PageHeader>

    <PanelSection title="绑定列表" description="连接到 Bot 的群组和频道资产。">
      <el-table :data="filteredChats" stripe>
        <el-table-column prop="title" label="名称" min-width="180" />
        <el-table-column label="类型" width="110">
          <template #default="{ row }">
            <el-tag effect="dark" type="info">{{ kindLabel(chatKind(row)) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="permission" label="权限" min-width="150" />
        <el-table-column prop="members" label="成员" width="120" />
        <el-table-column label="同步" width="120">
          <template #default="{ row }">
            <el-tag :type="syncTag(row.syncStatus)" effect="dark">{{ syncLabel(row.syncStatus) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="owner" label="负责人" min-width="120" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button text type="primary" @click="goConfig(row)">配置</el-button>
            <el-button text @click="goLogs(row)">日志</el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <el-dialog v-model="bindVisible" title="绑定聊天" width="520px">
      <el-form label-position="top">
        <el-form-item label="Chat ID">
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
              <el-input v-model="bindForm.bound_by" placeholder="admin" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="名称">
          <el-input v-model="bindForm.title" placeholder="群组或频道名称" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="bindForm.username" placeholder="@username，可选" />
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
import { ElMessage } from "element-plus";
import { Plus, Refresh } from "@element-plus/icons-vue";
import { useRouter } from "vue-router";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { bindChat, fetchChats } from "@/api/chats";
import type { ChatRecord } from "@/types/api";

const router = useRouter();
const kind = ref<"all" | ChatRecord["kind"]>("all");
const chats = ref<ChatRecord[]>([]);
const loading = ref(false);
const saving = ref(false);
const bindVisible = ref(false);
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

async function submitBind(): Promise<void> {
  const chatID = Number(bindForm.chat_id);
  if (!Number.isFinite(chatID) || chatID === 0) {
    ElMessage.warning("请输入有效 Chat ID");
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

function goConfig(chat: ChatRecord): void {
  void router.push({ path: "/admin/config", query: { chat_id: chatValue(chat) } });
}

function goLogs(chat: ChatRecord): void {
  void router.push({ path: "/points/logs", query: { chat_id: chatValue(chat) } });
}

function errorMessage(error: unknown): string {
  const payload = (error as { payload?: { error?: string } })?.payload;
  return payload?.error || "接口不可用";
}

onMounted(loadChats);
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.select {
  width: 160px;
}

.wide-control {
  width: 100%;
}

@media (max-width: 720px) {
  .select {
    width: 100%;
  }
}
</style>
