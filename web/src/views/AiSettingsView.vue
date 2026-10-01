<template>
  <div class="page">
    <PageHeader eyebrow="系统" title="AI 设置" description="配置 AI 对话与广告判定的 LLM 模型，保存后立即生效，无需重启。">
      <template #actions>
        <el-button :icon="Refresh" :loading="loading" @click="loadSettings">刷新</el-button>
      </template>
    </PageHeader>

    <el-row :gutter="16">
      <el-col :xs="24" :lg="14">
        <PanelSection title="模型配置" description="AI 对话（/ai）与 AI 广告判定共用此 LLM 配置，使用 OpenAI 兼容接口。">
          <el-form label-position="top">
            <el-form-item label="启用 AI 功能">
              <el-switch v-model="form.enabled" active-text="启用" inactive-text="停用" />
            </el-form-item>
            <el-form-item label="服务商（Provider）">
              <el-input v-model="form.provider" placeholder="例如 deepseek / openai / gemini" clearable />
            </el-form-item>
            <el-form-item>
              <template #label>
                API Key
                <el-tag v-if="settings.api_key_set" size="small" type="success" style="margin-left:8px">已配置</el-tag>
              </template>
              <el-input
                v-model="form.api_key"
                type="password"
                show-password
                :placeholder="settings.api_key_set ? '已配置，留空保持不变' : 'sk-...'"
              />
            </el-form-item>
            <el-form-item label="模型（Model）">
              <el-input v-model="form.model" placeholder="deepseek-chat" clearable />
            </el-form-item>
            <el-form-item label="接口地址（Endpoint）">
              <el-input v-model="form.endpoint" placeholder="https://api.deepseek.com/v1" clearable />
            </el-form-item>
            <el-form-item label="系统提示词（System Prompt）">
              <el-input
                v-model="form.system_prompt"
                type="textarea"
                :rows="4"
                placeholder="留空则使用默认中文助手提示词"
              />
            </el-form-item>
            <el-form-item>
              <template #label>
                温度（Temperature）
                <span class="field-hint">{{ form.temperature }}</span>
              </template>
              <el-slider v-model="form.temperature" :min="0.1" :max="2" :step="0.1" show-input />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :icon="Check" :loading="saving" @click="saveSettings">保存 AI 配置</el-button>
            </el-form-item>
          </el-form>
        </PanelSection>
      </el-col>

      <el-col :xs="24" :lg="10">
        <PanelSection title="配置状态" description="当前生效的 AI 配置。">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="AI 功能">
              <el-tag :type="settings.enabled ? 'success' : 'danger'">{{ settings.enabled ? "已启用" : "已停用" }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="API Key">
              <el-tag :type="settings.api_key_set ? 'success' : 'danger'">{{ settings.api_key_set ? "已配置" : "未配置" }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="模型">{{ settings.model || "—" }}</el-descriptions-item>
            <el-descriptions-item label="接口地址">{{ settings.endpoint || "—" }}</el-descriptions-item>
          </el-descriptions>
        </PanelSection>

        <PanelSection title="使用说明" description="在 Telegram 里怎么用">
          <ul class="tips">
            <li>群内/私聊发送 <code>/ai 你的问题</code> 或 <code>/gpt 你的问题</code>。</li>
            <li>群里也可直接发 <code>ai 你的问题</code>（无需斜杠）。</li>
            <li>配置保存后<strong>立即生效</strong>，无需重启 Bot。</li>
            <li>API Key 只会写入数据库，回读时不会明文返回。</li>
            <li>同一份配置同时用于「AI 对话」与「AI 广告判定」。</li>
          </ul>
        </PanelSection>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { Check, Refresh } from "@element-plus/icons-vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { fetchAiSettings, updateAiSettings } from "@/api/ai";
import type { AiSettings, AiSettingsUpdatePayload } from "@/api/ai";

const loading = ref(false);
const saving = ref(false);

const settings = reactive<AiSettings>({
  enabled: false,
  provider: "",
  api_key_set: false,
  model: "",
  endpoint: "",
  system_prompt: "",
  temperature: 0.7,
});

const form = reactive({
  enabled: false,
  provider: "",
  api_key: "",
  model: "",
  endpoint: "",
  system_prompt: "",
  temperature: 0.7,
});

async function loadSettings(): Promise<void> {
  loading.value = true;
  try {
    const data = await fetchAiSettings();
    Object.assign(settings, data);
    form.enabled = data.enabled;
    form.provider = data.provider;
    form.api_key = "";
    form.model = data.model;
    form.endpoint = data.endpoint;
    form.system_prompt = data.system_prompt;
    form.temperature = data.temperature || 0.7;
  } catch {
    ElMessage.error("加载 AI 设置失败");
  } finally {
    loading.value = false;
  }
}

async function saveSettings(): Promise<void> {
  saving.value = true;
  try {
    const payload: AiSettingsUpdatePayload = {
      enabled: form.enabled,
      provider: form.provider.trim(),
      model: form.model.trim(),
      endpoint: form.endpoint.trim(),
      system_prompt: form.system_prompt,
      temperature: form.temperature,
    };
    if (form.api_key.trim()) payload.api_key = form.api_key.trim();

    const data = await updateAiSettings(payload);
    Object.assign(settings, data);
    form.api_key = "";
    form.system_prompt = data.system_prompt;
    ElMessage.success("AI 配置已保存，立即生效");
  } catch (err: unknown) {
    const raw = err as { payload?: { error?: string } };
    const msg = raw?.payload?.error ?? (err instanceof Error ? err.message : "保存失败");
    ElMessage.error(msg);
  } finally {
    saving.value = false;
  }
}

onMounted(() => {
  void loadSettings();
});
</script>

<style scoped>
.tips {
  margin: 0;
  padding: 0 0 0 18px;
  color: var(--app-muted);
  font-size: 13px;
  line-height: 1.8;
}

.tips code {
  padding: 1px 4px;
  border-radius: 4px;
  background: var(--app-surface-2);
  font-size: 12px;
}

.field-hint {
  margin-left: 4px;
  color: var(--app-muted);
  font-size: 12px;
}
</style>
