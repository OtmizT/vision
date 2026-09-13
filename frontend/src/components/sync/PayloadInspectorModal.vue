<template>
  <div v-if="visible" class="modal-overlay" @click="$emit('close')">
    <div class="modal-card" @click.stop>
      <div class="modal-header">
        <div class="header-info">
          <span class="header-label">INSPETOR DE PAYLOAD</span>
          <h2 class="header-title">{{ item?.executor }}</h2>
        </div>
        <button class="btn-close" @click="$emit('close')">✕</button>
      </div>

      <div class="modal-tabs">
        <button 
          :class="['tab-item', { active: activeTab === 'request' }]" 
          @click="activeTab = 'request'"
        >
          REQUEST
        </button>
        <button 
          :class="['tab-item', { active: activeTab === 'response' }]" 
          @click="activeTab = 'response'"
        >
          RESPONSE
        </button>
      </div>

      <div class="modal-body">
        <div v-if="activeTab === 'request'" class="tab-pane">
          <div v-if="requestData" class="pane-content">
            <div class="pane-header">
              <span class="pane-label">JSON ENVIADO (SANITIZADO)</span>
              <button class="btn-copy" @click="copyToClipboard(requestData)">COPIAR JSON</button>
            </div>
            <pre class="json-block">{{ requestData }}</pre>
          </div>
          <div v-else class="empty-state">Nenhum payload registrado para este módulo.</div>
        </div>

        <div v-if="activeTab === 'response'" class="tab-pane">
          <div v-if="responseData" class="pane-content">
            <div class="pane-header">
              <span class="pane-label">METADADOS DA RESPOSTA</span>
              <button class="btn-copy" @click="copyToClipboard(responseData)">COPIAR JSON</button>
            </div>
            <pre class="json-block">{{ responseData }}</pre>
          </div>
          <div v-else class="empty-state">Nenhuma resposta registrada para este módulo.</div>
        </div>
      </div>

      <div class="modal-footer">
        <div class="security-note">
          <span class="icon">🛡</span>
          Campos sensíveis (secrets/tokens) são automaticamente ocultados no frontend.
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'

interface SyncJobProgress {
  executor: string;
  status: string;
  ultimo_payload?: any;
  ultimo_response?: any;
  erro_payload?: any;
  erro_response?: string;
}

const props = defineProps<{
  visible: boolean
  item: SyncJobProgress | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const activeTab = ref<'request' | 'response'>('request')

/**
 * Função de segurança crítica: remove recursivamente qualquer campo sensível.
 */
function sanitizePayload(obj: unknown): unknown {
  if (obj === null || typeof obj !== 'object') {
    return obj
  }

  if (Array.isArray(obj)) {
    return obj.map(sanitizePayload)
  }

  const sanitized: Record<string, unknown> = {}
  const sensitiveKeys = ['app_secret', 'appsecret', 'secret', 'token']

  for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
    const lowerKey = key.toLowerCase()
    const reallySensitive = sensitiveKeys.some(sk => lowerKey.indexOf(sk) !== -1)

    if (reallySensitive) {
      sanitized[key] = "***REDACTED***"
    } else if (typeof value === 'object') {
      sanitized[key] = sanitizePayload(value)
    } else {
      sanitized[key] = value
    }
  }

  return sanitized
}

const requestData = computed(() => {
  if (!props.item) return null
  const raw = props.item.status === 'erro' ? props.item.erro_payload : props.item.ultimo_payload
  if (!raw) return null
  return JSON.stringify(sanitizePayload(raw), null, 2)
})

const responseData = computed(() => {
  if (!props.item) return null
  const raw = props.item.status === 'erro' ? props.item.erro_response : props.item.ultimo_response
  if (!raw) return null
  
  // Se for string (erro_response costuma ser), tenta parsear para sanitizar se for JSON
  if (typeof raw === 'string') {
    try {
      const parsed = JSON.parse(raw)
      return JSON.stringify(sanitizePayload(parsed), null, 2)
    } catch {
      return raw // string pura (ex: erro de rede)
    }
  }
  
  return JSON.stringify(sanitizePayload(raw), null, 2)
})

async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    alert('Copiado para o clipboard!')
  } catch (err) {
    console.error('Erro ao copiar:', err)
  }
}

function handleEsc(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.visible) {
    emit('close')
  }
}

onMounted(() => window.addEventListener('keydown', handleEsc))
onUnmounted(() => window.removeEventListener('keydown', handleEsc))

// Reset tab ao abrir para novo item
watch(() => props.item, () => { activeTab.value = 'request' })
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0; background: var(--overlay);
  backdrop-filter: blur(6px); z-index: 2000;
  display: flex; align-items: center; justify-content: center;
}
.modal-card {
  width: 800px; max-width: 90vw; height: 80vh;
  background: var(--bg); border: 1px solid var(--border);
  border-radius: 12px; display: flex; flex-direction: column;
  box-shadow: 0 30px 60px rgba(0,0,0,0.6);
  animation: modal-in 0.2s ease-out;
}
@keyframes modal-in { from { opacity: 0; transform: scale(0.95); } to { opacity: 1; transform: scale(1); } }

.modal-header {
  padding: 24px; border-bottom: 1px solid var(--border);
  display: flex; justify-content: space-between; align-items: center;
}
.header-label { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim); letter-spacing: 1px; }
.header-title { margin: 4px 0 0 0; font-size: var(--fs-lg); color: var(--primary); }

.btn-close {
  background: var(--surface-2); border: 1px solid var(--border-strong); color: var(--text-muted);
  border-radius: 6px; width: 32px; height: 32px; cursor: pointer;
}

.modal-tabs {
  display: flex; gap: 24px; padding: 0 24px; border-bottom: 1px solid var(--border);
}
.tab-item {
  background: transparent; border: none; padding: 16px 0;
  font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: 700; color: var(--text-dim);
  cursor: pointer; position: relative;
}
.tab-item.active { color: var(--primary); }
.tab-item.active::after {
  content: ''; position: absolute; bottom: -1px; left: 0; right: 0;
  height: 2px; background: var(--primary);
}

.modal-body { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.tab-pane { flex: 1; display: flex; flex-direction: column; padding: 24px; }
.pane-content { flex: 1; display: flex; flex-direction: column; overflow: hidden; }

.pane-header {
  display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;
}
.pane-label { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim); }

.btn-copy {
  background: var(--surface-2); border: 1px solid var(--border-strong); color: var(--primary);
  border-radius: 4px; padding: 4px 12px; font-size: var(--fs-xs); font-weight: 700; cursor: pointer;
}

.json-block {
  flex: 1; background: var(--surface-2); border: 1px solid var(--border-strong);
  border-radius: 8px; padding: 20px; font-family: var(--font-mono);
  font-size: var(--fs-xs); color: var(--text-muted); overflow: auto;
  line-height: 1.5;
}

.empty-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--text-dim); font-size: var(--fs-sm); font-style: italic;
}

.modal-footer {
  padding: 16px 24px; border-top: 1px solid var(--border);
  background: var(--surface-2);
}
.security-note {
  display: flex; align-items: center; gap: 8px;
  font-size: var(--fs-xs); color: var(--text-dim);
}
</style>
