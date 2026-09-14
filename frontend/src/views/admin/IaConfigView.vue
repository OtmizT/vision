<template>
  <div class="ic-page">
    <header class="ic-head">
      <div>
        <h1 class="ic-titulo">Assistente de IA</h1>
        <p class="ic-sub">
          Credencial única da plataforma. Cada grupo é habilitado separadamente.
        </p>
      </div>
    </header>

    <div v-if="carregando" class="ic-aviso">Carregando…</div>

    <template v-else>
      <!-- ── Credencial ── -->
      <section class="card ic-card">
        <h2 class="section-title">Conexão</h2>

        <div class="ic-grid">
          <div class="field">
            <label for="prov">PROVEDOR</label>
            <select id="prov" v-model="form.provedor" class="input-el">
              <option value="deepseek">DeepSeek</option>
              <option value="openai">OpenAI</option>
              <option value="groq">Groq</option>
            </select>
          </div>

          <div class="field">
            <label for="mod">MODELO</label>
            <input id="mod" v-model="form.modelo" class="input-el" placeholder="deepseek-v4-flash" />
          </div>

          <div class="field ic-larga">
            <label for="url">BASE URL</label>
            <input id="url" v-model="form.base_url" class="input-el"
                   placeholder="vazio usa o padrão do provedor" />
          </div>

          <div class="field ic-larga">
            <label for="key">CHAVE DE API</label>
            <input id="key" v-model="chaveNova" type="password" class="input-el"
                   :placeholder="cfg?.tem_chave ? `Cadastrada (${cfg.api_key_mascarada}) — deixe vazio para manter` : 'Cole a chave aqui'" />
            <!-- A regra que evita a chave circular: vazio PRESERVA. Sem isso a
                 tela precisaria receber a credencial em claro só para salvar
                 uma mudança de modelo. -->
            <p class="ic-dica">
              Deixar em branco mantém a chave atual.
              <button v-if="cfg?.tem_chave" class="ic-link" type="button" @click="limparChave = !limparChave">
                {{ limparChave ? 'cancelar remoção' : 'remover a chave' }}
              </button>
            </p>
            <p v-if="limparChave" class="ic-alerta">A chave será apagada ao salvar.</p>
          </div>

          <div class="field">
            <label for="mt">MAX TOKENS POR RESPOSTA</label>
            <input id="mt" v-model.number="form.max_tokens" type="number" min="256" class="input-el" />
          </div>

          <div class="field">
            <label for="teto">TETO DIÁRIO POR USUÁRIO</label>
            <input id="teto" v-model.number="form.teto_tokens_dia" type="number" min="1000" class="input-el" />
            <p class="ic-dica">Cada pergunta custa tokens. O teto evita que um uso em laço vire fatura.</p>
          </div>
        </div>

        <label class="ic-switch">
          <input type="checkbox" v-model="form.ativo" />
          <span>Assistente ativo na plataforma</span>
        </label>
        <p class="ic-dica">
          Desligando aqui, o recurso some para todos os grupos, sem apagar a credencial.
        </p>

        <p v-if="erro" class="ic-erro">{{ erro }}</p>
        <p v-if="salvo" class="ic-ok">Configuração salva.</p>

        <div class="ic-acoes">
          <button class="btn-primary" :disabled="salvando" @click="salvar">
            {{ salvando ? 'Salvando…' : 'Salvar' }}
          </button>
          <span v-if="cfg?.updated_by_email" class="ic-dica">
            Última alteração por {{ cfg.updated_by_email }}
          </span>
        </div>
      </section>

      <!-- ── Grupos ── -->
      <section class="card ic-card">
        <h2 class="section-title">Grupos habilitados</h2>
        <p class="ic-dica ic-espaco">
          Habilitar um grupo significa que os dados financeiros daquele cliente
          passam a ser enviados ao provedor. Grupo novo nasce desabilitado.
        </p>

        <div v-if="!form.ativo" class="ic-alerta ic-espaco">
          O assistente está desativado na plataforma — nenhum grupo funcionará
          enquanto isso não mudar.
        </div>

        <table class="ic-tabela">
          <thead>
            <tr>
              <th>GRUPO</th>
              <th>ESTADO</th>
              <th>ÚLTIMA ALTERAÇÃO</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="g in grupos" :key="g.grupo_id">
              <td>
                <div class="ic-gnome">{{ g.grupo_nome }}</div>
                <div class="ic-gslug">{{ g.grupo_slug }}</div>
              </td>
              <td>
                <label class="ic-switch">
                  <input
                    type="checkbox"
                    :checked="g.ativa"
                    :disabled="alterando === g.grupo_id"
                    @change="alternarGrupo(g)"
                  />
                  <span :class="g.ativa ? 'ic-on' : 'ic-off'">
                    {{ g.ativa ? 'habilitado' : 'desabilitado' }}
                  </span>
                </label>
              </td>
              <td class="ic-quando">
                <template v-if="g.updated_at">
                  {{ fmtData(g.updated_at) }}
                  <span v-if="g.updated_by_email"> · {{ g.updated_by_email }}</span>
                </template>
                <span v-else class="ic-nunca">nunca alterado</span>
              </td>
            </tr>
            <tr v-if="!grupos.length">
              <td colspan="3" class="ic-vazio">Nenhum grupo cadastrado.</td>
            </tr>
          </tbody>
        </table>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { iaConfigApi, type ConfigIA, type GrupoIA } from '@/api/ia'

