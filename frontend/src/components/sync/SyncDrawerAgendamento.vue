<template>
  <div class="agend-content">

    <!-- Status geral -->
    <div class="card-section">
      <div class="section-label">STATUS DO AGENDAMENTO</div>
      <div class="status-row">
        <div class="status-info">
          <span :class="['status-badge', localAtivo ? 'status-badge--on' : 'status-badge--off']">
            <span class="status-dot"></span>
            {{ localAtivo ? 'Agendamento ativo' : 'Agendamento pausado' }}
          </span>
          <span class="status-hint">
            {{ localAtivo ? 'Jobs automáticos serão criados conforme os intervalos configurados.' : 'Nenhum job automático será criado enquanto pausado.' }}
          </span>
        </div>
        <label class="switch">
          <input type="checkbox" v-model="localAtivo">
          <span class="slider"></span>
        </label>
      </div>
    </div>

    <!-- Próximos syncs -->
    <div class="grid-2">

      <!-- Incremental -->
      <div class="card-section">
        <div class="section-label">PRÓXIMO INCREMENTAL</div>
        <div class="next-sync-block">
          <div v-if="props.control.proximo_sync_at" class="next-datetime">
            {{ fmtDateTime(props.control.proximo_sync_at) }}
          </div>
          <div v-else class="next-datetime next-datetime--none">Não agendado</div>

          <div v-if="countdownIncremental" :class="['countdown', countdownIncremental === 'Agora' ? 'countdown--now' : '']">
            {{ countdownIncremental }}
          </div>

          <div class="last-sync" v-if="props.control.ultimo_sync_at">
            Último: {{ fmtDateTime(props.control.ultimo_sync_at) }}
          </div>
        </div>

        <div class="interval-row">
          <span class="interval-label">Intervalo</span>
          <select v-model="localIntervaloMin" class="select-sm">
            <option :value="60">1 hora</option>
            <option :value="120">2 horas</option>
            <option :value="240">4 horas</option>
            <option :value="720">12 horas</option>
          </select>
        </div>
      </div>

      <!-- Full -->
      <div class="card-section">
        <div class="section-label">PRÓXIMO FULL</div>
        <div class="next-sync-block">
          <div v-if="props.control.proximo_full_sync_at" class="next-datetime">
            {{ fmtDateTime(props.control.proximo_full_sync_at) }}
          </div>
          <div v-else class="next-datetime next-datetime--none">Não agendado</div>

          <div v-if="countdownFull" :class="['countdown', countdownFull === 'Agora' ? 'countdown--now' : '']">
            {{ countdownFull }}
          </div>

          <div class="last-sync" v-if="props.control.ultimo_full_sync_at">
            Último: {{ fmtDateTime(props.control.ultimo_full_sync_at) }}
          </div>
        </div>

        <div class="interval-row">
          <span class="interval-label">Intervalo</span>
          <select v-model="localIntervaloFullDias" class="select-sm">
            <option :value="5">5 dias</option>
            <option :value="7">7 dias</option>
            <option :value="15">15 dias</option>
          </select>
        </div>
      </div>

    </div>

    <!-- Módulos do próximo sync -->
    <div class="card-section">
      <div class="section-label">MÓDULOS NO PRÓXIMO SYNC</div>
      <p class="section-desc">Executores que serão processados automaticamente.</p>

      <div v-if="executorConfigs.length === 0" class="empty-executors">
        Nenhum módulo configurado
      </div>
      <div v-else class="executors-grid">
        <div
          v-for="cfg in executorConfigs"
          :key="cfg.executor"
          :class="['executor-chip', cfg.ativo ? 'executor-chip--on' : 'executor-chip--off']"
        >
          <span class="chip-dot"></span>
          <span class="chip-name">{{ cfg.executor }}</span>
          <span class="chip-status">{{ cfg.ativo ? 'ativo' : 'desativado' }}</span>
        </div>
      </div>
    </div>

    <!-- Salvar -->
    <div class="save-row">
      <button class="btn-save" @click="handleSave" :disabled="saving">
        <span v-if="saving" class="spinner-small"></span>
        <span v-else>SALVAR AGENDAMENTO</span>
      </button>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'

interface SyncControl {
  ativo: boolean
  intervalo_incremental_min: number
  intervalo_full_dias: number
  ultimo_sync_at: string | null
  proximo_sync_at: string | null
  ultimo_full_sync_at: string | null
  proximo_full_sync_at: string | null
}

interface ExecutorConfig {
  executor: string
  ativo: boolean
  notas?: string
}

const props = defineProps<{
  control: SyncControl
  executorConfigs: ExecutorConfig[]
  saving: boolean
}>()

const emit = defineEmits<{
  (e: 'salvarConfig', config: { ativo: boolean; intervalo_incremental_min: number; intervalo_full_dias: number }): void
}>()

// Estado local
const localAtivo = ref(props.control.ativo)
const localIntervaloMin = ref(props.control.intervalo_incremental_min)
const localIntervaloFullDias = ref(props.control.intervalo_full_dias)

watch(() => props.control, (v) => {
  localAtivo.value = v.ativo
  localIntervaloMin.value = v.intervalo_incremental_min
  localIntervaloFullDias.value = v.intervalo_full_dias
}, { deep: true })

// Countdown
const now = ref(Date.now())
let timer: ReturnType<typeof setInterval>

onMounted(() => { timer = setInterval(() => { now.value = Date.now() }, 1000) })
onUnmounted(() => clearInterval(timer))

