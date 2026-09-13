<template>
  <div style="padding:24px">

    <!-- Header -->
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:20px;flex-wrap:wrap;gap:12px">
      <div class="section-title" style="margin:0">SQL EXPLORER</div>

      <!-- Seletor de grupo (admin_global) -->
      <div v-if="auth.isAdminGlobal" style="display:flex;align-items:center;gap:8px">
        <span style="font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)">GRUPO</span>
        <select
          v-model="selectedGrupoId"
          class="select-input"
          style="min-width:180px"
        >
          <option value="">Selecione um grupo...</option>
          <option v-for="g in grupos" :key="g.id" :value="g.id">{{ g.nome }}</option>
        </select>
      </div>
      <div v-else style="font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)">
        {{ grupoNome }}
      </div>
    </div>

    <!-- Aviso sem grupo selecionado -->
    <div
      v-if="!activeGrupoId"
      style="padding:48px;text-align:center;font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)"
    >
      Selecione um grupo para começar.
    </div>

    <template v-else>
      <div class="explorer-layout">

        <!-- Sidebar de tabelas -->
        <div class="tables-sidebar">
          <div class="tables-sidebar-header">TABELAS</div>
          <div
            v-for="t in TABLES"
            :key="t"
            class="table-item"
            @click="insertTableQuery(t)"
            :title="`SELECT * FROM ${t} LIMIT 100`"
          >
            <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5" style="width:13px;height:13px;flex-shrink:0;color:var(--primary)">
              <rect x="2" y="4" width="16" height="3" rx="1"/>
              <rect x="2" y="9" width="16" height="3" rx="1"/>
              <rect x="2" y="14" width="16" height="3" rx="1"/>
            </svg>
            <span>{{ t }}</span>
          </div>
        </div>

        <!-- Painel principal -->
        <div class="main-panel">

          <!-- Editor de query -->
          <div class="editor-wrapper">
            <textarea
              ref="editorRef"
              v-model="sql"
              class="sql-editor"
              placeholder="SELECT * FROM clientes LIMIT 100"
              spellcheck="false"
              autocomplete="off"
              autocorrect="off"
              autocapitalize="off"
              @keydown="onKeydown"
            />
            <div class="editor-footer">
              <span style="font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)">
                Ctrl+Enter para executar
              </span>
              <button
                class="btn-primary"
                style="padding:6px 18px;font-size: var(--fs-xs)"
                :disabled="executing || !sql.trim()"
                @click="runQuery"
              >
                <span v-if="executing" class="spinner" style="width:14px;height:14px;display:inline-block;vertical-align:middle;margin-right:6px" />
                {{ executing ? 'Executando...' : 'Executar' }}
              </button>
            </div>
          </div>

          <!-- Estado de erro da API -->
          <div
            v-if="queryError"
            style="margin-top:12px;padding:12px 14px;background:var(--danger-weak);border:1px solid var(--danger-weak);border-radius:8px;font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--danger)"
          >
            {{ queryError }}
          </div>

          <!-- Resultado -->
          <div v-if="result" class="result-panel">
            <div style="display:flex;align-items:center;gap:10px;margin-bottom:10px;flex-wrap:wrap">
              <span style="font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)">
                {{ result.row_count }} {{ result.row_count === 1 ? 'linha' : 'linhas' }}
                &nbsp;·&nbsp;
                {{ elapsed }}ms
              </span>
              <span
                v-if="result.truncated"
                style="font-family:var(--font-mono);font-size: var(--fs-xs);padding:2px 8px;border-radius:4px;background:var(--warning-weak);color:var(--warning);border:1px solid var(--warning-weak)"
              >
                Truncado em 1000 linhas
              </span>
            </div>

            <div v-if="result.columns.length === 0" style="font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim);padding:24px 0;text-align:center">
              Nenhum resultado.
            </div>

            <div v-else style="overflow-x:auto">
              <table class="result-table">
                <thead>
                  <tr>
                    <th v-for="col in result.columns" :key="col">{{ col }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, ri) in result.rows" :key="ri">
                    <td v-for="(cell, ci) in row" :key="ci">{{ formatCell(cell) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Estado vazio inicial -->
          <div
            v-else-if="!executing && !queryError"
            style="padding:48px;text-align:center;font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)"
          >
            Execute uma query para ver os resultados.
          </div>

        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { gruposApi, type Grupo } from '@/api/grupos'
import { queryApi, type QueryResult } from '@/api/query'

const auth = useAuthStore()

// ── Tabelas disponíveis ──────────────────────────────────────
// Os nomes precisam bater com o que o provisionador cria em
// internal/db/provisioner.go. Aqui constavam 'contas_a_pagar' e
// 'contas_a_receber', que nao existem: as tabelas sao contas_pagar e
// contas_receber. Clicar nelas montava consulta contra relacao inexistente.
const TABLES = [
  'clientes',
  'categorias',
  'departamentos',
  'contas_correntes',
  'contas_pagar',
  'contas_receber',
  'extrato',
  'movimentos_financeiros',
  'ordens_servico',
  'projetos',
]

// ── Grupos (admin_global) ────────────────────────────────────
const grupos         = ref<Grupo[]>([])
const selectedGrupoId = ref('')

const activeGrupoId = computed(() => {
  if (auth.isAdminGlobal) return selectedGrupoId.value
  return auth.user?.grupo_id ?? ''
})

const grupoNome = computed(() => {
  if (auth.isAdminGlobal) {
    return grupos.value.find(g => g.id === activeGrupoId.value)?.nome ?? ''
  }
  return auth.user?.grupo_id ?? ''
})

async function loadGrupos() {
  if (!auth.isAdminGlobal) return
  try {
    const r = await gruposApi.list({ per_page: 200 })
    grupos.value = r.data.data ?? []
  } catch {
    // silencioso — admin_grupo não precisa da lista
  }
}

// ── Editor ───────────────────────────────────────────────────
const sql       = ref('')
const editorRef = ref<HTMLTextAreaElement | null>(null)

function insertTableQuery(table: string) {
  sql.value = `SELECT * FROM ${table} LIMIT 100`
  editorRef.value?.focus()
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault()
    runQuery()
  }
}

// ── Execução ─────────────────────────────────────────────────
const executing  = ref(false)
const queryError = ref('')
const result     = ref<QueryResult | null>(null)
const elapsed    = ref(0)

async function runQuery() {
  const gid = activeGrupoId.value
  if (!gid || !sql.value.trim() || executing.value) return

  executing.value  = true
  queryError.value = ''
  result.value     = null

  const t0 = performance.now()
  try {
    const r = await queryApi.execute(gid, sql.value.trim())
    elapsed.value = Math.round(performance.now() - t0)
    result.value  = r.data.data
  } catch (e: unknown) {
    elapsed.value    = Math.round(performance.now() - t0)
    const axErr = e as { response?: { data?: { message?: string } } }
    queryError.value = axErr?.response?.data?.message ?? 'Erro ao executar a query.'
  } finally {
    executing.value = false
  }
}

// ── Helpers ──────────────────────────────────────────────────
function formatCell(v: unknown): string {
  if (v === null || v === undefined) return 'NULL'
  if (typeof v === 'object') return JSON.stringify(v)
  return String(v)
}

// Limpa resultado ao trocar de grupo
watch(activeGrupoId, () => {
  result.value     = null
  queryError.value = ''
})

onMounted(loadGrupos)
</script>

<style scoped>
.explorer-layout {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}

/* ── Sidebar de tabelas ── */
.tables-sidebar {
  width: 200px;
  flex-shrink: 0;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
  position: sticky;
  top: 80px;
}

.tables-sidebar-header {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  letter-spacing: 2px;
  color: var(--text-dim);
  padding: 10px 12px 6px;
  border-bottom: 1px solid var(--border);
}

.table-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-muted);
  cursor: pointer;
  transition: var(--transition);
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.table-item:last-child { border-bottom: none; }
.table-item:hover { background: var(--surface-2); color: var(--primary); }

