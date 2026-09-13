<template>
  <div :class="['sync-drawer', { 'sync-drawer--open': !!empresa }]">
    <div v-if="empresa" class="drawer-inner">
      <!-- Header -->
      <div class="drawer-header">
        <div class="drawer-header-main">
          <div class="company-info">
            <span class="company-label">EMPRESA SELECIONADA</span>
            <h2 class="company-name">{{ empresa.nome }}</h2>
          </div>
          <button class="btn-close" @click="$emit('close')" title="Fechar (ESC)">✕</button>
        </div>

        <!-- Tabs -->
        <div class="drawer-tabs">
          <button 
            :class="['tab-item', { active: aba === 'progresso' }]" 
            @click="$emit('switchAba', 'progresso')"
          >
            ● PROGRESSO
          </button>
          <button 
            :class="['tab-item', { active: aba === 'historico' }]" 
            @click="$emit('switchAba', 'historico')"
          >
            📋 HISTÓRICO
          </button>
          <button 
            :class="['tab-item', { active: aba === 'config' }]" 
            @click="$emit('switchAba', 'config')"
          >
            ⚙ CONFIG
          </button>
        </div>
      </div>

      <!-- Content -->
      <div class="drawer-body">
        <SyncDrawerProgresso 
          v-if="aba === 'progresso'"
          :job="currentJob"
          :progress="progress"
          @inspecionar-payload="handleInspecionar"
        />

        <SyncDrawerHistorico 
          v-if="aba === 'historico'"
          :empresa-id="empresa.id"
          :selected-job-id="selectedJobId"
          @select-job="(id) => $emit('selectJob', id)"
        />

        <SyncDrawerConfig 
          v-if="aba === 'config'"
          :control="statusMap[empresa.id]?.controle || defaultControl"
          :executor-configs="executorConfigs"
          :saving="saving"
          @salvar-config="(cfg) => $emit('salvarConfig', cfg)"
          @toggle-executor="(p) => $emit('toggleExecutor', p)"
          @forcar-executor="(p) => $emit('forcarExecutor', p)"
        />
      </div>

      <PayloadInspectorModal 
        :visible="showPayloadModal"
        :item="inspectedItem"
        @close="showPayloadModal = false"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import SyncDrawerProgresso from './SyncDrawerProgresso.vue'
import SyncDrawerHistorico from './SyncDrawerHistorico.vue'
import SyncDrawerConfig from './SyncDrawerConfig.vue'
import PayloadInspectorModal from './PayloadInspectorModal.vue'

interface Empresa { id: string; nome: string }
interface SyncJob { id: string; tipo: string; status: string; erro: string; iniciado_at: string | null; concluido_at: string | null; executor?: string }
interface SyncJobProgress {
  executor: string;
  status: string;
  pagina_atual: number;
  total_paginas: number;
  registros_proc: number;
  registros_total: number;
  erro: string | null;
  iniciado_at: string | null;
  concluido_at: string | null;
  updated_at: string;
}
interface SyncControl { 
  ativo: boolean; 
  intervalo_incremental_min: number; 
  intervalo_full_dias: number;
}
interface SyncStatus { controle: SyncControl; ultimo_job: SyncJob | null }
interface ExecutorConfig {
  executor: string;
  ativo: boolean;
  notas?: string;
  updated_at?: string;
}

interface Props {
  empresa: Empresa | null
  aba: 'progresso' | 'historico' | 'config'
  statusMap: Record<string, SyncStatus>
  jobs: SyncJob[]
  progress: SyncJobProgress[]
  loadingJobs: boolean
  executorConfigs: ExecutorConfig[]
  saving: boolean
  selectedJobId: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'switchAba', aba: 'progresso' | 'historico' | 'config'): void
  (e: 'forcar', tipo: string, executor?: string): void
  (e: 'salvarConfig', config: any): void
  (e: 'toggleExecutor', payload: { executor: string, ativo: boolean, notas: string | null }): void
  (e: 'forcarExecutor', payload: { executor: string, tipo: 'manual' | 'full' }): void
  (e: 'selectJob', jobId: string): void
  (e: 'inspecionarPayload', item: SyncJobProgress): void
}>()

const currentJob = computed(() => {
  if (!props.empresa) return null
  // Se estiver na aba progresso, mostra o job selecionado (que por padrão é o último)
  return props.jobs.find(j => j.id === props.selectedJobId) || props.statusMap[props.empresa.id]?.ultimo_job || null
})

const defaultControl: SyncControl = {
  ativo: true,
  intervalo_incremental_min: 60,
  intervalo_full_dias: 7
}

const inspectedItem = ref<SyncJobProgress | null>(null)
const showPayloadModal = ref(false)

function handleInspecionar(item: SyncJobProgress) {
  inspectedItem.value = item
  showPayloadModal.value = true
}
</script>

<style scoped>
.sync-drawer {
  position: fixed;
  top: 0; right: 0; bottom: 0;
  width: 480px;
  background: var(--bg);
  border-left: 1px solid var(--border);
  z-index: 400;
  transform: translateX(100%);
  transition: transform 0.28s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex; flex-direction: column;
}

.sync-drawer--open {
  transform: translateX(0);
  box-shadow: -8px 0 40px rgba(0,0,0,0.4);
}

.drawer-inner {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.drawer-header {
  padding: 24px 24px 0 24px;
  background: var(--surface-2);
  border-bottom: 1px solid var(--border);
}

.drawer-header-main {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.company-label {
  font-family: var(--font-body);
  font-size: var(--fs-xs);
  color: var(--text-dim);
  letter-spacing: 1px;
}

.company-name {
  margin: 4px 0 0 0;
  font-size: var(--fs-lg);
  font-weight: 700;
  color: var(--primary);
}

.btn-close {
  background: var(--surface-2);
  border: 1px solid var(--border-strong);
  color: var(--text-muted);
  border-radius: 6px;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: var(--transition);
}
.btn-close:hover {
  border-color: var(--primary);
  color: var(--primary);
}

.drawer-tabs {
  display: flex;
  gap: 24px;
}

.tab-item {
  background: transparent;
  border: none;
  padding: 0 0 12px 0;
  font-family: var(--font-body);
  font-size: var(--fs-xs);
  font-weight: 700;
  color: var(--text-dim);
  cursor: pointer;
  position: relative;
  transition: var(--transition);
}

.tab-item:hover {
  color: var(--text);
}

.tab-item.active {
  color: var(--primary);
}

.tab-item.active::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--primary);
}

.drawer-body {
  flex: 1;
  overflow-y: auto;
  background: var(--bg);
}

@media (max-width: 1024px) {
  .sync-drawer {
    width: 100%;
  }
}
</style>
