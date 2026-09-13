<template>
  <div style="padding:24px">
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:16px">
      <div class="section-title" style="margin:0">EMPRESAS</div>
      <button v-if="!auth.isAdminGlobal" class="btn-primary" @click="openCreate">+ Nova Empresa</button>
    </div>

    <div class="table-card">
      
    <!-- Seletor de grupo para admin_global -->
    <div v-if="auth.isAdminGlobal && grupos.length > 0" style="margin-bottom:16px">
      <label style="font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim);letter-spacing:1.5px;text-transform:uppercase;display:block;margin-bottom:6px">GRUPO</label>
      <select v-model="grupoId" class="input-el" style="max-width:320px" @change="load">
        <option value="">Selecione um grupo...</option>
        <option v-for="g in grupos" :key="g.id" :value="g.id">{{ g.nome }}</option>
      </select>
    </div>
<div v-if="loading" style="padding:48px;text-align:center"><div class="spinner"></div></div>
      <div v-else-if="error" style="padding:32px;text-align:center;font-family:var(--font-body);font-size: var(--fs-xs);color:var(--danger)">{{ error }}</div>
      <div v-else-if="empresas.length===0" style="padding:48px;text-align:center;font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim)">Nenhuma empresa cadastrada.</div>
      <div v-else style="overflow-x:auto">
        <table>
          <thead><tr>
            <th>NOME</th><th>CNPJ</th><th>APP KEY</th><th>APP SECRET</th><th>STATUS</th><th>SYNC</th><th style="text-align:right">ACOES</th>
          </tr></thead>
          <tbody>
            <tr v-for="e in empresas" :key="e.id">
              <td style="font-weight:600">{{ e.nome }}</td>
              <td style="font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim)">{{ e.cnpj||"-" }}</td>
              <td style="font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim)">{{ e.app_key }}</td>
              <td style="font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim)">{{ e.app_secret }}</td>
              <td><span :class="['pill', statusCls(e.status)]">{{ e.status }}</span></td>
              <td><span :class="['pill', syncCls(e.status_sync)]">{{ e.status_sync }}</span></td>
              <td style="text-align:right;white-space:nowrap">
                <RouterLink :to="`/sync`" class="btn-ghost" style="margin-right:6px;font-size: var(--fs-xs)">Sync</RouterLink>
                <button class="btn-ghost" @click="openEdit(e)" style="margin-right:6px">Editar</button>
                <button v-if="e.status==='deletando'" class="btn-primary" style="font-size: var(--fs-xs)" @click="reativar(e)">Reativar</button>
                <button v-else class="btn-danger" @click="confirmDelete(e)">Excluir</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal criar/editar -->
    <div v-if="showModal" class="modal-overlay" @mousedown.self="showModal=false">
      <div class="modal-box">
        <div class="modal-header">
          <span style="font-size: var(--fs-md);font-weight:700">{{ editing ? "Editar Empresa" : "Nova Empresa" }}</span>
          <button @click="showModal=false" class="btn-close">x</button>
        </div>
        <div class="modal-body">
          <div class="field"><label>NOME</label><input v-model="form.nome" class="input-el" placeholder="Razao social" /><p v-if="fe.nome" class="err">{{ fe.nome }}</p></div>
          <div class="field"><label>CNPJ</label><input v-model="form.cnpj" class="input-el" placeholder="00.000.000/0001-00" /></div>
          <div class="field"><label>APP KEY</label><input v-model="form.app_key" class="input-el" placeholder="App Key do Omie" /><p v-if="fe.app_key" class="err">{{ fe.app_key }}</p></div>
          <div class="field">
            <label>APP SECRET</label>
            <input v-model="form.app_secret" type="password" class="input-el" :placeholder="editing ? 'Deixe vazio para manter atual' : 'App Secret do Omie'" />
            <p style="font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim);margin-top:4px">O secret nao sera exibido apos salvar.</p>
            <p v-if="fe.app_secret" class="err">{{ fe.app_secret }}</p>
          </div>
          <p v-if="saveErr" class="err-box">{{ saveErr }}</p>
        </div>
        <div class="modal-footer">
          <button class="btn-ghost" @click="showModal=false">Cancelar</button>
          <button class="btn-primary" :disabled="saving" @click="save">{{ saving ? "..." : "Salvar" }}</button>
        </div>
      </div>
    </div>

    <!-- Confirm delete -->
    <div v-if="showConfirm" class="modal-overlay" @mousedown.self="showConfirm=false">
      <div class="modal-box" style="max-width:420px">
        <div class="modal-header"><span style="font-size: var(--fs-md);font-weight:700">Confirmar exclusao</span><button @click="showConfirm=false" class="btn-close">x</button></div>
        <div class="modal-body">
          <p style="font-size: var(--fs-sm);color:var(--text-muted)">Deseja excluir a empresa <strong style="color:var(--text)">{{ delTarget?.nome }}</strong>?</p>
          <p style="font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim);margin-top:8px">A empresa sera marcada para exclusao em 30 dias. Voce pode reativar durante esse periodo.</p>
          <p v-if="deleteErr" class="err-box" style="margin-top:10px">{{ deleteErr }}</p>
        </div>
        <div class="modal-footer">
          <button class="btn-ghost" @click="showConfirm=false">Cancelar</button>
          <button class="btn-danger" :disabled="deleting" @click="doDelete">{{ deleting ? "..." : "Excluir" }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, computed } from "vue"
