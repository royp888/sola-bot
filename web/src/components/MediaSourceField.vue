<template>
  <div class="media-source-field">
    <div class="media-url-row">
      <el-input
        :model-value="modelValue"
        class="media-url-input"
        :placeholder="placeholder"
        @update:model-value="emit('update:modelValue', String($event ?? ''))"
      />
      <div class="media-actions">
        <el-button @click="openLocalPicker">从本机选择</el-button>
        <el-button @click="openMobilePicker">从手机拍摄/选择</el-button>
      </div>
    </div>

    <input ref="localInputRef" class="hidden-input" type="file" :accept="accept" @change="onFilePicked" />
    <input ref="mobileInputRef" class="hidden-input" type="file" :accept="accept" capture="environment" @change="onFilePicked" />

    <div v-if="mediaFile || existingFileName" class="media-meta">
      <div>
        <strong>{{ mediaFile?.name || existingFileName }}</strong>
        <span>{{ mediaFile?.mime_type || existingMime || defaultMimeLabel }}</span>
      </div>
      <el-button text type="danger" @click="clearInlineMedia">移除已选文件</el-button>
    </div>

    <div class="media-note">
      <span>支持{{ acceptLabel }}。如果已经上传文件，发送时会优先使用该文件；只有未上传文件时才会使用上方链接。</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { ElMessage } from "element-plus";

export interface InlineMediaFileValue {
  name: string;
  mime_type: string;
  data_base64: string;
}

const props = defineProps<{
  modelValue: string;
  mediaType: "text" | "photo" | "video" | "document";
  mediaFile?: InlineMediaFileValue | null;
  existingFileName?: string;
  existingMime?: string;
  placeholder?: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
  "update:mediaFile": [value: InlineMediaFileValue | null];
}>();

const localInputRef = ref<HTMLInputElement>();
const mobileInputRef = ref<HTMLInputElement>();

const accept = computed(() => {
  if (props.mediaType === "photo") return "image/*";
  if (props.mediaType === "video") return "video/*";
  if (props.mediaType === "document") return "*/*";
  return "image/*,video/*";
});

const acceptLabel = computed(() => {
  if (props.mediaType === "photo") return "图片文件";
  if (props.mediaType === "video") return "视频文件";
  if (props.mediaType === "document") return "文档或其他文件";
  return "图片或视频文件";
});

const defaultMimeLabel = computed(() => {
  if (props.mediaType === "photo") return "图片媒体";
  if (props.mediaType === "video") return "视频媒体";
  if (props.mediaType === "document") return "文件媒体";
  return "媒体文件";
});

function openLocalPicker(): void {
  localInputRef.value?.click();
}

function openMobilePicker(): void {
  mobileInputRef.value?.click();
}

async function onFilePicked(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  try {
    const dataUrl = await readFileAsDataUrl(file);
    const dataBase64 = dataUrl.split(",")[1] || "";
    if (!dataBase64) {
      throw new Error("empty");
    }
    emit("update:modelValue", "");
    emit("update:mediaFile", {
      name: file.name,
      mime_type: file.type || "application/octet-stream",
      data_base64: dataBase64,
    });
    ElMessage.success("已选择媒体文件");
  } catch {
    ElMessage.error("文件读取失败，请重新选择");
  } finally {
    input.value = "";
  }
}

function clearInlineMedia(): void {
  emit("update:mediaFile", null);
}

function readFileAsDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ""));
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}
</script>

<style scoped>
.media-source-field {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.media-url-row {
  display: flex;
  align-items: stretch;
  gap: 10px;
}

.media-url-input {
  flex: 1;
}

.media-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.hidden-input {
  display: none;
}

.media-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 12px 14px;
  border: 1px solid rgba(125, 169, 255, 0.14);
  border-radius: 14px;
  background: rgba(125, 169, 255, 0.08);
}

.media-meta div {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.media-meta strong,
.media-meta span,
.media-note span {
  overflow-wrap: anywhere;
}

.media-meta span,
.media-note {
  color: var(--app-muted);
  font-size: 12px;
}

@media (max-width: 720px) {
  .media-url-row,
  .media-actions {
    flex-direction: column;
  }

  .media-actions :deep(.el-button) {
    width: 100%;
  }
}
</style>
