<template>
  <div style="padding:24px">
    <div v-if="successMsg" style="margin-bottom:16px;padding:12px 16px;background:var(--primary-weak);border:1px solid var(--primary);border-radius:8px;font-size: var(--fs-sm);color:var(--primary)">
      {{ successMsg }}
    </div>
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:16px">
      <div class="section-title" style="margin:0">USUARIOS</div>
      <button class="btn-primary" @click="openCreate">+ Novo Usuario</button>
    </div>
    <div class="table-card">
      
    <!-- Seletor de grupo para admin_global -->
    <div v-if="auth.isAdminGlobal && grupos.length > 0" style="margin-bottom:16px">
      <label style="font-family:var(--font-display);font-size: var(--fs-xs);color:var(--text-dim);letter-spacing:1.5px;text-transform:uppercase;display:block;margin-bottom:6px">GRUPO</label>
      <select v-model="grupoId" class="input-el" style="max-width:320px" @change="load">
        <option value="">Selecione um grupo...</option>
        <option v-for="g in grupos" :key="g.id" :value="g.id">{{ g.nome }}</option>
      </select>
    </div>
<div v-if="loading" style="padding:48px;text-align:center"><div class="spinner"></div></div>
      <div v-else-if="error" style="padding:32px;text-align:center;font-family:var(--font-display);font-size: var(--fs-xs);color:var(--danger)">{{ error }}</div>
      <div v-else-if="usuarios.length===0" style="padding:48px;text-align:center;font-family:var(--font-display);font-size: var(--fs-xs);color:var(--text-dim)">Nenhum usuario cadastrado.</div>
      <div v-else style="overflow-x:auto">
        <table>
          <thead><tr><th>NOME</th><th>EMAIL</th><th>ROLE</th><th>ATIVO</th><th>CRIADO EM</th><th style="text-align:right">ACOES</th></tr></thead>
          <tbody>
            <tr v-for="u in usuarios" :key="u.id">
              <td style="font-weight:600">{{ u.nome }}</td>
              <td style="font-family:var(--font-display);font-size: var(--fs-xs);color:var(--text-dim)">{{ u.email }}</td>
              <td><span :class="['pill', roleCls(u.role)]">{{ rotuloPapel(u.role) }}</span></td>
              <td><span :class="['pill', u.ativo?'pill-green':'pill-gray']">{{ u.ativo?"Ativo":"Inativo" }}</span></td>
              <td style="font-family:var(--font-display);font-size: var(--fs-xs);color:var(--text-dim)">{{ fmt(u.created_at) }}</td>
              <td style="text-align:right;white-space:nowrap">
                <button class="btn-ghost" @click="openPwd(u)" style="margin-right:6px;font-size: var(--fs-xs)">Senha</button>
                <button class="btn-ghost" @click="openEdit(u)" style="margin-right:6px">Editar</button>
                <button class="btn-danger" @click="confirmDel(u)">Excluir</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal criar/editar -->
    <div v-if="showModal" class="modal-overlay" @mousedown.self="showModal=false">
      <div class="modal-box">
        <div class="modal-header"><span style="font-size: var(--fs-md);font-weight:700">{{ editing?"Editar Usuario":"Novo Usuario" }}</span><button @click="showModal=false" class="btn-close">x</button></div>
        <div class="modal-body">
          <div class="field"><label>NOME</label><input v-model="form.nome" class="input-el" placeholder="Nome completo" /><p v-if="fe.nome" class="err">{{ fe.nome }}</p></div>
          <div v-if="!editing" class="field"><label>EMAIL</label><input v-model="form.email" class="input-el" placeholder="email@empresa.com" /><p v-if="fe.email" class="err">{{ fe.email }}</p></div>
          <div v-if="!editing" class="field"><label>SENHA</label><input v-model="form.password" type="password" class="input-el" placeholder="Minimo 8 caracteres" /><p v-if="fe.password" class="err">{{ fe.password }}</p></div>
          <div class="field">
            <label>ROLE</label>
            <select v-model="form.role" class="input-el">
              <option v-for="r in roles" :key="r.value" :value="r.value">{{ r.label }}</option>
            </select>
          </div>
          <div v-if="editing" class="field" style="flex-direction:row;align-items:center;gap:10px">
            <input type="checkbox" v-model="form.ativo" id="ativo" style="width:16px;height:16px;accent-color:var(--primary)" />
            <label for="ativo" style="text-transform:none;letter-spacing:0;font-size: var(--fs-sm);color:var(--text-muted)">Usuario ativo</label>
          </div>
          <p v-if="saveErr" class="err-box">{{ saveErr }}</p>
        </div>
        <div class="modal-footer"><button class="btn-ghost" @click="showModal=false">Cancelar</button><button class="btn-primary" :disabled="saving" @click="save">{{ saving?"...":"Salvar" }}</button></div>
      </div>
    </div>

    <!-- Modal senha -->
    <div v-if="showPwd" class="modal-overlay" @mousedown.self="showPwd=false">
      <div class="modal-box" style="max-width:420px">
        <div class="modal-header"><span style="font-size: var(--fs-md);font-weight:700">Alterar Senha</span><button @click="showPwd=false" class="btn-close">x</button></div>
        <div class="modal-body">
          <div class="field"><label>NOVA SENHA</label><input v-model="pwd.p1" type="password" class="input-el" placeholder="Minimo 8 caracteres" /></div>
          <div class="field"><label>CONFIRMAR SENHA</label><input v-model="pwd.p2" type="password" class="input-el" placeholder="Repita a senha" /></div>
          <p v-if="pwdErr" class="err-box">{{ pwdErr }}</p>
        </div>
        <div class="modal-footer"><button class="btn-ghost" @click="showPwd=false">Cancelar</button><button class="btn-primary" :disabled="savingPwd" @click="savePwd">{{ savingPwd?"...":"Salvar" }}</button></div>
      </div>
    </div>

    <!-- Confirm delete -->
    <div v-if="showConfirm" class="modal-overlay" @mousedown.self="showConfirm=false">
      <div class="modal-box" style="max-width:420px">
        <div class="modal-header"><span style="font-size: var(--fs-md);font-weight:700">Confirmar exclusao</span><button @click="showConfirm=false" class="btn-close">x</button></div>
        <div class="modal-body">
          <p style="font-size: var(--fs-sm);color:var(--text-muted)">Excluir usuario <strong style="color:var(--text)">{{ delTarget?.nome }}</strong>?</p>
          <p v-if="delErr" class="err-box" style="margin-top:10px">{{ delErr }}</p>
        </div>
        <div class="modal-footer"><button class="btn-ghost" @click="showConfirm=false">Cancelar</button><button class="btn-danger" :disabled="deleting" @click="doDelete">{{ deleting?"...":"Excluir" }}</button></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { useAuthStore } from "@/stores/auth"