import { useAuthStore } from "@/stores/auth"
import api from "@/api/client"

interface Empresa { id:string;grupo_id:string;nome:string;cnpj:string;app_key:string;app_secret:string;status:string;status_sync:string;created_at:string }

const auth = useAuthStore()
// admin_global nao tem grupo_id proprio — usa seletor
const grupos = ref<{id:string;nome:string}[]>([])
const grupoId = ref(auth.user?.grupo_id ?? "")

// Garante que grupoId se atualiza quando auth.user chega depois do mount (pós-login)
watch(() => auth.user?.grupo_id, (gid) => {
  if (gid && !auth.isAdminGlobal && !grupoId.value) {
    grupoId.value = gid
    load()
  }
})
async function loadGrupos() {
  if(auth.isAdminGlobal) {
    try { const r=await api.get("/admin/grupos?page=1&per_page=100"); grupos.value=r.data.data??[] }
    catch{}
  }
}

const empresas = ref<Empresa[]>([])
const loading = ref(false)
const error = ref("")
const showModal = ref(false)
const showConfirm = ref(false)
const editing = ref<Empresa|null>(null)
const delTarget = ref<Empresa|null>(null)
const form = ref({ nome:"", cnpj:"", app_key:"", app_secret:"" })
const fe = ref({ nome:"", app_key:"", app_secret:"" })
const saveErr = ref("")
const deleteErr = ref("")
const saving = ref(false)
const deleting = ref(false)

async function load() {
  if (!grupoId.value) return
  loading.value=true; error.value=""
  try {
    const r = await api.get(`/admin/grupos/${grupoId.value}/empresas?page=1&per_page=100`)
    empresas.value = r.data.data ?? []
  } catch(e:any) { error.value = e?.response?.data?.message ?? "Erro ao carregar" }
  finally { loading.value=false }
}

function openCreate() { editing.value=null; form.value={nome:"",cnpj:"",app_key:"",app_secret:""}; fe.value={nome:"",app_key:"",app_secret:""}; saveErr.value=""; showModal.value=true }
function openEdit(e:Empresa) { editing.value=e; form.value={nome:e.nome,cnpj:e.cnpj,app_key:e.app_key,app_secret:""}; fe.value={nome:"",app_key:"",app_secret:""}; saveErr.value=""; showModal.value=true }
function confirmDelete(e:Empresa) { delTarget.value=e; deleteErr.value=""; showConfirm.value=true }

async function save() {
  fe.value={nome:"",app_key:"",app_secret:""}; saveErr.value=""
  if(!form.value.nome.trim()){fe.value.nome="Nome obrigatorio";return}
  if(!form.value.app_key.trim()){fe.value.app_key="App Key obrigatorio";return}
  if(!editing.value && !form.value.app_secret.trim()){fe.value.app_secret="App Secret obrigatorio";return}
  saving.value=true
  try {
    const payload:any = { nome:form.value.nome, cnpj:form.value.cnpj, app_key:form.value.app_key }
    if(form.value.app_secret) payload.app_secret = form.value.app_secret
    if(editing.value) {
      await api.put(`/admin/grupos/${grupoId.value}/empresas/${editing.value.id}`, {...payload, app_secret: form.value.app_secret||editing.value.app_secret})
    } else {
      await api.post(`/admin/grupos/${grupoId.value}/empresas`, payload)
    }
    showModal.value=false; await load()
  } catch(e:any) { saveErr.value = e?.response?.data?.message ?? "Erro ao salvar" }
  finally { saving.value=false }
}

async function doDelete() {
  if(!delTarget.value) return
  deleting.value=true; deleteErr.value=""
  try { await api.delete(`/admin/grupos/${grupoId.value}/empresas/${delTarget.value.id}`); showConfirm.value=false; await load() }
  catch(e:any) { deleteErr.value = e?.response?.data?.message ?? "Erro" }
  finally { deleting.value=false }
}

async function reativar(e:Empresa) {
  try { await api.post(`/admin/grupos/${grupoId.value}/empresas/${e.id}/reativar`); await load() }
  catch(err:any) { alert(err?.response?.data?.message ?? "Erro") }
}

