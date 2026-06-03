<template>
  <header class="page-header">
    <div class="copy">
      <p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
      <div class="title-row">
        <h1>{{ title }}</h1>
        <div v-if="$slots.meta" class="meta">
          <slot name="meta" />
        </div>
      </div>
      <p v-if="description" class="description">
        {{ description }}
      </p>
    </div>
    <div v-if="$slots.actions" class="actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  description?: string;
  eyebrow?: string;
}>();
</script>

<style scoped>
.page-header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: 12px 20px;
}

.copy {
  min-width: 0;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

h1 {
  margin: 0;
  font-size: 26px;
  line-height: 1.12;
  font-weight: 740;
}

.eyebrow {
  margin: 0 0 6px;
  color: var(--app-muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.description {
  margin: 8px 0 0;
  max-width: 680px;
  color: var(--app-muted);
  font-size: 13px;
  line-height: 1.55;
}

.meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 22px;
  color: var(--app-muted-strong);
  font-size: 12px;
  font-weight: 600;
}

.meta::before {
  content: "";
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: var(--app-accent);
}

.actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  justify-self: end;
  align-self: center;
  max-width: min(100%, 640px);
  gap: 10px;
}

.actions :deep(.header-actions-toolbar) {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr)) auto;
  align-items: start;
  justify-content: stretch;
  gap: 10px;
  width: min(100%, 640px);
}

.actions :deep(.header-actions-toolbar > *) {
  min-width: 0;
}

.actions :deep(.header-action-field) {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.actions :deep(.header-action-label) {
  color: var(--app-muted);
  font-size: 11px;
  font-weight: 600;
  line-height: 1.2;
}

.actions :deep(.header-action-field .el-select),
.actions :deep(.header-action-field .el-input),
.actions :deep(.header-action-field .chat-select),
.actions :deep(.header-action-field .user-select) {
  width: 100%;
}

.actions :deep(.header-action-button) {
  min-width: 120px;
}

.actions :deep(.header-action-button .el-button) {
  width: 100%;
}

.actions :deep(.header-action-hint) {
  color: var(--app-muted);
  font-size: 11px;
  line-height: 1.4;
}

@media (max-width: 960px) {
  h1 {
    font-size: 22px;
  }

  .page-header {
    grid-template-columns: 1fr;
  }

  .actions {
    justify-content: flex-start;
    justify-self: stretch;
    max-width: none;
    width: 100%;
  }
}

@media (max-width: 720px) {
  .page-header {
    gap: 14px;
  }

  h1 {
    font-size: 20px;
  }

  .actions {
    width: 100%;
  }

  .actions :deep(.header-actions-toolbar) {
    grid-template-columns: 1fr;
  }

  .actions :deep(.header-action-button),
  .actions :deep(.el-button),
  .actions :deep(.el-select),
  .actions :deep(.el-input),
  .actions :deep(.chat-select),
  .actions :deep(.user-select) {
    width: 100%;
  }
}
</style>
