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
  gap: 16px;
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
  font-size: 22px;
  line-height: 1.25;
  font-weight: 700;
  letter-spacing: 0;
}

.eyebrow {
  margin: 0 0 6px;
  color: var(--app-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.description {
  margin: 8px 0 0;
  max-width: 760px;
  color: var(--app-muted);
  font-size: 13px;
  line-height: 1.6;
}

.meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--app-muted);
  font-size: 12px;
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