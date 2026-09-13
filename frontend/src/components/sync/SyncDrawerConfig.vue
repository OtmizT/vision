<template>
  <div class="drawer-tab-content">
    <div class="section-header">
      <div class="section-title">Módulos Habilitados</div>
      <p class="section-desc">Ative ou desative módulos específicos para esta empresa. Para configurar intervalos e agendamento, use a aba Agendamento.</p>
    </div>

    <div class="executors-list">
      <div v-for="cfg in executorConfigs" :key="cfg.executor" class="executor-item">
        <div class="executor-main">
          <div class="executor-info">
            <span class="executor-name">{{ cfg.executor }}</span>
            <span v-if="cfg.updated_at" class="executor-meta">Atualizado em {{ fmtDate(cfg.updated_at) }}</span>
          </div>
          <div class="executor-actions">
            <button
              class="btn-exec"
              :disabled="!cfg.ativo || forcingExecutor === cfg.executor"
              :title="cfg.ativo ? 'Forçar sync incremental deste módulo' : 'Módulo desativado'"
              @click="handleForcarIncremental(cfg.executor)"
            >
              <span v-if="forcingExecutor === cfg.executor" class="spinner-small"></span>
              <span v-else>▶</span>
            </button>
            <button
              class="btn-exec btn-exec--full"
              :disabled="!cfg.ativo || forcingExecutor === cfg.executor"
              :title="cfg.ativo ? 'Forçar full sync deste módulo' : 'Módulo desativado'"
              @click="handleForcarFull(cfg.executor)"
            >↺</button>
            <label class="switch">
              <input type="checkbox" :checked="cfg.ativo" @change="handleToggleExecutor(cfg)">
              <span class="slider"></span>
            </label>
          </div>
        </div>
        <div v-if="cfg.notas" class="executor-notes">
          {{ cfg.notas }}
        </div>
      </div>
    </div>

    <!-- Modal simples para edição de notas ao trocar status do executor -->
    <div v-if="editingExecutor" class="mini-modal-overlay">
      <div class="mini-modal">
        <div class="section-title">{{ editingExecutor.ativo ? 'DESATIVAR' : 'ATIVAR' }} {{ editingExecutor.executor.toUpperCase() }}</div>
        <div class="form-group">
          <label>NOTAS / MOTIVO</label>
          <textarea v-model="executorNotes" class="input-block" rows="3" placeholder="Opcional..."></textarea>
        </div>
        <div class="modal-actions">
          <button class="btn-inline" @click="editingExecutor = null">CANCELAR</button>
          <button class="btn-inline btn-primary" @click="confirmToggleExecutor">CONFIRMAR</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

interface ExecutorConfig {
  executor: string
  ativo: boolean
  notas?: string
  updated_at?: string
}

const props = defineProps<{
  control: unknown
  executorConfigs: ExecutorConfig[]
  saving: boolean
}>()

const emit = defineEmits<{
  (e: 'salvarConfig', config: unknown): void
  (e: 'toggleExecutor', payload: { executor: string, ativo: boolean, notas: string | null }): void
  (e: 'forcarExecutor', payload: { executor: string, tipo: 'manual' | 'full' }): void
}>()

const editingExecutor = ref<ExecutorConfig | null>(null)
const executorNotes = ref('')
const forcingExecutor = ref<string | null>(null)

function handleToggleExecutor(cfg: ExecutorConfig) {
  editingExecutor.value = cfg
  executorNotes.value = cfg.notas || ''
}

function confirmToggleExecutor() {
  if (!editingExecutor.value) return
  emit('toggleExecutor', {
    executor: editingExecutor.value.executor,
    ativo: !editingExecutor.value.ativo,
    notas: executorNotes.value || null
  })
  editingExecutor.value = null
}

function handleForcarIncremental(executor: string) {
  forcingExecutor.value = executor
  emit('forcarExecutor', { executor, tipo: 'manual' })
  setTimeout(() => { forcingExecutor.value = null }, 3000)
}

