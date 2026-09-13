<template>
  <div class="empresa-page">

    <!-- Header com navegação -->
    <div class="page-header">
      <button class="btn-back" @click="router.push('/sync')">
        ← Voltar
      </button>
      <div class="breadcrumb">
        <span class="bc-root">SINCRONIZAÇÃO</span>
        <span class="bc-sep">/</span>
        <span class="bc-current">{{ empresa?.nome ?? empresaId }}</span>
      </div>
      <div class="header-actions">
        <span v-if="isRunning" class="badge-running">
          <span class="dot-pulse"></span>
          Em execução
        </span>
        <button class="btn-action" title="Forçar Incremental" :disabled="!podeForcar" @click="tentarForcar('manual')">
          <span v-if="forcingType === 'manual'" class="spinner-small"></span>
          <span v-else>▶ Incremental</span>
        </button>
        <button class="btn-action btn-action-warn" title="Forçar Full" :disabled="!podeForcar" @click="confirmarFull">
          <span v-if="forcingType === 'full'" class="spinner-small"></span>
          <span v-else>↺ Full</span>
        </button>
      </div>
    </div>

    <!-- Tabs -->
    <div class="tabs">
      <button
        v-for="t in tabs" :key="t.id"
        :class="['tab', { 'tab--active': abaAtiva === t.id }]"
        @click="mudarAba(t.id)"
      >{{ t.label }}</button>
    </div>

    <!-- Conteúdo das abas -->
    <div class="tab-body">

      <SyncDrawerProgresso
        v-if="abaAtiva === 'progresso'"
        :job="jobAtual"
        :progress="currentJobProgress"
        @inspecionarPayload="(item) => { inspectedProgress = item; showPayloadModal = true }"
      />

      <SyncDrawerHistorico
        v-if="abaAtiva === 'historico'"
        :empresa-id="empresaId"
        :selected-job-id="selectedJobId"
        @selectJob="selectJob"
      />

      <SyncDrawerAgendamento
        v-if="abaAtiva === 'agendamento'"
        :control="controle"
        :executor-configs="executorConfigs"
        :saving="saving"
        @salvarConfig="updateConfig"
      />

      <SyncDrawerConfig
        v-if="abaAtiva === 'config'"
        :control="controle"
        :executor-configs="executorConfigs"
        :saving="saving"
        @salvarConfig="updateConfig"
        @toggleExecutor="updateExecutorConfig"
        @forcarExecutor="(p) => p.tipo === 'full'
          ? confirmarFullExecutor(p.executor)
          : tentarForcar('manual', p.executor)"
      />

    </div>

    <!-- Modal de Job Ativo (409) -->
    <JobAtivoModal
      v-if="conflictJob"
      :job="conflictJob"
      @close="conflictJob = null"
      @viewJob="(id) => { conflictJob = null; selectJob(id) }"
    />

    <!-- Inspetor de Payload -->
    <PayloadInspectorModal
      :visible="showPayloadModal"
      :item="inspectedProgress"
      @close="showPayloadModal = false"
    />

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { useAuthStore } from '@/stores/auth'
import api from '@/api/client'
import SyncDrawerProgresso from '@/components/sync/SyncDrawerProgresso.vue'
import SyncDrawerHistorico from '@/components/sync/SyncDrawerHistorico.vue'
import SyncDrawerConfig from '@/components/sync/SyncDrawerConfig.vue'
import SyncDrawerAgendamento from '@/components/sync/SyncDrawerAgendamento.vue'
import JobAtivoModal from '@/components/sync/JobAtivoModal.vue'
import PayloadInspectorModal from '@/components/sync/PayloadInspectorModal.vue'

// ── Tipos ──────────────────────────────────────────────────────
interface Empresa { id: string; nome: string }
interface SyncJob {
  id: string; empresa_id?: string; tipo: string; status: string
  erro: string; iniciado_at: string | null; concluido_at: string | null; executor?: string
}
interface SyncJobProgress {
  executor: string; status: string; pagina_atual: number; total_paginas: number
  registros_proc: number; registros_total: number; erro: string | null
  iniciado_at: string | null; concluido_at: string | null; updated_at: string
  ultimo_payload?: unknown; ultimo_response?: unknown; erro_payload?: unknown; erro_response?: string
}
interface SyncControl {
  ativo: boolean; intervalo_incremental_min: number; intervalo_full_dias: number
  ultimo_sync_at: string | null; proximo_sync_at: string | null
  ultimo_full_sync_at: string | null; proximo_full_sync_at: string | null
}
interface ExecutorConfig { executor: string; ativo: boolean; notas?: string; updated_at?: string }

