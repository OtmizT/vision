<template>
  <div style="padding:24px">
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px">
      <div class="section-title" style="margin:0">SINCRONIZACAO — PAINEL DE CONTROLE</div>
      <div v-if="empresasRodando > 0" class="running-bar">
        <span class="dot-pulse"></span>
        {{ empresasRodando }} {{ empresasRodando === 1 ? 'empresa' : 'empresas' }} com sync em andamento
      </div>
    </div>

    <div v-if="!grupoId" style="font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)">Nenhum grupo associado ao seu usuario.</div>
    <template v-else>
      <div class="table-card">
        <div v-if="loading" style="padding:32px;text-align:center"><div class="spinner"></div></div>
        <div v-else-if="empresas.length===0" style="padding:32px;text-align:center;font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)">Nenhuma empresa encontrada.</div>
        <div v-else style="overflow-x:auto">
          <table>
            <thead>
              <tr>
                <th>EMPRESA</th>
                <th>STATUS</th>
                <th>ULTIMO SYNC</th>
                <th>PROX. SYNC</th>
                <th style="text-align:right">ACOES</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="e in empresas" :key="e.id">
                <tr :class="{ 'flash-success': flashingRows[e.id] }">
                  <td class="td-name" @click="router.push(`/sync/${e.id}`)">
                    <div style="display:flex;align-items:center;gap:8px;cursor:pointer">
                      {{ e.nome }}
                    </div>
                  </td>
                  <td>
                    <div style="display:flex;align-items:center;gap:8px">
                      <span v-if="isRunning(e.id)" class="dot-pulse"></span>
                      <span :class="['pill', statusMap[e.id]?.controle?.ativo ? 'pill-green' : 'pill-gray']">
                        {{ statusMap[e.id]?.controle?.ativo ? "Ativo" : "Inativo" }}
                      </span>
                    </div>
                  </td>
                  <td class="td-mono">{{ fmtDate(statusMap[e.id]?.controle?.ultimo_sync_at) }}</td>
                  <td class="td-mono">{{ fmtDate(statusMap[e.id]?.controle?.proximo_sync_at) }}</td>
                  <td style="text-align:right">
                    <div style="display:flex;gap:6px;justify-content:flex-end;align-items:center">
                      <span v-if="statusMap[e.id]?.ultimo_job?.status === 'pendente'" class="badge-pending">Aguardando</span>
                      <span v-else-if="isRunning(e.id)" class="badge-running">Em execução</span>

                      <button class="btn-icon" title="Forçar Incremental" @click="tentarForcar(e.id, 'manual')" :disabled="!podeForcar(e.id)">
                        <span v-if="forcingId === e.id && forcingType === 'manual'" class="spinner-small"></span>
                        <span v-else>▶</span>
                      </button>
                      <button class="btn-icon btn-icon-warn" title="Forçar Full" @click="confirmFull(e)" :disabled="!podeForcar(e.id)">
                        <span v-if="forcingId === e.id && forcingType === 'full'" class="spinner-small"></span>
                        <span v-else>↺</span>
                      </button>
                      <button class="btn-icon" title="Configurações" @click="router.push(`/sync/${e.id}?aba=config`)">
                        ⚙
                      </button>
                    </div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <!-- Modal de Job Ativo (Conflito 409) -->
    <JobAtivoModal
      v-if="conflictJob"
      :job="conflictJob"
      @close="conflictJob = null"
      @viewJob="(id) => { router.push(`/sync/${conflictJobEmpresaId}`); conflictJob = null }"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue"
import { useRouter } from "vue-router"
import { useAuthStore } from "@/stores/auth"
import api from "@/api/client"
import JobAtivoModal from "@/components/sync/JobAtivoModal.vue"

interface Empresa { id: string; nome: string }
interface SyncJob { id: string; tipo: string; status: string; erro: string; iniciado_at: string | null; concluido_at: string | null }
interface SyncControl {
  ativo: boolean; intervalo_incremental_min: number; intervalo_full_dias: number;
  ultimo_sync_at: string | null; proximo_sync_at: string | null;
  ultimo_full_sync_at: string | null; proximo_full_sync_at: string | null;
}
interface SyncStatus { controle: SyncControl; ultimo_job: SyncJob | null }

const router = useRouter()
const auth = useAuthStore()
const grupoId = computed(() => auth.user?.grupo_id ?? "")
const empresas = ref<Empresa[]>([])
const statusMap = ref<Record<string, SyncStatus>>({})
const loading = ref(false)
const forcingId = ref("")
const forcingType = ref<"manual" | "full" | "">("")
const conflictJob = ref<SyncJob | null>(null)
const conflictJobEmpresaId = ref("")
const flashingRows = ref<Record<string, boolean>>({})

let pollTimer: ReturnType<typeof setInterval> | null = null

const empresasRodando = computed(() =>
  Object.values(statusMap.value).filter(s =>
    s.ultimo_job?.status === 'rodando' || s.ultimo_job?.status === 'pendente'
  ).length
)

