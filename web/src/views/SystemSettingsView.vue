<template>
  <div class="page">
    <PageHeader eyebrow="系统" title="系统设置" description="Turnstile 验证、管理员密码等全局配置，无需重启即可生效。">
      <template #actions>
        <el-button :icon="Refresh" :loading="loading" @click="loadSettings">刷新</el-button>
      </template>
    </PageHeader>

    <el-row :gutter="16">
      <el-col :xs="24" :lg="14">
        <PanelSection title="机器人信息" description="当前运行的 Telegram Bot 基本信息（只读）。">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="Bot Token">
              <el-tag type="info">{{ settings.bot_token_masked || "未配置" }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="管理员用户名">
              <el-tag>{{ settings.admin_username || "未配置" }}</el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </PanelSection>

        <PanelSection title="Cloudflare Turnstile" description="人机验证配置，用于入群验证 Mini App。修改后立即生效。">
          <el-form label-position="top">
            <el-form-item label="Site Key（前端公钥）">
              <el-input v-model="turnstileForm.site_key" placeholder="0x4AAAAAAA..." clearable />
            </el-form-item>
            <el-form-item>
              <template #label>
                Secret Key（服务端密钥）
                <el-tag v-if="settings.turnstile_secret_key_set" size="small" type="success" style="margin-left:8px">已配置</el-tag>
              </template>
              <el-input
                v-model="turnstileForm.secret_key"
                type="password"
                show-password
                :placeholder="settings.turnstile_secret_key_set ? '已配置，留空保持不变' : '0x4AAAAAAA...'"
              />
            </el-form-item>
            <el-form-item>
              <template #label>
                Verify Secret（HMAC 签名密钥）
                <el-tag v-if="settings.turnstile_verify_secret_set" size="small" type="success" style="margin-left:8px">已配置</el-tag>
              </template>
              <el-input
                v-model="turnstileForm.verify_secret"
                type="password"
                show-password
                :placeholder="settings.turnstile_verify_secret_set ? '已配置，留空保持不变' : '32+ 字节的随机字符串'"
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :icon="Check" :loading="savingTurnstile" @click="saveTurnstile">保存 Turnstile 配置</el-button>
            </el-form-item>
          </el-form>
        </PanelSection>

        <PanelSection title="修改管理员密码" description="修改后立即生效，无需重启。">
          <el-alert
            v-if="settings.admin_password_override"
            title="当前密码已通过数据库设置覆盖"
            type="info"
            :closable="false"
            style="margin-bottom: 16px"
          />
          <el-form label-position="top">
            <el-form-item label="当前密码">
              <el-input v-model="passwordForm.current" type="password" show-password placeholder="输入当前密码" />
            </el-form-item>
            <el-form-item label="新密码">
              <el-input v-model="passwordForm.newPwd" type="password" show-password placeholder="至少 8 位" />
            </el-form-item>
            <el-form-item label="确认新密码">
              <el-input v-model="passwordForm.confirm" type="password" show-password placeholder="再次输入新密码" />
            </el-form-item>
            <el-form-item>
              <el-button type="danger" :icon="Lock" :loading="savingPassword" @click="savePassword">更新密码</el-button>
            </el-form-item>
          </el-form>
        </PanelSection>
      </el-col>

      <el-col :xs="24" :lg="10">
        <PanelSection title="配置状态" description="各项配置的当前生效状态。">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="Turnstile Site Key">
              <el-tag :type="settings.turnstile_site_key ? 'success' : 'danger'">
                {{ settings.turnstile_site_key ? "已配置" : "未配置" }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Turnstile Secret Key">
              <el-tag :type="settings.turnstile_secret_key_set ? 'success' : 'danger'">
                {{ settings.turnstile_secret_key_set ? "已配置" : "未配置" }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Verify Secret">
              <el-tag :type="settings.turnstile_verify_secret_set ? 'success' : 'danger'">
                {{ settings.turnstile_verify_secret_set ? "已配置" : "未配置" }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="管理员密码">
              <el-tag :type="settings.admin_password_override ? 'warning' : 'info'">
                {{ settings.admin_password_override ? "数据库覆盖" : "使用环境变量" }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </PanelSection>

        <PanelSection title="提示" description="配置说明">
          <ul class="tips">
            <li>Turnstile 配置保存到数据库，<strong>立即生效</strong>，无需重启服务。</li>
            <li>Bot Token 只能通过服务器 <code>.env</code> 文件修改。</li>
            <li>密码修改后旧密码立即失效，请妥善记录新密码。</li>
            <li>Verify Secret 用于签名入群验证链接，修改后旧链接将失效。</li>
          </ul>
        </PanelSection>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { Check, Lock, Refresh } from "@element-plus/icons-vue";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { fetchSystemSettings, updateSystemSettings } from "@/api/system-settings";
import type { SystemSettings } from "@/api/system-settings";

const loading = ref(false);
const savingTurnstile = ref(false);
const savingPassword = ref(false);

const settings = reactive<SystemSettings>({
  turnstile_site_key: "",
  turnstile_secret_key_set: false,
  turnstile_verify_secret_set: false,
  admin_username: "",
  admin_password_override: false,
  bot_token_masked: "",
});

const turnstileForm = reactive({
  site_key: "",
  secret_key: "",
  verify_secret: "",
});

const passwordForm = reactive({
  current: "",
  newPwd: "",
  confirm: "",
});

async function loadSettings(): Promise<void> {
  loading.value = true;
  try {
    const data = await fetchSystemSettings();
    Object.assign(settings, data);
    turnstileForm.site_key = data.turnstile_site_key;
    turnstileForm.secret_key = "";
    turnstileForm.verify_secret = "";
  } catch {
    ElMessage.error("加载设置失败");
  } finally {
    loading.value = false;
  }
}

async function saveTurnstile(): Promise<void> {
  savingTurnstile.value = true;
  try {
    const payload: Record<string, string> = {};
    if (turnstileForm.site_key.trim()) payload.turnstile_site_key = turnstileForm.site_key.trim();
    if (turnstileForm.secret_key.trim()) payload.turnstile_secret_key = turnstileForm.secret_key.trim();
    if (turnstileForm.verify_secret.trim()) payload.turnstile_verify_secret = turnstileForm.verify_secret.trim();
    if (Object.keys(payload).length === 0) {
      ElMessage.warning("未修改任何字段");
      return;
    }
    const data = await updateSystemSettings(payload);
    Object.assign(settings, data);
    turnstileForm.site_key = data.turnstile_site_key;
    turnstileForm.secret_key = "";
    turnstileForm.verify_secret = "";
    ElMessage.success("Turnstile 配置已保存");
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : "保存失败";
    ElMessage.error(msg);
  } finally {
    savingTurnstile.value = false;
  }
}

async function savePassword(): Promise<void> {
  if (!passwordForm.current) {
    ElMessage.error("请输入当前密码");
    return;
  }
  if (!passwordForm.newPwd || passwordForm.newPwd.length < 8) {
    ElMessage.error("新密码至少 8 位");
    return;
  }
  if (passwordForm.newPwd !== passwordForm.confirm) {
    ElMessage.error("两次输入的密码不一致");
    return;
  }
  savingPassword.value = true;
  try {
    const data = await updateSystemSettings({
      current_admin_password: passwordForm.current,
      new_admin_password: passwordForm.newPwd,
    });
    Object.assign(settings, data);
    passwordForm.current = "";
    passwordForm.newPwd = "";
    passwordForm.confirm = "";
    ElMessage.success("密码已更新");
  } catch (err: unknown) {
    const raw = err as { payload?: { error?: string } };
    const msg = raw?.payload?.error ?? (err instanceof Error ? err.message : "修改失败");
    ElMessage.error(msg);
  } finally {
    savingPassword.value = false;
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
</style>
