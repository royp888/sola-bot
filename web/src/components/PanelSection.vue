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
  border-radius: 14px;
  background: var(--app-panel-bg);
  box-shadow: var(--app-shadow-soft);
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  padding: 16px 18px 14px;
  border-bottom: 1px solid var(--app-panel-divider);
}

.panel-copy {
  min-width: 0;
}

h2 {
  margin: 0;
  font-size: 15px;
  line-height: 1.3;
  font-weight: 680;
}

p {
  margin: 6px 0 0;
  color: var(--app-muted);
  font-size: 12px;
  line-height: 1.6;
}

.panel-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex-wrap: wrap;
}

.panel-body {
  padding: 18px;
  overflow-x: auto;
}

@media (max-width: 720px) {
  .panel-head {
    flex-direction: column;
    padding: 16px 16px 12px;
  }

  .panel-body {
    padding: 16px;
  }

  .panel-actions {
    width: 100%;
    justify-content: stretch;
  }

  .panel-actions > * {
    width: 100%;
  }
}
</style>
