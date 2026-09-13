<template>
  <div style="padding:24px">
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:16px">
      <div class="section-title" style="margin:0">GRUPOS / TENANTS</div>
      <button class="btn-primary" @click="openCreate">+ Novo Grupo</button>
    </div>

    <div class="table-card">
      <div v-if="loading" style="padding:48px;text-align:center">
        <div class="spinner"></div>
      </div>
      <div v-else-if="error" style="padding:32px;text-align:center;font-family:var(--font-body);font-size: var(--fs-xs);color:var(--danger)">{{ error }}</div>
      <div v-else-if="grupos.length === 0" style="padding:48px;text-align:center;font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim)">Nenhum grupo cadastrado.</div>
      <div v-else style="overflow-x:auto">
        <table>
          <thead><tr>
            <th>NOME</th><th>SLUG</th><th>SCHEMA</th><th>STATUS</th><th>CRIADO EM</th><th style="text-align:right">ACOES</th>
          </tr></thead>
          <tbody>
            <tr v-for="g in grupos" :key="g.id">
              <td style="font-weight:600">{{ g.nome }}</td>
              <td style="font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim)">{{ g.slug }}</td>
              <td style="font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim)">{{ g.schema_name }}</td>
              <td><span :class="['pill', g.status === 'ativo' ? 'pill-green' : 'pill-gray']">{{ g.status }}</span></td>
              <td style="font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim)">{{ fmtDate(g.created_at) }}</td>
              <td style="text-align:right">
                <button class="btn-ghost" @click="openEdit(g)" style="margin-right:6px">Editar</button>
                <button class="btn-danger" @click="confirmDelete(g)">Excluir</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal criar/editar -->
    <AppModal
      v-model="showModal"
      :title="editing ? 'Editar Grupo' : 'Novo Grupo'"
      :subtitle="!editing ? `Passo ${currentStep} de 3` : ''"
      :size="currentStep === 3 ? 'lg' : 'md'"
    >
      <div v-if="editing || currentStep === 1" class="wizard-step">
        <AppInput
          v-model="form.nome"
          label="NOME DO GRUPO"
          placeholder="Ex: Grupo Alpha"
          :error="formErr.nome"
          @input="autoSlug"
        />
        <AppInput
          v-model="form.slug"
          label="SLUG"
          placeholder="grupo-alpha"
          :disabled="!!editing"
          :error="formErr.slug"
          hint="Apenas letras minúsculas, números e hifens. Imutável após criação."
        />
      </div>

      <div v-else-if="currentStep === 2" class="wizard-step">
        <div class="step-info">
          <p class="step-title">Administrador do Grupo</p>
          <p class="step-sub">Crie o usuário responsável por gerenciar este grupo.</p>
        </div>
        <AppInput
          v-model="form.adminNome"
          label="NOME COMPLETO"
          placeholder="Ex: João Silva"
          :error="formErr.adminNome"
        />
        <AppInput
          v-model="form.adminEmail"
          label="E-MAIL"
          type="email"
          placeholder="joao@email.com"
          :error="formErr.adminEmail"
        />
        <AppInput
          v-model="form.adminPassword"
          label="SENHA"
          type="password"
          placeholder="Mínimo 8 caracteres"
          :error="formErr.adminPassword"
        />
      </div>

      <div v-else-if="currentStep === 3" class="wizard-step">
        <div class="step-info">
          <p class="step-title">Primeira Empresa (Opcional)</p>
          <p class="step-sub">Configure a primeira empresa para iniciar a sincronização.</p>
        </div>
        <div style="display:grid;grid-template-columns:1fr 1fr;gap:16px">
          <AppInput
            v-model="form.empresaNome"
            label="NOME DA EMPRESA"
            placeholder="Ex: Alpha LTDA"
          />
          <AppInput
            v-model="form.empresaCnpj"
            label="CNPJ (OPCIONAL)"
            placeholder="00.000.000/0000-00"
          />
        </div>
        <div style="display:grid;grid-template-columns:1fr 1fr;gap:16px">
          <AppInput
            v-model="form.empresaAppKey"
            label="APP KEY (OMIE)"
            placeholder="Chave de acesso"
          />
          <AppInput
            v-model="form.empresaAppSecret"
            label="APP SECRET (OMIE)"
            type="password"
            placeholder="Segredo de acesso"
            hint="Não será exibido após salvar"
          />
        </div>
      </div>

      <p v-if="saveErr" class="err-box">{{ saveErr }}</p>

      <template #footer>
        <div style="display:flex;justify-content:space-between;width:100%">
          <div>
            <AppButton variant="ghost" @click="showModal = false">Cancelar</AppButton>
          </div>
          <div style="display:flex;gap:8px">
            <AppButton v-if="!editing && currentStep > 1" variant="secondary" @click="prevStep">Anterior</AppButton>
            
            <AppButton v-if="editing" :loading="saving" @click="save">Salvar Alterações</AppButton>
            <AppButton v-else-if="currentStep < 3" @click="nextStep">Próximo</AppButton>
            <template v-else>
              <AppButton variant="ghost" @click="save" :disabled="saving">Pular — adicionar depois</AppButton>
              <AppButton :loading="saving" @click="save">Finalizar Wizard</AppButton>
            </template>
          </div>
        </div>
      </template>
    </AppModal>

    <!-- Confirm delete -->
    <AppModal
      v-model="showConfirm"
      title="Confirmar exclusão"
      size="sm"
    >
      <p style="font-size: var(--fs-sm);color:var(--text-muted)">Deseja excluir o grupo <strong style="color:var(--text)">{{ delTarget?.nome }}</strong>?</p>
      <p style="font-family:var(--font-body);font-size: var(--fs-xs);color:var(--text-dim);margin-top:8px">Todas as empresas devem estar inativas para prosseguir.</p>
      <p v-if="deleteErr" class="err-box" style="margin-top:10px">{{ deleteErr }}</p>

      <template #footer>
        <AppButton variant="ghost" @click="showConfirm = false">Cancelar</AppButton>
        <AppButton variant="danger" :loading="deleting" @click="doDelete">Excluir</AppButton>
      </template>
    </AppModal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from "vue"
