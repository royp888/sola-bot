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
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding-bottom: 6px;
}

.copy {
  min-width: 0;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

h1 {
  margin: 0;
  font-size: 32px;
  line-height: 1.08;
  font-weight: 780;
}

.eyebrow {
  margin: 0 0 10px;
  color: var(--app-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.description {
  margin: 12px 0 0;
  max-width: 860px;
  color: var(--app-muted-strong);
  font-size: 14px;
  line-height: 1.65;
}

.meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid var(--app-border);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.04);
  color: var(--app-text);
  font-size: 12px;
}

.actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex-wrap: wrap;
}

@media (max-width: 960px) {
  h1 {
    font-size: 28px;
  }
}

@media (max-width: 720px) {
  .page-header {
    flex-direction: column;
  }

  h1 {
    font-size: 24px;
  }

  .actions {
    width: 100%;
    justify-content: stretch;
  }

  .actions :deep(.el-button),
  .actions :deep(.el-select),
  .actions :deep(.el-input) {
    width: 100%;
  }
}
</style>