/* ── Painel principal ── */
.main-panel {
  flex: 1;
  min-width: 0;
}

/* ── Editor SQL ── */
.editor-wrapper {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
}

.sql-editor {
  width: 100%;
  min-height: 140px;
  padding: 16px;
  background: transparent;
  color: var(--text);
  font-family: var(--font-mono);
  font-size: var(--fs-sm);
  line-height: 1.7;
  border: none;
  outline: none;
  resize: vertical;
  box-sizing: border-box;
  caret-color: var(--primary);
}
.sql-editor::placeholder { color: var(--text-dim); }

.editor-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-top: 1px solid var(--border);
  background: var(--surface-2);
}

/* ── Seletor de grupo ── */
.select-input {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  padding: 6px 10px;
  outline: none;
  transition: var(--transition);
}
.select-input:focus { border-color: var(--primary); }

/* ── Resultado ── */
.result-panel {
  margin-top: 16px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 16px;
}

.result-table {
  width: 100%;
  border-collapse: collapse;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
}

.result-table th {
  text-align: left;
  padding: 8px 12px;
  font-size: var(--fs-xs);
  letter-spacing: 1px;
  color: var(--text-dim);
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
  background: var(--surface-2);
}

.result-table td {
  padding: 7px 12px;
  color: var(--text);
  border-bottom: 1px solid var(--border);
  max-width: 340px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.result-table tbody tr:nth-child(even) td { background: var(--surface-2); }
.result-table tbody tr:hover td { background: var(--primary-weak); }

@media (max-width: 640px) {
  .explorer-layout { flex-direction: column; }
  .tables-sidebar  { width: 100%; position: static; }
}
</style>
