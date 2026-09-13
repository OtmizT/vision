<template>
  <div style="padding:24px">
    <div class="section-title">PERMISSOES</div>
    <div style="display:grid;grid-template-columns:280px 1fr;gap:20px;align-items:start" class="perm-grid">
      <!-- Lista de usuarios -->
      <div class="table-card" style="padding:0">
        <div style="padding:14px 16px;border-bottom:1px solid var(--border);font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim);letter-spacing:1.5px">USUARIOS</div>
        <div v-if="loadingU" style="padding:32px;text-align:center"><div class="spinner"></div></div>
        <div v-else>
          <div v-for="u in usuarios" :key="u.id" :class="['user-row', {active: selectedUser?.id===u.id}]" @click="selectUser(u)">
            <div class="u-avatar">{{ initials(u.nome) }}</div>
            <div style="min-width:0">
              <p style="font-size: var(--fs-sm);font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">{{ u.nome }}</p>
              <p style="font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)">{{ u.role }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Permissoes do usuario selecionado -->
      <div>
        <div v-if="!selectedUser" style="background:var(--surface);border:1px solid var(--border);border-radius:12px;padding:48px;text-align:center;font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)">
          Selecione um usuario para gerenciar as permissoes.
        </div>
        <template v-else>
          <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:14px">
            <p style="font-size: var(--fs-base);font-weight:700">{{ selectedUser.nome }}</p>
            <button class="btn-primary" style="font-size: var(--fs-xs)" @click="showGrant=true">+ Conceder</button>
          </div>
          <div class="table-card">
            <div v-if="loadingP" style="padding:32px;text-align:center"><div class="spinner"></div></div>
            <div v-else-if="permissoes.length===0" style="padding:32px;text-align:center;font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)">Nenhuma permissao concedida.</div>
            <div v-else style="overflow-x:auto">
              <table>
                <thead><tr><th>EMPRESA</th><th>RECURSO</th><th>ACAO</th><th>CONCEDIDO EM</th><th style="text-align:right">REVOGAR</th></tr></thead>
                <tbody>
                  <tr v-for="p in permissoes" :key="p.id">
                    <td style="font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)">{{ empresaName(p.empresa_id) }}</td>
                    <td><span class="pill pill-blue">{{ p.recurso }}</span></td>
                    <td><span class="pill pill-gray">{{ p.acao }}</span></td>
                    <td style="font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim)">{{ fmt(p.created_at) }}</td>
                    <td style="text-align:right"><button class="btn-danger" @click="revoke(p)">Revogar</button></td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </template>
      </div>
    </div>

    <!-- Modal grant -->
    <div v-if="showGrant" class="modal-overlay" @mousedown.self="showGrant=false">
      <div class="modal-box" style="max-width:440px">
        <div class="modal-header"><span style="font-size: var(--fs-md);font-weight:700">Conceder Permissao</span><button @click="showGrant=false" class="btn-close">x</button></div>
        <div class="modal-body">
          <div class="field"><label>EMPRESA</label>
            <select v-model="gForm.empresa_id" class="input-el">
              <option value="">Selecione...</option>
              <option v-for="e in empresas" :key="e.id" :value="e.id">{{ e.nome }}</option>
            </select>
          </div>
          <div class="field"><label>RECURSO</label>
            <select v-model="gForm.recurso" class="input-el">
              <option value="dashboard">dashboard</option>
              <option value="sync">sync</option>
              <option value="admin">admin</option>
            </select>
          </div>
          <div class="field"><label>ACAO</label>
            <select v-model="gForm.acao" class="input-el">
              <option value="ver">ver</option>
              <option value="editar">editar</option>
              <option value="forcar_sync">forcar_sync</option>
            </select>
          </div>
          <p v-if="grantErr" class="err-box">{{ grantErr }}</p>
        </div>
        <div class="modal-footer"><button class="btn-ghost" @click="showGrant=false">Cancelar</button><button class="btn-primary" :disabled="granting" @click="doGrant">{{ granting?"...":"Conceder" }}</button></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { useAuthStore } from "@/stores/auth"
import api from "@/api/client"
interface U { id:string; nome:string; role:string }
interface E { id:string; nome:string }
interface P { id:string; usuario_id:string; empresa_id:string; recurso:string; acao:string; created_at:string }
const auth = useAuthStore()
const grupoId = computed(() => auth.user?.grupo_id ?? "")
const usuarios=ref<U[]>([]); const empresas=ref<E[]>([]); const permissoes=ref<P[]>([])
const selectedUser=ref<U|null>(null)
const loadingU=ref(false); const loadingP=ref(false)
const showGrant=ref(false); const granting=ref(false); const grantErr=ref("")
const gForm=ref({ empresa_id:"", recurso:"sync", acao:"ver" })

async function loadUsuarios() {
  if(!grupoId.value) return
  loadingU.value=true
  try { const r=await api.get(`/admin/grupos/${grupoId.value}/usuarios?page=1&per_page=100`); usuarios.value=r.data.data??[] }
  catch{} finally{loadingU.value=false}
}
async function loadEmpresas() {
  if(!grupoId.value) return
  try { const r=await api.get(`/admin/grupos/${grupoId.value}/empresas?page=1&per_page=100`); empresas.value=r.data.data??[] }
  catch{}
}
async function selectUser(u:U) {
  selectedUser.value=u; loadingP.value=true
  try { const r=await api.get(`/admin/permissoes/usuario/${u.id}`); permissoes.value=r.data.data??[] }
  catch{} finally{loadingP.value=false}
}
async function revoke(p:P) {
  try {
    await api.post("/admin/permissoes/revoke",{usuario_id:p.usuario_id,empresa_id:p.empresa_id,recurso:p.recurso,acao:p.acao})
    if(selectedUser.value) await selectUser(selectedUser.value)
  } catch(e:any){ alert(e?.response?.data?.message??"Erro") }
}
async function doGrant() {
  if(!gForm.value.empresa_id){grantErr.value="Selecione uma empresa";return}
  granting.value=true; grantErr.value=""
  try {
    await api.post("/admin/permissoes/grant",{usuario_id:selectedUser.value!.id,...gForm.value})
    showGrant.value=false
    if(selectedUser.value) await selectUser(selectedUser.value)
  } catch(e:any){grantErr.value=e?.response?.data?.message??"Erro"} finally{granting.value=false}
}
function empresaName(id:string){ return empresas.value.find(e=>e.id===id)?.nome ?? id.slice(0,8)+"..." }
function initials(n:string){ return n.split(" ").map(w=>w[0]).slice(0,2).join("").toUpperCase() }
function fmt(d:string){ return d?new Date(d).toLocaleDateString("pt-BR"):"-" }
onMounted(()=>{ loadUsuarios(); loadEmpresas() })
</script>

<style scoped>
.perm-grid{ }
@media(max-width:900px){.perm-grid{grid-template-columns:1fr!important}}
.table-card{background:var(--surface);border:1px solid var(--border);border-radius:12px;overflow:hidden}
table{width:100%;border-collapse:collapse}
th{font-family:var(--font-mono);font-size: var(--fs-xs);letter-spacing:1.5px;text-transform:uppercase;color:var(--text-dim);padding:11px 18px;text-align:left;background:var(--surface-2);border-bottom:1px solid var(--border)}
td{padding:10px 18px;font-size: var(--fs-sm);color:var(--text);border-bottom:1px solid var(--border)}
tr:last-child td{border-bottom:none}tr:hover td{background:var(--surface-2)}
.user-row{display:flex;align-items:center;gap:10px;padding:11px 16px;cursor:pointer;border-bottom:1px solid var(--border);transition:var(--transition)}
.user-row:last-child{border-bottom:none}
.user-row:hover{background:var(--surface-2)}
.user-row.active{background:var(--primary-weak);border-left:2px solid var(--primary)}
.u-avatar{width:28px;height:28px;border-radius:7px;background:linear-gradient(135deg,var(--primary),var(--primary-line));display:flex;align-items:center;justify-content:center;font-size: var(--fs-xs);font-weight:800;color:var(--text-oncolor);flex-shrink:0}
.pill{display:inline-flex;padding:2px 9px;border-radius:20px;font-family:var(--font-mono);font-size: var(--fs-xs);font-weight:600}
.pill-blue{background:var(--primary-weak);color:var(--info)}.pill-gray{background:var(--surface-2);color:var(--text-dim)}
.btn-primary{background:var(--primary);color:var(--text-oncolor);border:none;border-radius:8px;padding:8px 16px;font-size: var(--fs-sm);font-weight:600;cursor:pointer}
.btn-primary:hover:not(:disabled){background:var(--primary-hover)}.btn-primary:disabled{opacity:0.5;cursor:not-allowed}
.btn-danger{background:var(--danger-weak);color:var(--danger);border:1px solid var(--danger-weak);border-radius:6px;padding:4px 10px;font-size: var(--fs-xs);font-weight:600;cursor:pointer}
.btn-ghost{background:var(--surface-2);color:var(--text-muted);border:1px solid var(--border-strong);border-radius:8px;padding:6px 12px;font-size: var(--fs-xs);font-weight:600;cursor:pointer}
.btn-ghost:hover{border-color:var(--primary);color:var(--primary)}
.btn-close{background:var(--surface-2);color:var(--text-dim);border:1px solid var(--border-strong);border-radius:6px;width:28px;height:28px;cursor:pointer;font-size: var(--fs-base);display:flex;align-items:center;justify-content:center}
.btn-close:hover{border-color:var(--danger);color:var(--danger)}
.modal-overlay{position:fixed;inset:0;z-index:1000;background:var(--overlay);backdrop-filter:blur(4px);display:flex;align-items:center;justify-content:center;padding:24px}
.modal-box{background:var(--surface);border:1px solid var(--border-strong);border-radius:12px;box-shadow:var(--shadow-md);width:100%;max-width:520px;display:flex;flex-direction:column}
.modal-header{display:flex;align-items:center;justify-content:space-between;padding:18px 24px;border-bottom:1px solid var(--border)}
.modal-body{padding:24px;display:flex;flex-direction:column;gap:16px}
.modal-footer{padding:16px 24px;border-top:1px solid var(--border);display:flex;justify-content:flex-end;gap:8px}
.field{display:flex;flex-direction:column;gap:6px}
label{font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--text-dim);letter-spacing:1.5px;text-transform:uppercase}
.input-el{background:var(--surface-2);border:1px solid var(--border-strong);border-radius:8px;padding:9px 12px;font-size: var(--fs-sm);color:var(--text);outline:none}
.input-el:focus{border-color:var(--primary)}
.err-box{font-family:var(--font-mono);font-size: var(--fs-xs);color:var(--danger);background:var(--danger-weak);border:1px solid var(--danger-weak);border-radius:7px;padding:9px 12px}
.spinner{width:24px;height:24px;border:2px solid var(--border-strong);border-top-color:var(--primary);border-radius:50%;animation:spin 0.7s linear infinite;margin:0 auto}
@keyframes spin{to{transform:rotate(360deg)}}
</style>
