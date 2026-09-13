<template>
  <span :class="['badge', `badge-${variant}`]">
    <slot />
  </span>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'green' | 'red' | 'blue' | 'orange' | 'gray' | 'purple' | 'yellow'
}>(), { variant: 'gray' })
</script>

<style scoped>
.badge {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  padding: 3px 9px;
  border-radius: 20px;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}
.badge-green  { background: var(--success-weak); color: var(--success); border: 1px solid var(--success-weak); }
.badge-red    { background: var(--danger-weak);  color: var(--danger);  border: 1px solid var(--danger-weak); }
.badge-blue   { background: var(--info-weak);    color: var(--info);    border: 1px solid var(--info-weak); }
/* Sem token proprio para laranja: warning cobre o mesmo papel semantico e ja
   tem par claro/escuro, ao contrario do var(--warning) que estava fixo aqui. */
.badge-orange { background: var(--warning-weak); color: var(--warning); border: 1px solid var(--warning-weak); }
.badge-gray   { background: var(--surface-2);    color: var(--text-dim);   border: 1px solid var(--border); }
.badge-purple { background: var(--primary-weak); color: var(--primary); border: 1px solid var(--primary-weak); }
.badge-yellow { background: var(--warning-weak); color: var(--warning); border: 1px solid var(--warning-weak); }
</style>