// ── Setup ──────────────────────────────────────────────────────
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const empresaId = route.params.empresaId as string
const abaAtiva = ref<'progresso' | 'historico' | 'agendamento' | 'config'>(
  (route.query.aba as string) === 'historico'    ? 'historico'
  : (route.query.aba as string) === 'agendamento' ? 'agendamento'
  : (route.query.aba as string) === 'config'      ? 'config'
  : 'progresso'
)

const tabs = [
  { id: 'progresso',   label: '● PROGRESSO' },
  { id: 'historico',   label: '📋 HISTÓRICO' },
  { id: 'agendamento', label: '🗓 AGENDAMENTO' },
  { id: 'config',      label: '⚙ CONFIG' },
] as const

// ── Estado ─────────────────────────────────────────────────────
const empresa = ref<Empresa | null>(null)
const controle = ref<SyncControl>({
  ativo: true, intervalo_incremental_min: 60, intervalo_full_dias: 7,
  ultimo_sync_at: null, proximo_sync_at: null,
  ultimo_full_sync_at: null, proximo_full_sync_at: null,
})
const jobs = ref<SyncJob[]>([])
const selectedJobId = ref('')
const currentJobProgress = ref<SyncJobProgress[]>([])
const executorConfigs = ref<ExecutorConfig[]>([])
const saving = ref(false)
const forcingType = ref<'manual' | 'full' | ''>('')
const conflictJob = ref<SyncJob | null>(null)
const inspectedProgress = ref<SyncJobProgress | null>(null)
const showPayloadModal = ref(false)
let sseController: AbortController | null = null

// ── Computeds ──────────────────────────────────────────────────
const jobAtual = computed(() =>
  jobs.value.find(j => j.id === selectedJobId.value) ?? jobs.value[0] ?? null
)

const isRunning = computed(() => {
  const s = jobAtual.value?.status
  return s === 'rodando' || s === 'pendente'
})

const podeForcar = computed(() => !forcingType.value && !isRunning.value)

// ── Navegação e abas ───────────────────────────────────────────
function mudarAba(aba: typeof abaAtiva.value) {
  abaAtiva.value = aba
  router.replace({ query: { ...route.query, aba } })
}

// ── Carregamento de dados ──────────────────────────────────────
async function loadEmpresaStatus() {
  try {
    const r = await api.get(`/sync/${empresaId}/status`)
    const data = r.data.data
    if (data?.controle) controle.value = data.controle
    // Extrai nome da empresa do controle se disponível
    if (data?.empresa_nome && !empresa.value) {
      empresa.value = { id: empresaId, nome: data.empresa_nome }
    }
  } catch {}
}

async function loadEmpresa() {
  // Tenta extrair nome de jobs já carregados ou do status
  if (empresa.value) return
  try {
    const r = await api.get(`/sync/${empresaId}/status`)
    const data = r.data.data
    if (data?.empresa_nome) empresa.value = { id: empresaId, nome: data.empresa_nome }
    else empresa.value = { id: empresaId, nome: empresaId }
  } catch {
    empresa.value = { id: empresaId, nome: empresaId }
  }
}

async function loadJobs() {
  try {
    const r = await api.get(`/sync/${empresaId}/jobs?page=1&per_page=10`)
    jobs.value = r.data.data ?? []
    if (jobs.value.length > 0) {
      selectedJobId.value = jobs.value[0].id
      await loadProgress(selectedJobId.value)
    }
  } catch {}
}

async function loadProgress(jobId: string) {
  try {
    const r = await api.get(`/sync/${empresaId}/jobs/${jobId}/progress`)
    currentJobProgress.value = Array.isArray(r.data.data) ? r.data.data : []
  } catch {
    currentJobProgress.value = []
  }
}

async function loadExecutorConfigs() {
  try {
    const r = await api.get(`/sync/${empresaId}/executors`)
    executorConfigs.value = r.data.data ?? []
  } catch {}
}

async function selectJob(jobId: string) {
  selectedJobId.value = jobId
  await loadProgress(jobId)
}

// ── SSE ────────────────────────────────────────────────────────
let sseRetryCount = 0
const SSE_MAX_RETRIES = 5