import api from "@/api/client"
import AppModal from "@/components/ui/AppModal.vue"
import AppInput from "@/components/ui/AppInput.vue"
import AppButton from "@/components/ui/AppButton.vue"

interface Grupo { id:string; nome:string; slug:string; schema_name:string; status:string; created_at:string }

const grupos = ref<Grupo[]>([])
const loading = ref(false)
const error = ref("")
const showModal = ref(false)
const showConfirm = ref(false)
const editing = ref<Grupo|null>(null)
const delTarget = ref<Grupo|null>(null)

// Wizard State
const currentStep = ref(1)
const form = ref({
  // Passo 1
  nome: "",
  slug: "",
  // Passo 2
  adminNome: "",
  adminEmail: "",
  adminPassword: "",
  // Passo 3
  empresaNome: "",
  empresaCnpj: "",
  empresaAppKey: "",
  empresaAppSecret: ""
})

const formErr = ref({
  nome: "",
  slug: "",
  adminNome: "",
  adminEmail: "",
  adminPassword: "",
  empresaNome: "",
  empresaAppKey: "",
  empresaAppSecret: ""
})

const saveErr = ref("")
const deleteErr = ref("")
const saving = ref(false)
const deleting = ref(false)

async function load() {
  loading.value = true; error.value = ""
  try {
    const r = await api.get("/admin/grupos?page=1&per_page=100")
    grupos.value = r.data.data ?? []
  } catch(e:any) { error.value = e?.response?.data?.message ?? "Erro ao carregar" }
  finally { loading.value = false }
}

function openCreate() {
  editing.value = null
  currentStep.value = 1
  form.value = {
    nome: "", slug: "",
    adminNome: "", adminEmail: "", adminPassword: "",
    empresaNome: "", empresaCnpj: "", empresaAppKey: "", empresaAppSecret: ""
  }
  formErr.value = {
    nome: "", slug: "",
    adminNome: "", adminEmail: "", adminPassword: "",
    empresaNome: "", empresaAppKey: "", empresaAppSecret: ""
  }
  saveErr.value = ""
  showModal.value = true
}

function openEdit(g:Grupo) {
  editing.value = g
  currentStep.value = 1
  form.value.nome = g.nome
  form.value.slug = g.slug
  formErr.value = {
    nome: "", slug: "",
    adminNome: "", adminEmail: "", adminPassword: "",
    empresaNome: "", empresaAppKey: "", empresaAppSecret: ""
  }
  saveErr.value = ""
  showModal.value = true
}

function confirmDelete(g:Grupo) { delTarget.value=g; deleteErr.value=""; showConfirm.value=true }

function autoSlug() {
  if (!editing.value) {
    form.value.slug = form.value.nome.toLowerCase().normalize("NFD").replace(/[̀-ͯ]/g,"").replace(/[^a-z0-9]+/g,"-").replace(/^-|-$/g,"")
  }
}

function nextStep() {
  if (currentStep.value === 1) {
    formErr.value.nome = ""
    formErr.value.slug = ""
    if (!form.value.nome.trim()) { formErr.value.nome = "Nome obrigatório"; return }
    if (!form.value.slug.trim()) { formErr.value.slug = "Slug obrigatório"; return }
    currentStep.value = 2
  } else if (currentStep.value === 2) {
    formErr.value.adminNome = ""
    formErr.value.adminEmail = ""
    formErr.value.adminPassword = ""
    if (!form.value.adminNome.trim()) { formErr.value.adminNome = "Nome do administrador obrigatório"; return }
    if (!form.value.adminEmail.trim()) { formErr.value.adminEmail = "E-mail obrigatório"; return }
    if (form.value.adminPassword.length < 8) { formErr.value.adminPassword = "Senha deve ter no mínimo 8 caracteres"; return }
    currentStep.value = 3
  }
}

