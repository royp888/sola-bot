<template>
  <header class="page-header">
    <div>
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
  gap: 16px;
  margin-bottom: 18px;
}

h1 {
  margin: 0;
  font-size: 26px;
  line-height: 1.2;
  letter-spacing: 0;
}

.eyebrow {
  margin: 0 0 6px;
  color: var(--app-accent);
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.description {
  margin: 10px 0 0;
  max-width: 720px;
  color: var(--app-muted);
  line-height: 1.6;
}

.actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

@media (max-width: 720px) {
  .page-header {
    flex-direction: column;
  }
}
</style>
