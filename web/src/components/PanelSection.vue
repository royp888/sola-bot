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
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(28, 34, 45, 0.96) 0%, rgba(21, 26, 35, 0.96) 100%);
  box-shadow: var(--app-shadow-soft);
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  padding: 20px 22px 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.panel-copy {
  min-width: 0;
}

h2 {
  margin: 0;
  font-size: 18px;
  line-height: 1.3;
  font-weight: 700;
}

p {
  margin: 6px 0 0;
  color: var(--app-muted);
  font-size: 13px;
  line-height: 1.55;
}

.panel-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex-wrap: wrap;
}

.panel-body {
  padding: 20px 22px 22px;
  overflow-x: auto;
}

@media (max-width: 720px) {
  .panel-head {
    flex-direction: column;
    padding: 18px 18px 14px;
  }

  .panel-body {
    padding: 18px;
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