import api from "@/api/client"
import { papeisAtribuiveis, rotuloPapel } from "@/utils/papeis"
interface Usuario { id:string;nome:string;email:string;role:string;ativo:boolean;created_at:string }
const auth = useAuthStore()
const grupos = ref<{id:string;nome:string}[]>([])
const grupoId = ref(auth.user?.grupo_id ?? "")
async function loadGrupos() {
  if(auth.isAdminGlobal) {
    try { const r=await api.get("/admin/grupos?page=1&per_page=100"); grupos.value=r.data.data??[] }
    catch{}
  }
}
const usuarios = ref<Usuario[]>([])
const loading=ref(false); const error=ref("")
const showModal=ref(false); const showPwd=ref(false); const showConfirm=ref(false)
const editing=ref<Usuario|null>(null); const pwdTarget=ref<Usuario|null>(null); const delTarget=ref<Usuario|null>(null)
const form=ref({nome:"",email:"",password:"",role:"viewer",ativo:true})
const fe=ref({nome:"",email:"",password:""})
const saveErr=ref(""); const pwdErr=ref(""); const delErr=ref("")
const saving=ref(false); const savingPwd=ref(false); const deleting=ref(false)
const successMsg=ref("")
const pwd=ref({p1:"",p2:""})
// Admin Global saiu do seletor: era conceder, por uma rota de grupo, o papel que
// manda na plataforma inteira. Ver utils/papeis.ts.
const roles = computed(() => papeisAtribuiveis())
async function load() {
  if(!grupoId.value) return
  loading.value=true; error.value=""
  try { const r=await api.get(`/admin/grupos/${grupoId.value}/usuarios?page=1&per_page=100`); usuarios.value=r.data.data??[] }
  catch(e:any){error.value=e?.response?.data?.message??"Erro"}
  finally{loading.value=false}
}
function openCreate(){editing.value=null;form.value={nome:"",email:"",password:"",role:"viewer",ativo:true};fe.value={nome:"",email:"",password:""};saveErr.value="";showModal.value=true}
function openEdit(u:Usuario){editing.value=u;form.value={nome:u.nome,email:u.email,password:"",role:u.role,ativo:u.ativo};fe.value={nome:"",email:"",password:""};saveErr.value="";showModal.value=true}
function openPwd(u:Usuario){pwdTarget.value=u;pwd.value={p1:"",p2:""};pwdErr.value="";showPwd.value=true}
function confirmDel(u:Usuario){delTarget.value=u;delErr.value="";showConfirm.value=true}
async function save() {
  fe.value={nome:"",email:"",password:""}; saveErr.value=""; successMsg.value=""
  if(!form.value.nome.trim()){fe.value.nome="Obrigatorio";return}
  if(!editing.value && !form.value.email.trim()){fe.value.email="Obrigatorio";return}
  if(!editing.value && form.value.password.length<8){fe.value.password="Minimo 8 caracteres";return}
  saving.value=true
  try {
    if(editing.value) {
      await api.put(`/admin/grupos/${grupoId.value}/usuarios/${editing.value.id}`,{nome:form.value.nome,role:form.value.role,ativo:form.value.ativo})
      showModal.value=false; await load()
    } else {
      const r = await api.post(`/admin/grupos/${grupoId.value}/usuarios`,{nome:form.value.nome,email:form.value.email,password:form.value.password,role:form.value.role})
      showModal.value=false; await load()
      if(r.data?.data?.added_to_group) {
        successMsg.value="Usuário já existia no sistema e foi adicionado a este grupo. Role e senha não foram alterados."
        setTimeout(()=>successMsg.value="",6000)
      }
    }
  } catch(e:any){saveErr.value=e?.response?.data?.message??"Erro"} finally{saving.value=false}
}
async function savePwd() {
  pwdErr.value=""
  if(pwd.value.p1.length<8){pwdErr.value="Minimo 8 caracteres";return}
  if(pwd.value.p1!==pwd.value.p2){pwdErr.value="Senhas nao conferem";return}
  savingPwd.value=true
  try { await api.put(`/admin/grupos/${grupoId.value}/usuarios/${pwdTarget.value!.id}/password`,{password:pwd.value.p1}); showPwd.value=false }
  catch(e:any){pwdErr.value=e?.response?.data?.message??"Erro"} finally{savingPwd.value=false}
}
async function doDelete() {
  if(!delTarget.value) return
  deleting.value=true; delErr.value=""
  try { await api.delete(`/admin/grupos/${grupoId.value}/usuarios/${delTarget.value.id}`); showConfirm.value=false; await load() }
  catch(e:any){delErr.value=e?.response?.data?.message??"Erro"} finally{deleting.value=false}
}
function roleCls(r:string){return r==="admin_global"?"pill-accent":r==="admin_grupo"?"pill-blue":"pill-gray"}
function fmt(d:string){return d?new Date(d).toLocaleDateString("pt-BR"):"-"}
onMounted(()=>{ loadGrupos(); if(grupoId.value) load() })
</script>

