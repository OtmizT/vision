<template>
  <div class="drawer-tab-content">
    <div class="section-header">
      <div class="section-title">Histórico de Jobs</div>
      <p class="section-desc">Visualize as execuções passadas e selecione para ver detalhes.</p>
    </div>

    <div v-if="loading && jobs.length === 0" class="loading-state">
      <div class="spinner-small"></div>
      <span>Carregando histórico...</span>
    </div>

    <div v-else-if="jobs.length === 0" class="empty-state">
      <p>Nenhum job registrado para esta empresa.</p>
    </div>

    <div v-else class="jobs-list">
      <div 
        v-for="j in jobs" 
        :key="j.id" 
        class="job-item" 
        :class="{ 'job-item--selected': selectedJobId === j.id }"
        @click="$emit('selectJob', j.id)"
      >
        <div class="job-main">
          <div class="job-meta">
            <span class="job-type td-mono">
              {{ j.tipo }}
              <span v-if="j.executor" class="executor-tag">({{ j.executor }})</span>
            </span>
            <span class="job-date td-mono">{{ fmtDate(j.iniciado_at) }}</span>
          </div>
          <div class="job-status">
            <span :class="['pill-small', statusCls(j.status)]">{{ j.status }}</span>
          </div>
        </div>
        
        <div class="job-footer">
          <span class="job-duration td-mono">⏱ {{ duration(j) }}</span>
          <span v-if="j.erro" class="job-error td-mono" :title="j.erro">{{ j.erro }}</span>
        </div>
      </div>

      <button 
        v-if="hasMore" 
        class="btn-load-more" 
        @click="loadMore" 
        :disabled="loading"
      >
        <span v-if="loading" class="spinner-small"></span>
        <span v-else>CARREGAR MAIS</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import api from "@/api/client"

interface SyncJob {
  id: string
  tipo: string
  status: string
  erro: string
  iniciado_at: string | null
  concluido_at: string | null
  executor?: string
}

const props = defineProps<{
  empresaId: string
  selectedJobId: string
}>()

const emit = defineEmits<{
  (e: 'selectJob', jobId: string): void
}>()

const jobs = ref<SyncJob[]>([])
const loading = ref(false)
const page = ref(1)
const hasMore = ref(true)
const perPage = 10

async function fetchJobs(reset = false) {
  if (reset) {
    page.value = 1
    jobs.value = []
    hasMore.value = true
  }
  
  if (!hasMore.value || loading.value) return

  loading.value = true
  try {
    const r = await api.get(`/sync/${props.empresaId}/jobs?page=${page.value}&per_page=${perPage}`)
    const newJobs = r.data.data ?? []
    
    if (newJobs.length < perPage) {
      hasMore.value = false
    }
    
    jobs.value = [...jobs.value, ...newJobs]
  } catch (err) {
    console.error('Erro ao buscar histórico:', err)
    hasMore.value = false
  } finally {
    loading.value = false
  }
}

function loadMore() {
  page.value++
  fetchJobs()
}

onMounted(() => {
  fetchJobs(true)
})

watch(() => props.empresaId, () => {
  fetchJobs(true)
})

function fmtDate(d: string | null) {
  if (!d) return "-"
  const dt = new Date(d)
  return dt.toLocaleString("pt-BR", { 
    day: '2-digit', 
    month: '2-digit', 
    hour: '2-digit', 
    minute: '2-digit' 
  })
}

function duration(j: SyncJob) {
  if (!j.iniciado_at || !j.concluido_at) return "-"
  const s = Math.round((new Date(j.concluido_at).getTime() - new Date(j.iniciado_at).getTime()) / 1000)
  return s > 60 ? `${Math.floor(s / 60)}m ${s % 60}s` : `${s}s`
}

function statusCls(s: string) {
  switch (s) {
    case 'concluido': return 'st-ok'
    case 'erro': return 'st-erro'
    case 'rodando': return 'st-ativo'
    case 'pendente': return 'st-neutro'
    default: return 'st-neutro'
  }
}
</script>

<style scoped>
.drawer-tab-content { padding: 24px; }

.section-header { margin-bottom: 20px; }
.section-title { font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: 700; color: var(--primary); text-transform: uppercase; letter-spacing: 1px; margin-bottom: 4px; }
.section-desc { font-size: var(--fs-xs); color: var(--text-dim); margin: 0; }

.loading-state, .empty-state { padding: 48px 24px; text-align: center; color: var(--text-dim); font-size: var(--fs-xs); display: flex; flex-direction: column; align-items: center; gap: 12px; }

.jobs-list { display: flex; flex-direction: column; gap: 10px; }

.job-item { 
  background: var(--surface-2); 
  border: 1px solid var(--border); 
  border-radius: 8px; 
  padding: 12px 16px; 
  cursor: pointer; 
  transition: var(--transition);
}
.job-item:hover { background: var(--surface-2); border-color: var(--border-strong); }
.job-item--selected { border-color: var(--primary); background: var(--primary-weak); }

.job-main { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.job-meta { display: flex; flex-direction: column; gap: 2px; }
.job-type { font-size: var(--fs-xs); font-weight: 700; color: var(--text); text-transform: uppercase; }
.executor-tag { color: var(--primary); font-size: var(--fs-xs); margin-left: 4px; }
.job-date { font-size: var(--fs-xs); color: var(--text-dim); }

.job-footer { display: flex; justify-content: space-between; align-items: center; gap: 16px; }
.job-duration { font-size: var(--fs-xs); color: var(--text-muted); }
.job-error { font-size: var(--fs-xs); color: var(--danger); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; flex: 1; text-align: right; }

.btn-load-more { 
  margin-top: 16px; 
  background: var(--surface-2); 
  border: 1px solid var(--border-strong); 
  color: var(--text-muted); 
  border-radius: 6px; 
  padding: 10px; 
  font-family: var(--font-mono); 
  font-size: var(--fs-xs); 
  font-weight: 700; 
  cursor: pointer; 
  transition: var(--transition); 
}
.btn-load-more:hover:not(:disabled) { border-color: var(--primary); color: var(--primary); }
.btn-load-more:disabled { opacity: 0.5; cursor: not-allowed; }

.td-mono { font-family: var(--font-mono); }
.pill-small { font-family: var(--font-mono); font-size: var(--fs-xs); text-transform: uppercase; font-weight: 700; }
.st-ok { color: var(--success); }
.st-erro { color: var(--danger); }
.st-ativo { color: var(--info); }
.st-neutro { color: var(--text-dim); }

.spinner-small { width: 14px; height: 14px; border: 1.5px solid var(--border-strong); border-top-color: var(--primary); border-radius: 50%; animation: spin 0.7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
