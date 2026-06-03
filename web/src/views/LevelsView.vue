<template>
  <div class="page">
    <PageHeader eyebrow="等级体系" title="等级规则" description="按群组维护积分等级、门槛和权益标记。">
      <template #actions>
        <ChatSelect v-model="filters.chatId" @update:model-value="loadLevels" />
        <el-button :icon="Refresh" :loading="loading" @click="loadLevels">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">新建等级</el-button>
      </template>
    </PageHeader>

    <PanelSection title="等级列表" description="接口：GET/POST/PATCH/DELETE /api/levels。">
      <el-alert v-if="error" class="alert" type="error" :closable="false" show-icon title="服务暂时不可用" />
      <el-table :data="levels" stripe v-loading="loading">
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="chat_id" label="群组" min-width="120" />
        <el-table-column prop="min_points" label="最低积分" width="120" sortable />
        <el-table-column prop="badge" label="徽章" min-width="120" />
        <el-table-column label="权限" min-width="180">
          <template #default="{ row }">{{ formatPermissions(row.permissions) }}</template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" min-width="170" />
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" :loading="deletingId === row.id" @click="removeLevel(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </PanelSection>

    <el-dialog v-model="dialogVisible" :title="editingLevel ? '编辑等级' : '新建等级'" width="520px">
      <el-form label-position="top">
        <el-form-item label="群组 ID">
          <el-input v-if="editingLevel" v-model="form.chat_id" disabled />
          <ChatSelect v-else v-model="form.chat_id" />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="form.name" maxlength="64" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="最低积分">
              <el-input-number v-model="form.min_points" class="wide-control" :min="0" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="徽章">
              <el-input v-model="form.badge" placeholder="VIP / 青铜" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="权限">
          <el-input v-model="permissionText" placeholder="逗号分隔，例如 post,lottery" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitLevel">保存</el-button>
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
import { createLevel, deleteLevel, fetchLevels, updateLevel } from "@/api/levels";
import type { ChatID, LevelPayload, LevelRecord } from "@/types/api";

const loading = ref(false);
const saving = ref(false);
const error = ref(false);
const dialogVisible = ref(false);
const levels = ref<LevelRecord[]>([]);
const editingLevel = ref<LevelRecord>();
const deletingId = ref<ChatID>();
const filters = reactive({ chatId: "" });
const form = reactive({
  chat_id: "",
  name: "",
  min_points: 0,
  badge: "",
});
const permissionText = ref("");

function parseNumericId(value: ChatID): number | undefined {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function parsePermissions(): string[] {
  return permissionText.value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function formatPermissions(permissions?: string[]): string {
  return permissions?.length ? permissions.join(", ") : "-";
}

function resetForm(): void {
  Object.assign(form, {
    chat_id: filters.chatId,
    name: "",
    min_points: 0,
    badge: "",
  });
  permissionText.value = "";
  editingLevel.value = undefined;
}

async function loadLevels(): Promise<void> {
  loading.value = true;
  error.value = false;
  try {
    levels.value = await fetchLevels({ chatId: filters.chatId });
  } catch {
    levels.value = [];
    error.value = true;
    ElMessage.error("服务暂时不可用");
  } finally {
    loading.value = false;
  }
}

function openCreate(): void {
  resetForm();
  dialogVisible.value = true;
}

function openEdit(row: LevelRecord): void {
  editingLevel.value = row;
  Object.assign(form, {
    chat_id: String(row.chat_id),
    name: row.name,
    min_points: row.min_points,
    badge: row.badge ?? "",
  });
  permissionText.value = row.permissions?.join(", ") ?? "";
  dialogVisible.value = true;
}

async function submitLevel(): Promise<void> {
  const chatId = parseNumericId(form.chat_id);
  if (!editingLevel.value && !chatId) {
    ElMessage.warning("请输入有效的群组 ID");
    return;
  }
  if (!form.name.trim()) {
    ElMessage.warning("请输入等级名称");
    return;
  }

  saving.value = true;
  try {
    const payload = {
      name: form.name.trim(),
      min_points: form.min_points,
      badge: form.badge.trim() || undefined,
      permissions: parsePermissions(),
    };
    if (editingLevel.value) {
      await updateLevel(editingLevel.value.id, payload);
    } else {
      await createLevel({ ...payload, chat_id: chatId } as LevelPayload);
    }
    ElMessage.success("等级规则已保存");
    dialogVisible.value = false;
    await loadLevels();
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    saving.value = false;
  }
}

async function removeLevel(row: LevelRecord): Promise<void> {
  await ElMessageBox.confirm(`确认删除等级「${row.name}」？`, "删除等级", {
    type: "warning",
    confirmButtonText: "删除",
    cancelButtonText: "取消",
  });
  deletingId.value = row.id;
  try {
    await deleteLevel(row.id);
    ElMessage.success("等级规则已删除");
    await loadLevels();
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    deletingId.value = undefined;
  }
}

onMounted(loadLevels);
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.alert {
  margin-bottom: 12px;
}

.wide-control {
  width: 100%;
}

</style>
