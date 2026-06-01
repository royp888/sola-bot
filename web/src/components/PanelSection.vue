<template>
  <section class="panel">
    <div v-if="title || description || $slots.actions" class="panel-head">
      <div>
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
  gap: 16px;
  padding: 18px 18px 0;
}

h2 {
  margin: 0;
  font-size: 16px;
  line-height: 1.3;
}

p {
  margin: 6px 0 0;
  color: var(--app-muted);
  font-size: 13px;
  line-height: 1.5;
}

.panel-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.panel-body {
  padding: 18px;
}
</style>
