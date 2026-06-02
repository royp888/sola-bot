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
  gap: 18px;
  min-height: 122px;
  padding: 18px 20px;
  border: 1px solid var(--app-border);
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(30, 37, 49, 0.95) 0%, rgba(22, 28, 38, 0.95) 100%);
  box-shadow: var(--app-shadow-soft);
}

.meta {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.label-row {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.tone-dot {
  width: 8px;
  height: 8px;
  margin-top: 4px;
  border-radius: 999px;
  background: var(--app-accent);
  box-shadow: 0 0 0 5px rgba(105, 168, 255, 0.12);
}

.label {
  color: var(--app-muted-strong);
  font-size: 13px;
}

.delta {
  color: var(--app-muted);
  font-size: 12px;
  white-space: nowrap;
}

.value {
  font-size: 40px;
  font-weight: 760;
  line-height: 1;
  letter-spacing: 0;
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

@media (max-width: 720px) {
  .stat-tile {
    min-height: 108px;
    padding: 16px 18px;
  }

  .value {
    font-size: 34px;
  }
}
</style>
