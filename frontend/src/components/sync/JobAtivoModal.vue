<template>
  <div class="modal-overlay" @click="$emit('close')">
    <div class="modal-card" @click.stop>
      <div class="section-title" style="color:var(--primary)">SINCRONISMO EM ANDAMENTO</div>
      
      <div class="info-box">
        <div class="info-row">
          <span class="label">ID DO JOB:</span>
          <span class="value td-mono">{{ job.id.split('-')[0] }}...</span>
        </div>
        <div class="info-row">
          <span class="label">TIPO:</span>
          <span class="value td-mono">{{ job.tipo.toUpperCase() }}</span>
        </div>
        <div class="info-row">
          <span class="label">STATUS:</span>
          <span class="value">
            <span :class="['pill', statusCls]">{{ job.status.toUpperCase() }}</span>
          </span>
        </div>
        <div class="info-row">
          <span class="label">INICIADO EM:</span>
          <span class="value td-mono">{{ fmtDate(job.iniciado_at) }}</span>
        </div>
      </div>

      <p class="message">
        Já existe um processo de sincronização ativo para esta empresa. 
        Aguarde a conclusão ou acompanhe o progresso detalhado.
      </p>

      <div class="modal-actions">
        <button class="btn-inline" @click="$emit('close')">FECHAR</button>
        <button class="btn-inline btn-primary" @click="$emit('viewJob', job.id)">
          IR PARA O JOB EM ANDAMENTO
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  job: {
    id: string
    tipo: string
    status: string
    iniciado_at: string | null
  }
}>()

defineEmits(['close', 'viewJob'])

const statusCls = computed(() => {
  switch (props.job.status) {
    case 'rodando': return 'pill-blue'
    case 'pendente': return 'pill-gray'
    case 'erro': return 'pill-red'
    case 'concluido': return 'pill-green'
    default: return 'pill-gray'
  }
})

function fmtDate(d: string | null) {
  if (!d) return "-"
  const dt = new Date(d)
  return dt.toLocaleString("pt-BR", { 
    day: '2-digit', 
    month: '2-digit', 
    hour: '2-digit', 
    minute: '2-digit',
    second: '2-digit'
  })
}
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0; background: var(--overlay);
  backdrop-filter: blur(4px); z-index: 2000;
  display: flex; align-items: center; justify-content: center;
}
.modal-card {
  width: 450px; background: var(--bg); border: 1px solid var(--border);
  border-radius: 12px; padding: 24px; box-shadow: 0 20px 40px rgba(0,0,0,0.5);
}
.info-box {
  background: var(--surface-2); border: 1px solid var(--border-strong);
  border-radius: 8px; padding: 16px; margin: 16px 0;
}
.info-row {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 8px;
}
.info-row:last-child { margin-bottom: 0; }
.label { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim); }
.value { font-size: var(--fs-sm); color: var(--text); }
.td-mono { font-family: var(--font-mono); font-size: var(--fs-xs); }

.message {
  font-size: var(--fs-xs); color: var(--text-muted); line-height: 1.6;
  margin-bottom: 24px;
}

.modal-actions {
  display: flex; gap: 12px; justify-content: flex-end;
}

.btn-primary {
  background: var(--primary) !important;
  color: var(--bg) !important;
  font-weight: 700;
}

.pill { display: inline-flex; padding: 2px 9px; border-radius: 20px; font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: 600; }
.pill-green { background: var(--success-weak); color: var(--success); }
.pill-red { background: var(--danger-weak); color: var(--danger); }
.pill-blue { background: var(--primary-weak); color: var(--info); }
.pill-gray { background: var(--surface-2); color: var(--text-dim); }

.section-title {
  font-family: var(--font-mono); font-size: var(--fs-xs); letter-spacing: 1.5px;
  text-transform: uppercase; margin-bottom: 16px;
}

.btn-inline {
  background: var(--surface-2); border: 1px solid var(--border-strong);
  color: var(--text); border-radius: 4px; padding: 8px 16px;
  font-size: var(--fs-xs); cursor: pointer; transition: var(--transition);
}
.btn-inline:hover { border-color: var(--primary); color: var(--primary); }
</style>
