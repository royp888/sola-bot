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
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: rgba(13, 18, 25, 0.86);
  box-shadow: 0 18px 60px rgba(3, 8, 14, 0.22);
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 14px 0;
}

.panel-copy {
  min-width: 0;
}

h2 {
  margin: 0;
  font-size: 15px;
  line-height: 1.3;
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
  padding: 14px;
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