async function loadData() {
  if (!grupoId.value) return
  loading.value = true
  try {
    const r = await api.get(`/admin/grupos/${grupoId.value}/empresas?page=1&per_page=100`)
    empresas.value = r.data.data ?? []
    await Promise.all(empresas.value.map(e => loadStatus(e.id)))
  } catch {
  } finally {
    loading.value = false
  }
}

async function loadStatus(id: string) {
  try {
    const r = await api.get(`/sync/${id}/status`)
    statusMap.value[id] = r.data.data
  } catch {}
}

function podeForcar(empresaId: string) {
  return forcingId.value !== empresaId && !isRunning(empresaId)
}

function isRunning(id: string) {
  const s = statusMap.value[id]?.ultimo_job?.status
  return s === "rodando" || s === "pendente"
}

function confirmFull(e: Empresa) {
  if (confirm(`Isso irá reprocessar TODOS os registros da empresa ${e.nome}.\nPode levar mais de 30 minutos. Confirmar?`)) {
    tentarForcar(e.id, 'full')
  }
}

async function tentarForcar(id: string, tipo: "manual" | "full") {
  if (!podeForcar(id)) return
  forcingId.value = id
  forcingType.value = tipo
  try {
    await api.post(`/sync/${id}/forcar`, { tipo })
    await loadStatus(id)
    router.push(`/sync/${id}?aba=progresso`)
  } catch (e: any) {
    if (e.response?.status === 409) {
      conflictJob.value = e.response.data.data?.job_ativo
      conflictJobEmpresaId.value = id
    } else {
      alert(e?.response?.data?.message ?? "Erro ao iniciar sync")
    }
  } finally {
    forcingId.value = ""
    forcingType.value = ""
  }
}

function fmtDate(d: string | null) {
  if (!d) return "-"
  return new Date(d).toLocaleString("pt-BR", { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function startPoll() {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = setInterval(() => {
    Promise.all(empresas.value.map(e => loadStatus(e.id)))
  }, 30000)
}

onMounted(() => { loadData(); startPoll() })
onUnmounted(() => { if (pollTimer) clearInterval(pollTimer) })
</script>

<style scoped>
.table-card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; overflow: hidden; margin-top: 16px; }
table { width: 100%; border-collapse: collapse; }
th { font-family: var(--font-mono); font-size: var(--fs-xs); letter-spacing: 1.5px; text-transform: uppercase; color: var(--text-dim); padding: 12px 18px; text-align: left; background: var(--surface-2); border-bottom: 1px solid var(--border); }
td { padding: 12px 18px; font-size: var(--fs-sm); color: var(--text); border-bottom: 1px solid var(--border); }
tr:hover td { background: var(--surface-2); }

.td-name { font-weight: 600; color: var(--primary); }
.td-mono { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-muted); }

.pill { display: inline-flex; padding: 2px 9px; border-radius: 20px; font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: 600; }
.pill-green { background: var(--success-weak); color: var(--success); }
.pill-gray { background: var(--surface-2); color: var(--text-dim); }

.btn-icon { background: var(--surface-2); border: 1px solid var(--border-strong); color: var(--text-muted); border-radius: 6px; width: 32px; height: 32px; display: flex; align-items: center; justify-content: center; cursor: pointer; transition: var(--transition); }
.btn-icon:hover:not(:disabled) { border-color: var(--primary); color: var(--primary); background: var(--primary-weak); }
.btn-icon-warn:hover:not(:disabled) { border-color: var(--warning); color: var(--warning); background: var(--warning-weak); }
.btn-icon:disabled { opacity: 0.3; cursor: not-allowed; }

.spinner { width: 24px; height: 24px; border: 2px solid var(--border-strong); border-top-color: var(--primary); border-radius: 50%; animation: spin 0.7s linear infinite; margin: 0 auto; }
.spinner-small { width: 14px; height: 14px; border: 1.5px solid var(--border-strong); border-top-color: var(--primary); border-radius: 50%; animation: spin 0.7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.running-bar { background: var(--primary-weak); border: 1px solid var(--primary); border-radius: 20px; padding: 4px 16px; font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--primary); display: flex; align-items: center; gap: 10px; }

.dot-pulse { position: relative; width: 6px; height: 6px; border-radius: 50%; background-color: var(--primary); display: inline-block; }
.dot-pulse::before { content: ''; position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 6px; height: 6px; border-radius: 50%; border: 1px solid var(--primary); animation: pulse-ring 1.5s cubic-bezier(0.455, 0.03, 0.515, 0.955) infinite; }
@keyframes pulse-ring { 0% { transform: translate(-50%,-50%) scale(1); opacity: 0.8; } 100% { transform: translate(-50%,-50%) scale(3); opacity: 0; } }

.badge-running { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--primary); background: var(--primary-weak); padding: 2px 8px; border-radius: 4px; margin-right: 8px; text-transform: uppercase; font-weight: 700; }
.badge-pending { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim); background: var(--surface-2); padding: 2px 8px; border-radius: 4px; margin-right: 8px; text-transform: uppercase; }

.flash-success { animation: flash-success-anim 3s ease-out; }
@keyframes flash-success-anim { 0% { background-color: var(--success-weak); } 100% { background-color: transparent; } }

.section-title { font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: 700; color: var(--primary); text-transform: uppercase; letter-spacing: 1px; }
</style>