function statusCls(s:string) { return s==="ativa"?"pill-green":s==="deletando"?"pill-red":"pill-gray" }
function syncCls(s:string) { return s==="ativo"?"pill-green":s==="erro"?"pill-red":s==="pausado"?"pill-yellow":"pill-gray" }

onMounted(()=>{ loadGrupos(); if(grupoId.value) load() })
</script>

<style scoped>
.table-card{background:var(--surface);border:1px solid var(--border);border-radius:12px;overflow:hidden}
table{width:100%;border-collapse:collapse}
th{font-family:var(--font-body);font-size: var(--fs-xs);letter-spacing:1.5px;text-transform:uppercase;color:var(--text-dim);padding:11px 18px;text-align:left;background:var(--surface-2);border-bottom:1px solid var(--border)}
td{padding:10px 18px;font-size: var(--fs-sm);color:var(--text);border-bottom:1px solid var(--border)}
tr:last-child td{border-bottom:none}tr:hover td{background:var(--surface-2)}
.pill{display:inline-flex;align-items:center;padding:2px 9px;border-radius:20px;font-family:var(--font-body);font-size: var(--fs-xs);font-weight:600}
.pill-green{background:var(--success-weak);color:var(--success)}.pill-red{background:var(--danger-weak);color:var(--danger)}.pill-yellow{background:var(--warning-weak);color:var(--warning)}.pill-gray{background:var(--surface-2);color:var(--text-dim)}
.btn-primary{background:var(--primary);color:var(--text-oncolor);border:none;border-radius:8px;padding:8px 16px;font-size: var(--fs-sm);font-weight:600;cursor:pointer;transition:var(--transition);text-decoration:none;display:inline-block}
.btn-primary:hover:not(:disabled){background:var(--primary-hover)}.btn-primary:disabled{opacity:0.5;cursor:not-allowed}
.btn-danger{background:var(--danger-weak);color:var(--danger);border:1px solid var(--danger-weak);border-radius:8px;padding:6px 12px;font-size: var(--fs-xs);font-weight:600;cursor:pointer;transition:var(--transition)}
.btn-danger:hover:not(:disabled){background:var(--danger-weak)}.btn-danger:disabled{opacity:0.5;cursor:not-allowed}
.btn-ghost{background:var(--surface-2);color:var(--text-muted);border:1px solid var(--border-strong);border-radius:8px;padding:6px 12px;font-size: var(--fs-xs);font-weight:600;cursor:pointer;transition:var(--transition);text-decoration:none;display:inline-block}
.btn-ghost:hover{border-color:var(--primary);color:var(--primary)}
.btn-close{background:var(--surface-2);color:var(--text-dim);border:1px solid var(--border-strong);border-radius:6px;width:28px;height:28px;cursor:pointer;font-size: var(--fs-base);display:flex;align-items:center;justify-content:center;transition:var(--transition)}
.btn-close:hover{border-color:var(--danger);color:var(--danger)}
.modal-overlay{position:fixed;inset:0;z-index:1000;background:var(--overlay);backdrop-filter:blur(4px);display:flex;align-items:center;justify-content:center;padding:24px}
.modal-box{background:var(--surface);border:1px solid var(--border-strong);border-radius:12px;box-shadow:var(--shadow-md);width:100%;max-width:520px;display:flex;flex-direction:column;max-height:calc(100vh - 48px);overflow:hidden}
.modal-header{display:flex;align-items:center;justify-content:space-between;padding:18px 24px;border-bottom:1px solid var(--border);flex-shrink:0}
.modal-body{padding:24px;display:flex;flex-direction:column;gap:16px;overflow-y:auto}
.modal-footer{padding:16px 24px;border-top:1px solid var(--border);display:flex;justify-content:flex-end;gap:8px;flex-shrink:0}
.field{display:flex;flex-direction:column;gap:6px}
label{font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim);letter-spacing:1.5px;text-transform:uppercase}
.input-el{background:var(--surface-2);border:1px solid var(--border-strong);border-radius:8px;padding:9px 12px;font-size: var(--fs-sm);color:var(--text);outline:none;transition:border-color 0.2s}
.input-el:focus{border-color:var(--primary)}.input-el:disabled{opacity:0.5}
.err{font-family:var(--font-body);font-size: var(--fs-xs);color:var(--danger)}
.err-box{font-family:var(--font-body);font-size: var(--fs-xs);color:var(--danger);background:var(--danger-weak);border:1px solid var(--danger-weak);border-radius:7px;padding:9px 12px}
.spinner{width:24px;height:24px;border:2px solid var(--border-strong);border-top-color:var(--primary);border-radius:50%;animation:spin 0.7s linear infinite;margin:0 auto}
@keyframes spin{to{transform:rotate(360deg)}}
</style>