<style scoped>
.table-card{background:var(--surface);border:1px solid var(--border);border-radius:12px;overflow:hidden}
table{width:100%;border-collapse:collapse}
th{font-family:var(--font-display);font-size: var(--fs-xs);letter-spacing:1.5px;text-transform:uppercase;color:var(--text-dim);padding:11px 18px;text-align:left;background:var(--surface-2);border-bottom:1px solid var(--border)}
td{padding:10px 18px;font-size: var(--fs-sm);color:var(--text);border-bottom:1px solid var(--border)}
tr:last-child td{border-bottom:none}tr:hover td{background:var(--surface-2)}
.pill{display:inline-flex;padding:2px 9px;border-radius:20px;font-family:var(--font-display);font-size: var(--fs-xs);font-weight:600}
.pill-green{background:var(--success-weak);color:var(--success)}.pill-gray{background:var(--surface-2);color:var(--text-dim)}.pill-blue{background:var(--primary-weak);color:var(--info)}.pill-accent{background:var(--primary-weak);color:var(--primary)}
.btn-primary{background:var(--primary);color:var(--text-oncolor);border:none;border-radius:8px;padding:8px 16px;font-size: var(--fs-sm);font-weight:600;cursor:pointer;transition:var(--transition)}
.btn-primary:hover:not(:disabled){background:var(--primary-hover)}.btn-primary:disabled{opacity:0.5;cursor:not-allowed}
.btn-danger{background:var(--danger-weak);color:var(--danger);border:1px solid var(--danger-weak);border-radius:8px;padding:6px 12px;font-size: var(--fs-xs);font-weight:600;cursor:pointer}
.btn-danger:hover:not(:disabled){background:var(--danger-weak)}.btn-danger:disabled{opacity:0.5;cursor:not-allowed}
.btn-ghost{background:var(--surface-2);color:var(--text-muted);border:1px solid var(--border-strong);border-radius:8px;padding:6px 12px;font-size: var(--fs-xs);font-weight:600;cursor:pointer;transition:var(--transition)}
.btn-ghost:hover{border-color:var(--primary);color:var(--primary)}
.btn-close{background:var(--surface-2);color:var(--text-dim);border:1px solid var(--border-strong);border-radius:6px;width:28px;height:28px;cursor:pointer;font-size: var(--fs-base);display:flex;align-items:center;justify-content:center}
.btn-close:hover{border-color:var(--danger);color:var(--danger)}
.modal-overlay{position:fixed;inset:0;z-index:1000;background:var(--overlay);backdrop-filter:blur(4px);display:flex;align-items:center;justify-content:center;padding:24px}
.modal-box{background:var(--surface);border:1px solid var(--border-strong);border-radius:12px;box-shadow:var(--shadow-md);width:100%;max-width:520px;display:flex;flex-direction:column;max-height:calc(100vh - 48px);overflow:hidden}
.modal-header{display:flex;align-items:center;justify-content:space-between;padding:18px 24px;border-bottom:1px solid var(--border);flex-shrink:0}
.modal-body{padding:24px;display:flex;flex-direction:column;gap:16px;overflow-y:auto}
.modal-footer{padding:16px 24px;border-top:1px solid var(--border);display:flex;justify-content:flex-end;gap:8px;flex-shrink:0}
.field{display:flex;flex-direction:column;gap:6px}
label{font-family:var(--font-display);font-size: var(--fs-xs);color:var(--text-dim);letter-spacing:1.5px;text-transform:uppercase}
.input-el{background:var(--surface-2);border:1px solid var(--border-strong);border-radius:8px;padding:9px 12px;font-size: var(--fs-sm);color:var(--text);outline:none;transition:border-color 0.2s}
.input-el:focus{border-color:var(--primary)}
.err{font-family:var(--font-display);font-size: var(--fs-xs);color:var(--danger)}
.err-box{font-family:var(--font-display);font-size: var(--fs-xs);color:var(--danger);background:var(--danger-weak);border:1px solid var(--danger-weak);border-radius:7px;padding:9px 12px}
.spinner{width:24px;height:24px;border:2px solid var(--border-strong);border-top-color:var(--primary);border-radius:50%;animation:spin 0.7s linear infinite;margin:0 auto}
@keyframes spin{to{transform:rotate(360deg)}}
</style>


