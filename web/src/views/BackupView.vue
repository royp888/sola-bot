<template>
  <div class="page">
    <PageHeader eyebrow="系统工具" title="备份恢复" description="导出业务配置或全量运营数据，并支持合并/覆盖恢复。">
      <template #actions>
        <el-select v-model="scope" class="select">
          <el-option label="业务配置" value="business" />
          <el-option label="全量数据" value="full" />
        </el-select>
        <el-button type="primary" :loading="exporting" @click="downloadBackup">导出备份</el-button>
      </template>
    </PageHeader>

    <PanelSection title="恢复数据" description="合并模式会追加导入，覆盖模式会先清空备份中包含的表。">
      <el-form label-position="top" class="restore-form">
        <el-form-item label="恢复模式">
          <el-segmented v-model="mode" :options="modeOptions" />
        </el-form-item>
        <el-form-item label="备份文件">
          <el-upload
            drag
            action="#"
            :auto-upload="false"
            :limit="1"
            :on-change="onFileChange"
            :on-remove="() => (file = undefined)"
          >
            <div class="upload-copy">拖拽 JSON 备份文件到这里，或点击选择</div>
          </el-upload>
        </el-form-item>
        <el-button type="warning" :loading="importing" @click="restoreBackup">开始恢复</el-button>
      </el-form>
    </PanelSection>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { ElMessage, ElMessageBox, type UploadFile } from "element-plus";
import PageHeader from "@/components/PageHeader.vue";
import PanelSection from "@/components/PanelSection.vue";
import { exportBackup, importBackupFile } from "@/api/backup";

const scope = ref<"business" | "full">("business");
const mode = ref<"merge" | "overwrite">("merge");
const file = ref<File>();
const exporting = ref(false);
const importing = ref(false);
const modeOptions = [
  { label: "合并", value: "merge" },
  { label: "覆盖", value: "overwrite" },
];

function onFileChange(uploadFile: UploadFile): void {
  file.value = uploadFile.raw;
}

async function downloadBackup(): Promise<void> {
  exporting.value = true;
  try {
    const blob = await exportBackup(scope.value);
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `sola-backup-${scope.value}-${new Date().toISOString().slice(0, 10)}.json`;
    anchor.click();
    URL.revokeObjectURL(url);
    ElMessage.success("备份已导出");
  } catch {
    ElMessage.error("备份导出失败");
  } finally {
    exporting.value = false;
  }
}

async function restoreBackup(): Promise<void> {
  if (!file.value) {
    ElMessage.warning("请先选择备份文件");
    return;
  }
  try {
    await ElMessageBox.confirm(`确认以「${mode.value === "merge" ? "合并" : "覆盖"}」模式恢复数据？`, "恢复确认", {
      type: "warning",
    });
  } catch {
    return;
  }
  importing.value = true;
  try {
    await importBackupFile(file.value, mode.value);
    ElMessage.success("备份已恢复");
  } catch {
    ElMessage.error("备份恢复失败");
  } finally {
    importing.value = false;
  }
}
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

.restore-form {
  max-width: 560px;
}

.upload-copy {
  padding: 28px 12px;
  color: var(--app-muted);
}
</style>
