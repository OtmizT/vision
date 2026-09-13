<template>
  <div class="empty-state fade-up">
    <div class="empty-icon">
      <slot name="icon">
        <!-- Default: inbox icon -->
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2">
          <path d="M2.25 13.5h3.86a2.25 2.25 0 012.012 1.244l.256.512a2.25 2.25 0 002.013 1.244h3.218a2.25 2.25 0 002.013-1.244l.256-.512a2.25 2.25 0 012.013-1.244h3.859m-19.5.338V18a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18v-4.162c0-.224-.034-.447-.1-.661L19.24 5.338a2.25 2.25 0 00-2.15-1.588H6.911a2.25 2.25 0 00-2.15 1.588L2.35 13.177a2.25 2.25 0 00-.1.661z"/>
        </svg>
      </slot>
    </div>
    <p class="empty-title">{{ title }}</p>
    <p v-if="description" class="empty-desc">{{ description }}</p>
    <slot name="action" />
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  title?:       string
  description?: string
}>(), { title: 'Nenhum registro encontrado' })
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 56px 24px;
  gap: 10px;
  text-align: center;
}
.empty-icon {
  width: 48px; height: 48px;
  color: var(--text-dim);
  margin-bottom: 4px;
}
.empty-icon svg { width: 100%; height: 100%; }
.empty-title {
  font-size: var(--fs-base);
  font-weight: 600;
  color: var(--text-muted);
}
.empty-desc {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-dim);
  max-width: 320px;
}
</style>
