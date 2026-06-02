<template>
  <section class="panel">
    <div v-if="title || description || $slots.actions" class="panel-head">
      <div class="panel-copy">
        <h2 v-if="title">{{ title }}</h2>
        <p v-if="description">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="panel-actions">
        <slot name="actions" />
      </div>
    </div>
    <div class="panel-body">
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  title?: string;
  description?: string;
}>();
</script>

<style scoped>
.panel {
  overflow: hidden;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius);
  background: var(--app-surface);
  box-shadow: var(--app-shadow);
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px 12px;
  border-bottom: 1px solid var(--app-border);
}

.panel-copy {
  min-width: 0;
}

h2 {
  margin: 0;
  font-size: 15px;
  line-height: 1.35;
}

p {
  margin: 4px 0 0;
  color: var(--app-muted);
  font-size: 12px;
  line-height: 1.5;
}

.panel-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

.panel-body {
  padding: 16px;
}

@media (max-width: 720px) {
  .panel-head {
    flex-direction: column;
  }

  .panel-actions {
    width: 100%;
    justify-content: stretch;
  }
}
</style>