function buildCountdown(target: string | null): string {
  if (!target) return ''
  const diff = new Date(target).getTime() - now.value
  if (diff <= 0) return 'Agora'
  const h = Math.floor(diff / 3600000)
  const m = Math.floor((diff % 3600000) / 60000)
  const s = Math.floor((diff % 60000) / 1000)
  if (h > 0) return `em ${h}h ${String(m).padStart(2,'0')}m ${String(s).padStart(2,'0')}s`
  if (m > 0) return `em ${m}m ${String(s).padStart(2,'0')}s`
  return `em ${s}s`
}

const countdownIncremental = computed(() => buildCountdown(props.control.proximo_sync_at))
const countdownFull        = computed(() => buildCountdown(props.control.proximo_full_sync_at))

function fmtDateTime(d: string | null) {
  if (!d) return '—'
  return new Date(d).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function handleSave() {
  emit('salvarConfig', {
    ativo: localAtivo.value,
    intervalo_incremental_min: localIntervaloMin.value,
    intervalo_full_dias: localIntervaloFullDias.value,
  })
}
</script>

<style scoped>
.agend-content { padding: 24px; display: flex; flex-direction: column; gap: 20px; }

/* Seções */
.card-section {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: var(--r);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-label {
  font-family: var(--font-body);
  font-size: var(--fs-xs);
  font-weight: 700;
  color: var(--primary);
  letter-spacing: 1.5px;
}
.section-desc { font-size: var(--fs-xs); color: var(--text-dim); margin: -8px 0 0; }

/* Status row */
.status-row { display: flex; justify-content: space-between; align-items: center; gap: 16px; }
.status-info { display: flex; flex-direction: column; gap: 6px; }

.status-badge {
  display: inline-flex; align-items: center; gap: 6px;
  font-family: var(--font-body); font-size: var(--fs-xs); font-weight: 600;
  padding: 4px 10px; border-radius: 999px;
}
.status-badge--on  { background: var(--success-weak);  color: var(--success); }
.status-badge--off { background: var(--surface-2); color: var(--text-dim); }

.status-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: currentColor; flex-shrink: 0;
}
.status-hint { font-size: var(--fs-xs); color: var(--text-dim); }

/* Grid 2 colunas */
.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
@media (max-width: 767px) { .grid-2 { grid-template-columns: 1fr; } }

/* Próximo sync block */
.next-sync-block { display: flex; flex-direction: column; gap: 6px; }

.next-datetime {
  font-size: var(--fs-lg); font-weight: 700; color: var(--text);
  font-variant-numeric: tabular-nums;
}
.next-datetime--none { color: var(--text-dim); font-size: var(--fs-base); font-weight: 400; }

.countdown {
  font-family: var(--font-body);
  font-size: var(--fs-sm);
  color: var(--primary);
  font-variant-numeric: tabular-nums;
}
.countdown--now { color: var(--success); }

.last-sync { font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text-dim); }

/* Interval row */
.interval-row {
  display: flex; align-items: center; justify-content: space-between;
  padding-top: 12px;
  border-top: 1px solid var(--border);
}
.interval-label { font-size: var(--fs-xs); font-weight: 500; color: var(--text-muted); }
.select-sm {
  background: var(--surface-2); border: 1px solid var(--border-strong);
  border-radius: 6px; padding: 5px 10px;
  font-size: var(--fs-xs); color: var(--text); outline: none; cursor: pointer;
  transition: border-color 0.2s;
}
.select-sm:hover { border-color: var(--primary); }

/* Executores */
.empty-executors { font-size: var(--fs-xs); color: var(--text-dim); }

.executors-grid {
  display: flex; flex-wrap: wrap; gap: 8px;
}

.executor-chip {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 6px 12px; border-radius: 8px;
  font-family: var(--font-body); font-size: var(--fs-xs); font-weight: 500;
  border: 1px solid transparent;
}
.executor-chip--on  {
  background: var(--primary-weak);
  border-color: var(--primary);
  color: var(--primary);
}
.executor-chip--off {
  background: var(--surface-2);
  border-color: var(--border);
  color: var(--text-dim);
  opacity: 0.6;
}

.chip-dot {
  width: 5px; height: 5px; border-radius: 50%;
  background: currentColor; flex-shrink: 0;
}
.chip-name { flex: 1; }
.chip-status {
  font-size: var(--fs-xs); opacity: 0.7;
  padding: 1px 5px; border-radius: 4px;
  background: var(--surface-2);
}

/* Salvar */
.save-row { display: flex; justify-content: flex-end; }
.btn-save {
  background: var(--surface-2); border: 1px solid var(--border-strong);
  color: var(--primary); border-radius: var(--r-sm);
  padding: 10px 24px; font-family: var(--font-body);
  font-size: var(--fs-xs); font-weight: 700; cursor: pointer;
  transition: var(--transition); display: flex; align-items: center; gap: 8px;
}
.btn-save:hover:not(:disabled) { background: var(--primary); color: var(--bg); border-color: var(--primary); }
.btn-save:disabled { opacity: 0.5; cursor: not-allowed; }

/* Switch */
.switch { position: relative; display: inline-block; width: 36px; height: 18px; flex-shrink: 0; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: var(--surface-2); border: 1px solid var(--border-strong); transition: .3s; border-radius: 20px; }
.slider:before { position: absolute; content: ""; height: 12px; width: 12px; left: 2px; bottom: 2px; background-color: var(--text-dim); transition: .3s; border-radius: 50%; }
input:checked + .slider { background-color: var(--primary); border-color: var(--primary); }
input:checked + .slider:before { transform: translateX(18px); background-color: var(--primary); }

.spinner-small { width: 12px; height: 12px; border: 1.5px solid var(--border-strong); border-top-color: var(--primary); border-radius: 50%; animation: spin 0.7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
