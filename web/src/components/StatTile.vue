<template>
  <article class="stat-tile" :data-tone="tone" :data-clickable="clickable" @click="handleClick">
    <div class="meta">
      <div class="label-stack">
        <div class="label-row">
          <span class="tone-dot" />
          <span class="label">{{ label }}</span>
        </div>
        <p v-if="description" class="description">{{ description }}</p>
      </div>
      <span v-if="badgeText" class="badge">{{ badgeText }}</span>
    </div>

    <div class="value-row">
      <div class="value-block">
        <div class="value">{{ value }}</div>
        <p v-if="valueHint" class="value-hint">{{ valueHint }}</p>
      </div>
      <span v-if="delta" class="delta">{{ delta }}</span>
    </div>
  </article>
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    label: string;
    value: string;
    delta?: string;
    description?: string;
    badgeText?: string;
    valueHint?: string;
    clickable?: boolean;
    tone?: "primary" | "success" | "warning" | "danger";
  }>(),
  {
    delta: "",
    description: "",
    badgeText: "",
    valueHint: "",
    clickable: false,
    tone: "primary",
  },
);

const emit = defineEmits<{
  (e: "select"): void;
}>();

function handleClick(): void {
  if (props.clickable) {
    emit("select");
  }
}
</script>

<style scoped>
.stat-tile {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 142px;
  padding: 16px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  background: linear-gradient(180deg, rgba(18, 27, 39, 0.94), rgba(14, 21, 31, 0.94));
  box-shadow: var(--app-shadow-soft);
  transition: transform 0.18s ease, border-color 0.18s ease, background 0.18s ease, box-shadow 0.18s ease;
}

.stat-tile[data-clickable="true"] {
  cursor: pointer;
}

.stat-tile[data-clickable="true"]:hover {
  transform: translateY(-1px);
  border-color: rgba(132, 170, 255, 0.12);
  background: linear-gradient(180deg, rgba(20, 29, 42, 0.96), rgba(15, 23, 34, 0.96));
  box-shadow: var(--app-shadow);
}

.meta,
.value-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.label-stack,
.value-block {
  min-width: 0;
}

.label-stack {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.label-row {
  display: inline-flex;
  align-items: center;
  gap: 9px;
}

.tone-dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: var(--app-accent);
}

.label {
  color: var(--app-muted-strong);
  font-size: 13px;
  font-weight: 600;
}

.description,
.value-hint {
  margin: 0;
  color: var(--app-muted);
  font-size: 12px;
  line-height: 1.6;
}

.badge {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border: 1px solid rgba(255, 255, 255, 0.04);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.025);
  color: var(--app-muted);
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.value-block {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.value {
  font-size: 28px;
  font-weight: 800;
  line-height: 1;
}

.delta {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 9px;
  border-radius: 999px;
  background: rgba(125, 169, 255, 0.1);
  color: var(--app-accent);
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}

.stat-tile[data-tone="success"] .delta {
  background: rgba(114, 192, 145, 0.12);
  color: var(--app-success);
}

.stat-tile[data-tone="success"] .tone-dot {
  background: var(--app-success);
}

.stat-tile[data-tone="warning"] .delta {
  background: rgba(216, 162, 95, 0.12);
  color: var(--app-warning);
}

.stat-tile[data-tone="warning"] .tone-dot {
  background: var(--app-warning);
}

.stat-tile[data-tone="danger"] .delta {
  background: rgba(210, 120, 120, 0.12);
  color: var(--app-danger);
}

.stat-tile[data-tone="danger"] .tone-dot {
  background: var(--app-danger);
}

@media (max-width: 720px) {
  .stat-tile {
    min-height: 132px;
    padding: 15px;
  }

  .meta,
  .value-row {
    flex-direction: column;
    align-items: flex-start;
  }

  .value {
    font-size: 26px;
  }
}
</style>
