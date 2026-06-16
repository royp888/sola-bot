<template>
  <div class="page">
    <PageHeader eyebrow="群组管理" title="群组配置" description="欢迎语、入群验证和警告上限。">
      <template #actions>
        <ChatSelect v-model="selectedChatId" @update:model-value="loadConfig" />
        <el-button :icon="Refresh" :loading="loading" @click="loadConfig">刷新</el-button>
        <el-button type="primary" :icon="Check" :loading="saving" @click="submitConfig">保存</el-button>
      </template>
    </PageHeader>

    <el-row :gutter="16">
      <el-col :xs="24" :lg="16">
        <PanelSection title="配置表单" description="对应 /api/admin/config/:chatID。">
          <el-form label-position="top">
            <el-form-item label="欢迎语">
              <el-input v-model="form.welcome_text" type="textarea" :rows="3" />
            </el-form-item>
            <el-row :gutter="12">
              <el-col :xs="24" :md="8">
                <el-form-item label="入群验证">
                  <el-switch v-model="form.verify_enabled" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="8">
                <el-form-item label="验证超时">
                  <el-input-number v-model="form.verify_timeout" class="wide-control" :min="10" :max="3600" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="8">
                <el-form-item label="警告上限">
                  <el-input-number v-model="form.warn_limit" class="wide-control" :min="1" :max="20" />
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>
        </PanelSection>
      </el-col>

      <el-col :xs="24" :lg="8">
        <PanelSection title="预览" description="保存前检查最终下发效果。">
          <div class="preview">
            <div><span>群组 ID</span><strong>{{ selectedChatId || "-" }}</strong></div>
            <div><span>验证</span><strong>{{ form.verify_enabled ? "开启" : "关闭" }}</strong></div>
            <div><span>超时</span><strong>{{ form.verify_timeout }} 秒</strong></div>
            <div><span>上限</span><strong>{{ form.warn_limit }}</strong></div>
            <div class="preview-message">{{ form.welcome_text }}</div>
          </div>
        </PanelSection>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { Check, Refresh } from "@element-plus/icons-vue";
import { useRoute } from "vue-router";
import ChatSelect from "@/components/ChatSelect.vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { fetchAdminConfig, updateAdminConfig } from "@/api/admin";
import type { ChatAdminConfigPayload } from "@/types/api";

const route = useRoute();
const selectedChatId = ref("");
const loading = ref(false);
const saving = ref(false);
const form = reactive<ChatAdminConfigPayload>({
  welcome_text: "",
  verify_enabled: false,
  verify_timeout: 60,
  warn_limit: 3,
});

async function loadConfig(): Promise<void> {
  if (!selectedChatId.value) return;
  loading.value = true;
  try {
    Object.assign(form, await fetchAdminConfig(selectedChatId.value));
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    loading.value = false;
  }
}

async function submitConfig(): Promise<void> {
  if (!selectedChatId.value) return;
  saving.value = true;
  try {
    Object.assign(form, await updateAdminConfig(selectedChatId.value, { ...form }));
    ElMessage.success("群组配置已保存");
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    saving.value = false;
  }
}

onMounted(() => {
  const queryChatID = route.query.chat_id;
  if (typeof queryChatID === "string") {
    selectedChatId.value = queryChatID;
  }
  void loadConfig();
});
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.wide-control {
  width: 100%;
}

.preview {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.preview div {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.preview span {
  color: var(--app-muted);
}

.preview-message {
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  line-height: 1.6;
  background: var(--app-table-header-bg);
}

@media (max-width: 720px) {
}
</style>