function prevStep() {
  if (currentStep.value > 1) currentStep.value--
}

async function save() {
  saveErr.value = ""

  if (editing.value) {
    if (!form.value.nome.trim()) { formErr.value.nome = "Nome obrigatório"; return }
    saving.value = true
    try {
      await api.put(`/admin/grupos/${editing.value.id}`, { nome: form.value.nome })
      showModal.value = false
      await load()
    } catch(e:any) {
      saveErr.value = e?.response?.data?.message ?? "Erro ao salvar"
    } finally {
      saving.value = false
    }
    return
  }

  // Wizard Save (Step 3 or Skip)
  saving.value = true
  let grupoId = ""

  try {
    // 1. Criar Grupo
    const resGrupo = await api.post("/admin/grupos", {
      nome: form.value.nome,
      slug: form.value.slug
    })
    grupoId = resGrupo.data.data.id

    // 2. Criar Admin
    try {
      await api.post(`/admin/grupos/${grupoId}/usuarios`, {
        nome: form.value.adminNome,
        email: form.value.adminEmail,
        password: form.value.adminPassword,
        role: "admin_grupo"
      })
    } catch (errAdmin) {
      alert("Grupo criado, mas houve erro ao criar o administrador. Acesse Usuários para completar.")
      showModal.value = false
      await load()
      return
    }

    // 3. Criar Empresa (se preenchido)
    if (form.value.empresaNome.trim()) {
      try {
        await api.post(`/admin/grupos/${grupoId}/empresas`, {
          nome: form.value.empresaNome,
          cnpj: form.value.empresaCnpj,
          app_key: form.value.empresaAppKey,
          app_secret: form.value.empresaAppSecret
        })
      } catch (errEmpresa) {
        alert("Grupo e administrador criados. Houve erro ao criar a empresa. Acesse Empresas para adicionar.")
        showModal.value = false
        await load()
        return
      }
    }

    showModal.value = false
    await load()
  } catch(e:any) {
    saveErr.value = e?.response?.data?.message ?? "Erro ao criar grupo"
  } finally {
    saving.value = false
  }
}

async function doDelete() {
  if (!delTarget.value) return
  deleting.value = true; deleteErr.value = ""
  try {
    await api.delete(`/admin/grupos/${delTarget.value.id}`)
    showConfirm.value = false
    await load()
  } catch(e:any) { deleteErr.value = e?.response?.data?.message ?? "Erro ao excluir" }
  finally { deleting.value = false }
}

function fmtDate(d:string) { return d ? new Date(d).toLocaleDateString("pt-BR") : "-" }
onMounted(load)
</script>

<style scoped>
.table-card{background:var(--surface);border:1px solid var(--border);border-radius:12px;overflow:hidden}
table{width:100%;border-collapse:collapse}
th{font-family:var(--font-body);font-size: var(--fs-xs);letter-spacing:1.5px;text-transform:uppercase;color:var(--text-dim);padding:11px 18px;text-align:left;background:var(--surface-2);border-bottom:1px solid var(--border)}
td{padding:10px 18px;font-size: var(--fs-sm);color:var(--text);border-bottom:1px solid var(--border)}
tr:last-child td{border-bottom:none}
tr:hover td{background:var(--surface-2)}
.pill{display:inline-flex;align-items:center;padding:2px 9px;border-radius:20px;font-family:var(--font-body);font-size: var(--fs-xs);font-weight:600}
.pill-green{background:var(--success-weak);color:var(--success)}
.pill-gray{background:var(--surface-2);color:var(--text-dim)}

.btn-primary{background:var(--primary);color:var(--text-oncolor);border:none;border-radius:8px;padding:8px 16px;font-size: var(--fs-sm);font-weight:600;cursor:pointer;transition:var(--transition)}
.btn-primary:hover:not(:disabled){background:var(--primary-hover)}

.btn-danger{background:var(--danger-weak);color:var(--danger);border:1px solid var(--danger-weak);border-radius:8px;padding:6px 12px;font-size: var(--fs-xs);font-weight:600;cursor:pointer;transition:var(--transition)}
.btn-danger:hover:not(:disabled){background:var(--danger-weak)}

.btn-ghost{background:var(--surface-2);color:var(--text-muted);border:1px solid var(--border-strong);border-radius:8px;padding:6px 12px;font-size: var(--fs-xs);font-weight:600;cursor:pointer;transition:var(--transition)}
.btn-ghost:hover{border-color:var(--primary);color:var(--primary)}

.wizard-step { display: flex; flex-direction: column; gap: 16px; }
.step-info { margin-bottom: 8px; }
.step-title { font-size: var(--fs-base); font-weight: 700; color: var(--text); }
.step-sub { font-size: var(--fs-xs); color: var(--text-dim); margin-top: 2px; }

.err-box {
  font-family: var(--font-body); font-size: var(--fs-xs); color: var(--danger);
  background: var(--danger-weak); border: 1px solid var(--danger-weak);
  border-radius: 7px; padding: 9px 12px; margin-top: 8px;
}
</style>
