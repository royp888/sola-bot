<template>
  <header class="page-header">
    <div class="copy">
      <p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
      <h1>{{ title }}</h1>
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
  gap: 12px;
  margin-bottom: 12px;
}

.copy {
  min-width: 0;
}

h1 {
  margin: 0;
  font-size: 24px;
  line-height: 1.2;
  letter-spacing: 0;
}

.eyebrow {
  margin: 0 0 4px;
  color: var(--app-accent);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.description {
  margin: 8px 0 0;
  max-width: 720px;
  color: var(--app-muted);
  font-size: 13px;
  line-height: 1.55;
}

.actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

@media (max-width: 720px) {
  .page-header {
    flex-direction: column;
  }

  .actions {
    width: 100%;
    justify-content: stretch;
  }
}
</style>
