<template>
  <span :class="['status-pill', cls]">
    <span class="status-dot" :class="{ 'status-dot--pulse': status === 'rodando' }" />
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const map: Record<string, { cls: string; label: string }> = {
  // Empresa / Grupo
  ativo:      { cls: 'pill-green',  label: 'Ativo' },
  ativa:      { cls: 'pill-green',  label: 'Ativa' },
  inativo:    { cls: 'pill-gray',   label: 'Inativo' },
  inativa:    { cls: 'pill-gray',   label: 'Inativa' },
  deletando:  { cls: 'pill-red',    label: 'Excluindo' },
  pausado:    { cls: 'pill-yellow', label: 'Pausado' },
  // Sync job
  pendente:   { cls: 'pill-gray',   label: 'Pendente' },
  rodando:    { cls: 'pill-blue',   label: 'Rodando' },
  concluido:  { cls: 'pill-green',  label: 'Concluído' },
  erro:       { cls: 'pill-red',    label: 'Erro' },
  // Status sync empresa
  erro_sync:  { cls: 'pill-red',    label: 'Erro' },
}

const entry   = computed(() => map[props.status] ?? { cls: 'pill-gray', label: props.status })
const cls     = computed(() => entry.value.cls)
const label   = computed(() => entry.value.label)
</script>

<style scoped>
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 9px;
  border-radius: 20px;
  font-family: var(--font-body);
  font-size: var(--fs-xs);
  font-weight: 600;
  white-space: nowrap;
}
.pill-green  { background: var(--success-weak); color: var(--success); }
.pill-red    { background: var(--danger-weak);  color: var(--danger); }
.pill-blue   { background: var(--info-weak);    color: var(--info); }
.pill-yellow { background: var(--warning-weak); color: var(--warning); }
.pill-gray   { background: var(--surface-2);    color: var(--text-dim); }

.status-dot {
  width: 5px; height: 5px;
  border-radius: 50%;
  background: currentColor;
  flex-shrink: 0;
}
.status-dot--pulse { animation: pulse 2s infinite; }
@keyframes pulse { 0%,100%{opacity:1} 50%{opacity:0.3} }
</style>
