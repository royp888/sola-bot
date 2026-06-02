<template>
  <article class="stat-tile" :data-tone="tone">
    <div class="meta">
      <div class="label-row">
        <span class="tone-dot" />
        <span class="label">{{ label }}</span>
      </div>
      <span class="delta">{{ delta }}</span>
    </div>
    <div class="value">{{ value }}</div>
  </article>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string;
    value: string;
    delta: string;
    tone?: "primary" | "success" | "warning" | "danger";
  }>(),
  {
    tone: "primary",
  },
);
</script>

<style scoped>
.stat-tile {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 92px;
  padding: 14px 16px;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius);
  background: var(--app-surface);
  box-shadow: var(--app-shadow);
}

.meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.label-row {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.tone-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: var(--app-accent);
}

.label,
.delta {
  color: var(--app-muted);
  font-size: 12px;
}

.value {
  font-size: 28px;
  font-weight: 700;
  line-height: 1;
}

.stat-tile[data-tone="success"] .tone-dot,
.stat-tile[data-tone="success"] .delta {
  background: var(--app-success);
  color: var(--app-success);
}

.stat-tile[data-tone="warning"] .tone-dot,
.stat-tile[data-tone="warning"] .delta {
  background: var(--app-warning);
  color: var(--app-warning);
}

.stat-tile[data-tone="danger"] .tone-dot,
.stat-tile[data-tone="danger"] .delta {
  background: var(--app-danger);
  color: var(--app-danger);
}
</style>