function openStream() {
  if (sseController) sseController.abort()
  sseController = new AbortController()
  const url = `${import.meta.env.VITE_API_URL || ''}/sync/${empresaId}/stream?token=${auth.accessToken}`
  fetchEventSource(url, {
    signal: sseController.signal,
    onopen(res) {
      if (res.ok) {
        sseRetryCount = 0  // conexão estabelecida — zera o contador de retry
        return
      }
      // 401/403 → não adianta reconectar
      if (res.status === 401 || res.status === 403) {
        sseController?.abort()
        throw new Error(`SSE auth error: ${res.status}`)
      }
    },
    onmessage(msg) {
      if (msg.event === 'heartbeat') return
      try { handleSSEEvent(msg.event, JSON.parse(msg.data)) } catch {}
    },
    onerror() {
      // 401/403 já tratados em onopen — aqui só erros de rede/timeout
      sseRetryCount++
      if (sseRetryCount > SSE_MAX_RETRIES) {
        sseController?.abort()
        throw new Error('SSE max retries reached')
      }
      // Retorna intervalo em ms para o fetchEventSource reconectar automaticamente
      const delay = Math.min(1000 * 2 ** sseRetryCount, 30000) // backoff: 2s, 4s, 8s, 16s, 30s
      return delay
    }
  })
}

function handleSSEEvent(type: string, data: any) {
  if (type === 'job.iniciado') {
    if (!jobs.value.find(j => j.id === data.id)) {
      jobs.value.unshift(data)
      if (jobs.value.length > 10) jobs.value.pop()
    }
    selectedJobId.value = data.id
    currentJobProgress.value = []
    loadEmpresaStatus()
  }
  if (type === 'job.concluido' || type === 'job.erro') {
    const idx = jobs.value.findIndex(j => j.id === data.id)
    if (idx !== -1) jobs.value[idx] = data
    loadEmpresaStatus()
  }
  if (selectedJobId.value === data.job_id) {
    const idx = currentJobProgress.value.findIndex(p => p.executor === data.executor)
    if (type === 'modulo.iniciado') {
      const entry: SyncJobProgress = {
        executor: data.executor, status: 'rodando', pagina_atual: 0, total_paginas: 0,
        registros_proc: 0, registros_total: 0, erro: null,
        iniciado_at: new Date().toISOString(), concluido_at: null, updated_at: new Date().toISOString()
      }
      if (idx === -1) currentJobProgress.value.push(entry)
      else currentJobProgress.value[idx] = { ...currentJobProgress.value[idx], ...entry }
    }
    if (type === 'modulo.progresso' && idx !== -1) {
      currentJobProgress.value[idx] = {
        ...currentJobProgress.value[idx],
        pagina_atual: data.pagina_atual, total_paginas: data.total_paginas,
        registros_proc: data.registros_proc, registros_total: data.registros_total,
        updated_at: new Date().toISOString()
      }
    }
    if (type === 'modulo.concluido' && idx !== -1) {
      currentJobProgress.value[idx] = {
        ...currentJobProgress.value[idx], status: 'concluido',
        registros_total: data.registros_total, concluido_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      }
    }
    if (type === 'modulo.erro' && idx !== -1) {
      currentJobProgress.value[idx] = {
        ...currentJobProgress.value[idx], status: 'erro', erro: data.erro,
        concluido_at: new Date().toISOString(), updated_at: new Date().toISOString()
      }
    }
  }
}

// ── Ações de sync ──────────────────────────────────────────────
async function tentarForcar(tipo: 'manual' | 'full', executor?: string) {
  if (!podeForcar.value) return
  forcingType.value = tipo
  try {
    const r = await api.post(`/sync/${empresaId}/forcar`, { tipo, executor })
    const newJob = r.data.data
    await loadEmpresaStatus()
    jobs.value.unshift(newJob)
    selectedJobId.value = newJob.id
    currentJobProgress.value = []
    mudarAba('progresso')
  } catch (e: any) {
    if (e.response?.status === 409) {
      conflictJob.value = e.response.data.data?.job_ativo
    } else {
      alert(e?.response?.data?.message ?? 'Erro ao iniciar sync')
    }
  } finally {
    forcingType.value = ''
  }
}

function confirmarFull() {
  if (confirm(`Isso irá reprocessar TODOS os registros da empresa ${empresa.value?.nome}.\nPode levar mais de 30 minutos. Confirmar?`)) {
    tentarForcar('full')
  }
}