function handleForcarFull(executor: string) {
  if (!confirm(`Isso irá reprocessar TODOS os registros de '${executor}'. Confirmar?`)) return
  forcingExecutor.value = executor
  emit('forcarExecutor', { executor, tipo: 'full' })
  setTimeout(() => { forcingExecutor.value = null }, 3000)
}

function fmtDate(d: string) {
  return new Date(d).toLocaleDateString('pt-BR')
}
</script>

<style scoped>
.drawer-tab-content { padding: 24px; }

.section-header { margin-bottom: 20px; }
.section-title { font-family: var(--font-body); font-size: var(--fs-xs); font-weight: 700; color: var(--primary); text-transform: uppercase; letter-spacing: 1px; margin-bottom: 4px; }
.section-desc { font-size: var(--fs-xs); color: var(--text-dim); margin: 0; }

/* Switch Toggle */
.switch { position: relative; display: inline-block; width: 36px; height: 18px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: var(--surface-2); border: 1px solid var(--border-strong); transition: .3s; border-radius: 20px; }
.slider:before { position: absolute; content: ""; height: 12px; width: 12px; left: 2px; bottom: 2px; background-color: var(--text-dim); transition: .3s; border-radius: 50%; }
input:checked + .slider { background-color: var(--primary); border-color: var(--primary); }
input:checked + .slider:before { transform: translateX(18px); background-color: var(--primary); }

.executors-list { display: flex; flex-direction: column; gap: 12px; }
.executor-item { background: var(--surface-2); border: 1px solid var(--border); border-radius: 8px; padding: 12px 16px; }
.executor-main { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.executor-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.btn-exec { background: var(--surface-2); border: 1px solid var(--border-strong); color: var(--text-dim); border-radius: 4px; padding: 4px 8px; font-size: var(--fs-xs); cursor: pointer; transition: var(--transition); min-width: 28px; display: flex; align-items: center; justify-content: center; }
.btn-exec:hover:not(:disabled) { border-color: var(--primary); color: var(--primary); }
.btn-exec--full:hover:not(:disabled) { border-color: var(--warning); color: var(--warning); }
.btn-exec:disabled { opacity: 0.35; cursor: not-allowed; }
.executor-info { display: flex; flex-direction: column; gap: 2px; }
.executor-name { font-family: var(--font-body); font-size: var(--fs-xs); font-weight: 600; color: var(--text); }
.executor-meta { font-size: var(--fs-xs); color: var(--text-dim); }
.executor-notes { margin-top: 8px; font-size: var(--fs-xs); color: var(--text-dim); font-style: italic; border-top: 1px solid var(--surface-2); padding-top: 8px; }

/* Mini Modal */
.mini-modal-overlay { position: fixed; inset: 0; background: var(--overlay); backdrop-filter: blur(2px); z-index: 2100; display: flex; align-items: center; justify-content: center; }
.mini-modal { width: 320px; background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 20px; box-shadow: 0 10px 30px rgba(0,0,0,0.5); }
.form-group { margin: 16px 0; }
.form-group label { display: block; font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text-dim); margin-bottom: 8px; }
.input-block { width: 100%; background: var(--surface-2); border: 1px solid var(--border-strong); border-radius: 6px; padding: 10px; color: var(--text); font-size: var(--fs-xs); outline: none; resize: none; }
.modal-actions { display: flex; gap: 10px; justify-content: flex-end; }
.btn-inline { background: var(--surface-2); border: 1px solid var(--border-strong); color: var(--text); border-radius: 4px; padding: 6px 12px; font-size: var(--fs-xs); cursor: pointer; }
.btn-primary { background: var(--primary) !important; color: var(--bg) !important; font-weight: 700; }

.spinner-small { width: 14px; height: 14px; border: 1.5px solid var(--border-strong); border-top-color: var(--primary); border-radius: 50%; animation: spin 0.7s linear infinite; margin: 0 auto; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
