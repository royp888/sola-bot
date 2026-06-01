<template>
  <div class="page">
    <PageHeader eyebrow="Growth" title="邀请链接追踪" description="创建专属邀请链接，并统计通过该链接加入的人数。">
      <template #actions>
        <ChatSelect v-model="selectedChatId" />
        <el-button :icon="Refresh" :loading="loading" @click="loadLinks">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">创建链接</el-button>
      </template>
    </PageHeader>

    <PanelSection title="邀请链接" description="需要 Bot 在群内拥有创建邀请链接权限。">
      <el-table :data="links" stripe>
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="chat_id" label="Chat" min-width="120" />
        <el-table-column prop="invite_link" label="链接" min-width="260" show-overflow-tooltip />
        <el-table-column prop="join_count" label="加入数" width="100" sortable />
        <el-table-column label="审批" width="100">
          <template #default="{ row }">{{ row.creates_join_request ? "开启" : "关闭" }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="removeLink(row)">撤销</el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <el-dialog v-model="dialogVisible" title="创建邀请链接" width="460px">
      <el-form label-position="top">
        <el-form-item label="Chat ID">
          <ChatSelect v-model="form.chat_id" />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="渠道 / 活动名称" />
        </el-form-item>
        <el-form-item label="入群审批">
          <el-switch v-model="form.creates_join_request" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitLink">创建</el-button>
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
import { createInviteLink, deleteInviteLink, fetchInviteLinks } from "@/api/inviteLinks";
import type { ChatID, InviteLinkPayload, InviteLinkRecord } from "@/types/api";

const selectedChatId = ref<ChatID | "">("");
const loading = ref(false);
const saving = ref(false);
const dialogVisible = ref(false);
const links = ref<InviteLinkRecord[]>([]);
const form = reactive<InviteLinkPayload>({
  chat_id: "",
  name: "",
  creates_join_request: false,
});

async function loadLinks(): Promise<void> {
  loading.value = true;
  try {
    links.value = await fetchInviteLinks(selectedChatId.value || undefined);
  } catch {
    links.value = [];
    ElMessage.error("邀请链接接口不可用");
  } finally {
    loading.value = false;
  }
}

function openCreate(): void {
  Object.assign(form, { chat_id: selectedChatId.value, name: "", creates_join_request: false });
  dialogVisible.value = true;
}

async function submitLink(): Promise<void> {
  if (!Number(form.chat_id)) {
    ElMessage.warning("请选择或输入 Chat ID");
    return;
  }
  saving.value = true;
  try {
    await createInviteLink({ ...form, chat_id: Number(form.chat_id) });
    ElMessage.success("邀请链接已创建");
    dialogVisible.value = false;
    await loadLinks();
  } catch {
    ElMessage.error("创建失败，请确认 Bot 权限");
  } finally {
    saving.value = false;
  }
}

async function removeLink(row: InviteLinkRecord): Promise<void> {
  try {
    await ElMessageBox.confirm(`确认撤销邀请链接「${row.name || row.id}」？`, "撤销链接", { type: "warning" });
  } catch {
    return;
  }
  await deleteInviteLink(row.id);
  ElMessage.success("邀请链接已撤销");
  await loadLinks();
}

onMounted(loadLinks);
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