const carregando = ref(true)
const salvando = ref(false)
const salvo = ref(false)
const erro = ref('')
const alterando = ref('')

const cfg = ref<ConfigIA | null>(null)
const grupos = ref<GrupoIA[]>([])

// chaveNova é separada do form: vazia significa "não mexer", e misturá-la ao
// resto faria o campo em branco apagar a credencial sem ninguém pedir.
const chaveNova = ref('')
const limparChave = ref(false)

const form = reactive({
  provedor: 'deepseek',
  modelo: 'deepseek-v4-flash',
  base_url: '',
  max_tokens: 2048,
  teto_tokens_dia: 200000,
  ativo: false,
})

function aplicar(c: ConfigIA) {
  cfg.value = c
  form.provedor = c.provedor
  form.modelo = c.modelo
  form.base_url = c.base_url
  form.max_tokens = c.max_tokens
  form.teto_tokens_dia = c.teto_tokens_dia
  form.ativo = c.ativo
}

async function carregar() {
  carregando.value = true
  try {
    const [c, g] = await Promise.all([iaConfigApi.get(), iaConfigApi.grupos()])
    aplicar(c.data.data)
    grupos.value = g.data.data ?? []
  } catch (e: unknown) {
    erro.value = msgDe(e, 'Não foi possível carregar a configuração.')
  } finally {
    carregando.value = false
  }
}

async function salvar() {
  salvando.value = true
  erro.value = ''
  salvo.value = false
  try {
    const { data } = await iaConfigApi.salvar({
      ...form,
      api_key: chaveNova.value,
      limpar_chave: limparChave.value,
    })
    aplicar(data.data)
    chaveNova.value = ''
    limparChave.value = false
    salvo.value = true
  } catch (e: unknown) {
    erro.value = msgDe(e, 'Não foi possível salvar.')
  } finally {
    salvando.value = false
  }
}

async function alternarGrupo(g: GrupoIA) {
  alterando.value = g.grupo_id
  erro.value = ''
  try {
    await iaConfigApi.setGrupo(g.grupo_id, !g.ativa)
    // Recarrega em vez de confiar no estado local: o servidor recusa habilitar
    // com o assistente desativado, e a tela precisa refletir o que ficou
    // gravado, não o que se pediu.
    const { data } = await iaConfigApi.grupos()
    grupos.value = data.data ?? []
  } catch (e: unknown) {
    erro.value = msgDe(e, 'Não foi possível alterar o grupo.')
  } finally {
    alterando.value = ''
  }
}

