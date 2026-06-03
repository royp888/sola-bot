<template>
  <div class="page">
    <PageHeader
      eyebrow="积分运营"
      title="积分配置"
      description="按群组调整消息计分、冷却和积分系统开关。"
    >
      <template #actions>
        <ChatSelect v-model="selectedChatId" @update:model-value="loadConfig" />
        <el-button :icon="Refresh" :loading="loading" @click="loadConfig">刷新</el-button>
        <el-button type="primary" :icon="Check" :loading="saving" @click="submitConfig">
          保存
        </el-button>
      </template>
    </PageHeader>

    <el-row :gutter="16">
      <el-col :xs="24" :lg="16">
        <PanelSection title="规则表单" description="字段会提交到当前 Chat 的 points-config 接口。">
          <el-form
            ref="formRef"
            :model="form"
            :rules="rules"
            label-position="top"
            class="config-form"
          >
            <div class="switch-row">
              <div>
                <strong>积分系统</strong>
                <span>{{ form.enabled ? "已开启" : "已关闭" }}</span>
              </div>
              <el-switch v-model="form.enabled" />
            </div>

            <el-form-item label="防刷冷却时间" prop="cooldown_seconds">
              <el-input-number
                v-model="form.cooldown_seconds"
                class="number-input"
                :min="0"
                :max="86400"
                :step="5"
                controls-position="right"
              />
            </el-form-item>

            <div class="points-grid">
              <el-form-item
                v-for="field in pointFields"
                :key="field.prop"
                :label="field.label"
                :prop="field.prop"
              >
                <el-input-number
                  v-model="form[field.prop]"
                  class="number-input"
                  :min="0"
                  :max="999"
                  controls-position="right"
                />
              </el-form-item>
            </div>
          </el-form>
        </PanelSection>
      </el-col>

      <el-col :xs="24" :lg="8">
        <PanelSection title="当前配置" description="保存前可快速核对即将生效的数值。">
          <div class="summary">
            <div class="summary-row">
              <span>群组 ID</span>
              <strong>{{ selectedChatId || "-" }}</strong>
            </div>
            <div class="summary-row">
              <span>状态</span>
              <el-tag :type="form.enabled ? 'success' : 'info'" effect="dark">
                {{ form.enabled ? "开启" : "关闭" }}
              </el-tag>
            </div>
            <div class="summary-row">
              <span>冷却</span>
              <strong>{{ form.cooldown_seconds }} 秒</strong>
            </div>
            <div v-for="field in pointFields" :key="field.prop" class="summary-row">
              <span>{{ field.shortLabel }}</span>
              <strong>{{ form[field.prop] }}</strong>
            </div>
          </div>
        </PanelSection>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import type { FormInstance, FormRules } from "element-plus";
import { ElMessage } from "element-plus";
import { Check, Refresh } from "@element-plus/icons-vue";
import ChatSelect from "@/components/ChatSelect.vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { fetchChatPointConfig, updateChatPointConfig } from "@/api/pointsConfig";
import type { ChatPointConfig, ChatPointConfigPayload } from "@/types/api";

type PointField = Exclude<keyof ChatPointConfigPayload, "enabled" | "cooldown_seconds">;

const formRef = ref<FormInstance>();
const selectedChatId = ref("");
const loading = ref(false);
const saving = ref(false);

const form = reactive<ChatPointConfigPayload>({
  enabled: true,
  cooldown_seconds: 60,
  point_text: 1,
  point_photo: 3,
  point_sticker: 2,
  point_video: 3,
  point_file: 2,
  point_voice: 3,
});

const pointFields: Array<{ prop: PointField; label: string; shortLabel: string }> = [
  { prop: "point_text", label: "文字消息分值", shortLabel: "文字" },
  { prop: "point_photo", label: "图片消息分值", shortLabel: "图片" },
  { prop: "point_sticker", label: "贴纸消息分值", shortLabel: "贴纸" },
  { prop: "point_video", label: "视频消息分值", shortLabel: "视频" },
  { prop: "point_file", label: "文件消息分值", shortLabel: "文件" },
  { prop: "point_voice", label: "语音消息分值", shortLabel: "语音" },
];

const numberRule = {
  type: "number",
  min: 0,
  message: "请输入不小于 0 的数字",
  trigger: "change",
} as const;

const rules = computed<FormRules<ChatPointConfigPayload>>(() => ({
  cooldown_seconds: [numberRule],
  point_text: [numberRule],
  point_photo: [numberRule],
  point_sticker: [numberRule],
  point_video: [numberRule],
  point_file: [numberRule],
  point_voice: [numberRule],
}));

function applyConfig(payload: ChatPointConfig | ChatPointConfigPayload): void {
  const {
    enabled,
    cooldown_seconds,
    point_text,
    point_photo,
    point_sticker,
    point_video,
    point_file,
    point_voice,
  } = payload;

  Object.assign(form, {
    enabled,
    cooldown_seconds,
    point_text,
    point_photo,
    point_sticker,
    point_video,
    point_file,
    point_voice,
  });
}

async function loadConfig(): Promise<void> {
  if (!selectedChatId.value) {
    ElMessage.warning("请先选择或输入群组 ID");
    return;
  }

  loading.value = true;
  try {
    const payload = await fetchChatPointConfig(selectedChatId.value);
    applyConfig(payload);
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    loading.value = false;
  }
}

async function submitConfig(): Promise<void> {
  if (!selectedChatId.value) {
    ElMessage.warning("请先选择或输入群组 ID");
    return;
  }

  await formRef.value?.validate();

  saving.value = true;
  try {
    const payload = await updateChatPointConfig(selectedChatId.value, { ...form });
    applyConfig(payload);
    ElMessage.success("积分配置已保存");
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    saving.value = false;
  }
}

onMounted(loadConfig);
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.config-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
}

.switch-row div {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.switch-row strong {
  font-size: 14px;
}

.switch-row span,
.summary-row span {
  color: var(--app-muted);
  font-size: 13px;
}

.points-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 14px;
}

.number-input {
  width: 100%;
}

.summary {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 32px;
}

.summary-row strong {
  overflow-wrap: anywhere;
  font-size: 14px;
}

@media (max-width: 720px) {
  .points-grid {
    grid-template-columns: 1fr;
  }
}
</style>
