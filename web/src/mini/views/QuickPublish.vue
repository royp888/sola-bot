<script setup lang="ts">
import { ref } from "vue";
import { request } from "@/api/http";
import { useChatStore } from "@/mini/stores/chat";

const { selectedChat } = useChatStore();

const content = ref("");
const publishing = ref(false);
const error = ref<string | null>(null);
const success = ref(false);

function getChatId(): string | null {
  const chat = selectedChat.value;
  if (!chat) return null;
  return String(chat.chat_id ?? (chat as any).id ?? "");
}

async function publish(): Promise<void> {
  const chatId = getChatId();
  if (!chatId) {
    error.value = "请先选择一个群组";
    return;
  }

  if (!content.value.trim()) {
    error.value = "请输入要发布的内容";
    return;
  }

  publishing.value = true;
  error.value = null;
  success.value = false;

  try {
    await request("/posts", {
      method: "POST",
      body: {
        chat_id: chatId,
        content: content.value,
        media_type: "text",
      },
    });
    success.value = true;
    content.value = "";
  } catch (e: any) {
    error.value = e?.message || "发布失败，请重试";
  } finally {
    publishing.value = false;
  }
}
</script>

<template>
  <div class="scrollable">
    <div v-if="!selectedChat" class="empty">
      <div class="empty-icon">📣</div>
      <p>请先选择一个群组</p>
    </div>

    <template v-else>
      <div style="margin-bottom: 16px;">
        <label style="font-size: 13px; color: var(--tg-hint); display: block; margin-bottom: 6px;">发布内容（支持 HTML）</label>
        <textarea
          class="input textarea"
          v-model="content"
          placeholder="输入要发布的消息内容...&#10;&#10;支持 HTML 标签：&#10;<b>粗体</b> <i>斜体</i> <a href='...'>链接</a>"
        ></textarea>
      </div>

      <div v-if="error" class="error">{{ error }}</div>
      <div v-if="success" style="color: #4caf50; font-size: 14px; margin-bottom: 12px;">✅ 发布成功！</div>

      <button
        class="btn btn-block"
        :disabled="publishing || !content.trim()"
        @click="publish"
      >
        {{ publishing ? '发布中...' : '🚀 立即发布' }}
      </button>

      <div class="html-help">
        <strong>HTML 格式帮助：</strong><br />
        <code>&lt;b&gt;粗体&lt;/b&gt;</code> — <b>粗体</b><br />
        <code>&lt;i&gt;斜体&lt;/i&gt;</code> — <i>斜体</i><br />
        <code>&lt;u&gt;下划线&lt;/u&gt;</code> — <u>下划线</u><br />
        <code>&lt;s&gt;删除线&lt;/s&gt;</code> — <s>删除线</s><br />
        <code>&lt;a href="..."&gt;链接&lt;/a&gt;</code> — 超链接<br />
        <code>&lt;code&gt;代码&lt;/code&gt;</code> — 等宽字体<br />
        <code>&lt;pre&gt;代码块&lt;/pre&gt;</code> — 代码块
      </div>
    </template>
  </div>
</template>
