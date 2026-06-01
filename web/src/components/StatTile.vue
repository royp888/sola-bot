<template>
  <article class="stat-tile" :data-tone="tone">
    <div class="meta">
      <span class="label">{{ label }}</span>
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
  border: 1px solid var(--app-border);
  border-radius: 8px;
  padding: 16px;
  background: linear-gradient(180deg, rgba(20, 27, 36, 0.96), rgba(14, 19, 26, 0.96));
}

.meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--app-muted);
  font-size: 13px;
}

.label {
  white-space: nowrap;
}

.delta {
  color: var(--app-accent);
  white-space: nowrap;
}

.value {
  margin-top: 12px;
  font-size: 28px;
  font-weight: 700;
  line-height: 1;
}

.stat-tile[data-tone="success"] .delta {
  color: var(--app-success);
}

.stat-tile[data-tone="warning"] .delta {
  color: var(--app-warning);
}

.stat-tile[data-tone="danger"] .delta {
  color: var(--app-danger);
}
</style>