function msgDe(e: unknown, padrao: string): string {
  const r = e as { response?: { data?: { message?: string } } }
  return r?.response?.data?.message ?? padrao
}

function fmtData(iso: string): string {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '—' : d.toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })
}

onMounted(carregar)
</script>

<style scoped>
.ic-page { padding: 4px 0 40px; }
.ic-head { margin-bottom: 18px; }
.ic-titulo { font-size: var(--fs-lg); font-weight: 700; color: var(--text); margin: 0; }
.ic-sub { font-size: var(--fs-sm); color: var(--text-muted); margin: 3px 0 0; }

.ic-card { padding: 20px 22px; margin-bottom: 18px; }
.ic-aviso { color: var(--text-dim); font-size: var(--fs-sm); padding: 24px 0; }

.ic-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 18px;
  margin-bottom: 16px;
}
.ic-larga { grid-column: 1 / -1; }

@media (max-width: 720px) {
  .ic-grid { grid-template-columns: 1fr; }
}

.field { display: flex; flex-direction: column; gap: 5px; }
label {
  font-size: var(--fs-xs); color: var(--text-dim);
  letter-spacing: 1.2px; text-transform: uppercase;
}
.input-el {
  background: var(--surface-2);
  border: 1px solid var(--border-strong);
  border-radius: var(--r-sm);
  padding: 9px 11px;
  font-size: var(--fs-sm);
  font-family: var(--font-body);
  color: var(--text);
  outline: none;
  transition: border-color var(--transition);
}
.input-el:focus { border-color: var(--primary); }

.ic-dica { font-size: var(--fs-xs); color: var(--text-dim); margin: 0; line-height: 1.5; }
.ic-espaco { margin-bottom: 12px; }

.ic-link {
  background: none; border: none; padding: 0;
  color: var(--primary-line); font-size: var(--fs-xs);
  cursor: pointer; text-decoration: underline;
}

.ic-alerta {
  font-size: var(--fs-xs); color: var(--warning);
  background: var(--warning-weak); border-radius: var(--r-sm);
  padding: 8px 11px; margin: 6px 0 0;
}

.ic-erro {
  font-size: var(--fs-xs); color: var(--danger);
  background: var(--danger-weak); border-radius: var(--r-sm);
  padding: 9px 12px; margin: 12px 0 0;
}
.ic-ok { font-size: var(--fs-xs); color: var(--success); margin: 12px 0 0; }

.ic-switch {
  display: inline-flex; align-items: center; gap: 8px;
  font-size: var(--fs-sm); color: var(--text); cursor: pointer;
  text-transform: none; letter-spacing: 0;
}
.ic-switch input { width: 15px; height: 15px; accent-color: var(--primary); cursor: pointer; }

.ic-on  { color: var(--success); font-weight: 600; }
.ic-off { color: var(--text-dim); }

.ic-acoes { display: flex; align-items: center; gap: 14px; margin-top: 16px; }

.ic-tabela { width: 100%; border-collapse: collapse; font-size: var(--fs-sm); }
.ic-tabela th {
  text-align: left; padding: 8px 10px;
  font-size: var(--fs-xs); color: var(--text-dim);
  letter-spacing: 1.2px; border-bottom: 1px solid var(--border);
  font-weight: 600;
}
.ic-tabela td { padding: 11px 10px; border-bottom: 1px solid var(--border); vertical-align: top; }

.ic-gnome { color: var(--text); font-weight: 600; }
.ic-gslug { font-size: var(--fs-xs); color: var(--text-dim); margin-top: 2px; }
.ic-quando { font-size: var(--fs-xs); color: var(--text-muted); }
.ic-nunca { color: var(--text-dim); font-style: italic; }
.ic-vazio { color: var(--text-dim); text-align: center; padding: 22px 0; }
</style>