function confirmarFullExecutor(executor: string) {
  if (confirm(`Isso irá reprocessar TODOS os registros de '${executor}'. Confirmar?`)) {
    tentarForcar('full', executor)
  }
}

async function updateConfig(partial: Partial<SyncControl>) {
  saving.value = true
  try {
    const payload = { ...controle.value, ...partial }
    await api.put(`/sync/${empresaId}/configurar`, payload)
    await loadEmpresaStatus()
  } catch (e: any) {
    alert(e?.response?.data?.message ?? 'Erro ao salvar configuração')
  } finally {
    saving.value = false
  }
}

async function updateExecutorConfig(payload: { executor: string; ativo: boolean; notas: string | null }) {
  try {
    await api.put(`/sync/${empresaId}/executors/${payload.executor}`, {
      ativo: payload.ativo,
      notas: payload.notas,
    })
    await loadExecutorConfigs()
  } catch (err: any) {
    alert(err.response?.data?.message || 'Erro ao atualizar configuração')
  }
}

// ── Lifecycle ──────────────────────────────────────────────────
onMounted(async () => {
  await Promise.all([loadEmpresa(), loadEmpresaStatus(), loadJobs(), loadExecutorConfigs()])
  openStream()
})

onUnmounted(() => {
  if (sseController) { sseController.abort(); sseController = null }
})
</script>

<style scoped>
.empresa-page { padding: 24px; display: flex; flex-direction: column; gap: 0; }

/* Header */
.page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}

.btn-back {
  background: var(--surface-2);
  border: 1px solid var(--border-strong);
  color: var(--text-muted);
  border-radius: 6px;
  padding: 6px 14px;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  cursor: pointer;
  transition: var(--transition);
  white-space: nowrap;
}
.btn-back:hover { border-color: var(--primary); color: var(--primary); }

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}
.bc-root { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim); text-transform: uppercase; letter-spacing: 1px; }
.bc-sep { color: var(--border-strong); }
.bc-current { font-family: var(--font-mono); font-size: var(--fs-sm); font-weight: 700; color: var(--primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.header-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }

.btn-action {
  background: var(--surface-2);
  border: 1px solid var(--border-strong);
  color: var(--text-muted);
  border-radius: 6px;
  padding: 6px 16px;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  font-weight: 700;
  cursor: pointer;
  transition: var(--transition);
  display: flex;
  align-items: center;
  gap: 6px;
}
.btn-action:hover:not(:disabled) { border-color: var(--primary); color: var(--primary); background: var(--primary-weak); }
.btn-action-warn:hover:not(:disabled) { border-color: var(--warning); color: var(--warning); background: var(--warning-weak); }
.btn-action:disabled { opacity: 0.3; cursor: not-allowed; }

/* Tabs */
.tabs {
  display: flex;
  gap: 2px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 0;
}
.tab {
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  padding: 10px 20px;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  font-weight: 700;
  color: var(--text-dim);
  cursor: pointer;
  transition: var(--transition);
  letter-spacing: 0.5px;
  margin-bottom: -1px;
}
.tab:hover { color: var(--text); }
.tab--active { color: var(--primary); border-bottom-color: var(--primary); }

/* Corpo */
.tab-body {
  background: var(--surface);
  border: 1px solid var(--border);
  border-top: none;
  border-radius: 0 0 12px 12px;
  min-height: 400px;
}

/* Badges */
.badge-running {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--primary);
  background: var(--primary-weak);
  padding: 4px 12px;
  border-radius: 20px;
  text-transform: uppercase;
  font-weight: 700;
  display: flex;
  align-items: center;
  gap: 8px;
}

.dot-pulse {
  position: relative;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: var(--primary);
  display: inline-block;
  flex-shrink: 0;
}
.dot-pulse::before {
  content: '';
  position: absolute;
  top: 50%; left: 50%;
  transform: translate(-50%, -50%);
  width: 6px; height: 6px;
  border-radius: 50%;
  border: 1px solid var(--primary);
  animation: pulse-ring 1.5s cubic-bezier(0.455, 0.03, 0.515, 0.955) infinite;
}
@keyframes pulse-ring {
  0% { transform: translate(-50%, -50%) scale(1); opacity: 0.8; }
  100% { transform: translate(-50%, -50%) scale(3); opacity: 0; }
}

.spinner-small {
  width: 12px; height: 12px;
  border: 1.5px solid var(--border-strong);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
  display: inline-block;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
