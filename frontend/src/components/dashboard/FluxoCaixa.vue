<template>
  <div class="fc">
    <div v-if="carregando" class="fc-state">
      <AppSpinner /> Carregando {{ rotuloCarregando }}...
    </div>
    <div v-else-if="erro" class="fc-state fc-state--erro">{{ erro }}</div>

    <template v-else-if="dados">
      <!-- Faixa de recorte: fina e sempre visível quando há recorte ativo.
           Morava no bloco recolhível, que saiu junto com o layout de três
           colunas. Não podia sair junto: um recorte esquecido explica um número
           "errado" sem dar pista de onde está. -->
      <div v-if="haRecorte && !ui.haFoco" class="fc-recorte">
        <span class="fc-recorte-txt">{{ rotuloRecorte }}</span>
        <button class="fc-btn fc-btn--limpar" @click="limparRecorte">✕ Limpar recorte</button>
      </div>

      <!-- Primeira fileira: categorias e clientes lado a lado. Vivem AQUI
           dentro, e não num componente acima, porque é o que os faz participar
           do recorte cruzado — fora daqui o calendário não os alcançaria. -->
      <div v-show="mostrar(ID.categorias) || mostrar(ID.clientes)" class="fc-graficos">
        <GraficoDonut
          v-show="mostrar(ID.categorias)"
          :class="{ 'em-foco': ui.emFoco(ID.categorias) }"
          :titulo="rotulos.donut"
          :subtitulo="rotuloRecorte"
          :itens="itensCategoria"
          :foco-id="ID.categorias"
          :selecionados="[...selCategorias]"
          @update:selecionados="selCategorias = new Set($event)"
        />
        <GraficoBarrasH
          v-show="mostrar(ID.clientes)"
          :class="{ 'em-foco': ui.emFoco(ID.clientes) }"
          :titulo="rotulos.barras"
          :subtitulo="rotuloRecorte"
          :itens="itensEntidade"
          :foco-id="ID.clientes"
          :selecionados="[...selEntidades]"
          @update:selecionados="selEntidades = new Set($event)"
        />
      </div>

      <!-- Segunda fileira: calendário e a coluna de resumo. É o mesmo arranjo
           das abas de contas — aqui deixou de ser exclusivo do layout lateral. -->
      <div v-show="mostrar(ID.calendario) || mostrar(ID.resumo)" class="fc-grid">
        <section v-show="mostrar(ID.calendario)"
                 :class="['fc-card', 'fc-card--cal', { 'em-foco': ui.emFoco(ID.calendario) }]">
          <div class="fc-card-head">
            <div>
              <div class="fc-card-title">{{ tituloCalendario }}</div>
              <div class="fc-card-sub">
                {{ nomeMes[dados.mes - 1] }} de {{ dados.ano }}
                 · Ctrl+clique soma dias
              </div>
            </div>
            <div class="fc-card-acoes">
              <button v-if="selDias.size" class="fc-btn" @click="selDias = new Set()">
                Ver mês inteiro
              </button>
              <BotaoFoco :id="ID.calendario" />
            </div>
          </div>

          <div class="cal-semana">
            <span v-for="(d, i) in ['D','S','T','Q','Q','S','S']" :key="i">{{ d }}</span>
          </div>
          <div class="cal-grid">
            <div v-for="v in vaziosAntes" :key="`v${v}`" class="cal-vazio" />
            <button
              v-for="d in diasDoMes" :key="d.dia"
              :class="['cal-dia', {
                'cal-dia--sel': selDias.has(d.dia),
                'cal-dia--hoje': d.dia === diaDeHoje,
                'cal-dia--vazio': !d.temLancamento,
              }]"
              :disabled="!d.temLancamento"
              @click="cliqueDia(d.dia, $event)"
            >
              <span class="cal-num">{{ d.dia }}</span>
              <span v-if="d.entradas" class="cal-val cal-val--in">{{ fmtCompacto(d.entradas) }}</span>
              <span v-if="d.saidas" class="cal-val cal-val--out">{{ fmtCompacto(-d.saidas) }}</span>
              <span v-if="d.temPrevisto" class="cal-prev" title="Contém valores previstos" />
            </button>
          </div>
        </section>

        <div v-show="mostrar(ID.resumo)" class="fc-lateral">
          <section class="fc-card">
            <ResumoMes :resumo="resumo" :tipo="tipo" :titulo="tituloResumo" :subtitulo="rotuloRecorte" />
          </section>

          <section class="fc-card fc-card--venc">
            <ProximosVencimentos :itens="dados.proximos_vencimentos" />
          </section>
        </div>
      </div>

      <section v-if="pivoIntradia" v-show="mostrar(ID.intradia)"
               :class="['fc-card', { 'em-foco': ui.emFoco(ID.intradia) }]">
        <PivotDiario :transacoes="recortadas" :ano="dados.ano" :mes="dados.mes" />
      </section>

      <!-- Listagem: realizado e pendente, a coluna Status distingue -->
      <section v-show="mostrar(ID.transacoes)"
               :class="['fc-card', { 'em-foco': ui.emFoco(ID.transacoes) }]">
        <div class="fc-card-head">
          <div>
            <div class="fc-card-title">TRANSAÇÕES</div>
            <div class="fc-card-sub">{{ rotuloRecorte }} — efetuadas e pendentes</div>
          </div>
          <div class="fc-filtros">
            <input v-model="busca" class="fc-input" placeholder="Buscar descrição ou categoria..." />
            <select v-if="!umLadoSo" v-model="filtroTipo" class="fc-select">
              <option value="">Todos os tipos</option>
              <option value="receita">Recebimentos</option>
              <option value="despesa">Pagamentos</option>
            </select>
            <select v-model="filtroSituacao" class="fc-select">
              <option value="">Todas as situações</option>
              <option value="efetuada">Efetuadas</option>
              <option value="pendente">Pendentes</option>
              <option value="atrasada">Atrasadas</option>
            </select>
            <BotaoFoco :id="ID.transacoes" />
          </div>
        </div>

        <p v-if="!listagem.length" class="fc-vazio">Nenhuma transação no período.</p>
        <div v-else class="fc-scroll">
          <table class="fc-table">
            <thead>
              <tr>
                <th>DATA</th><th>DESCRIÇÃO</th><th>TIPO</th><th>CATEGORIA</th>
                <th class="ta-r">VALOR</th><th>STATUS</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(t, i) in listagem" :key="i">
                <td class="mono">{{ t.data }}</td>
                <td>{{ t.descricao }}</td>
                <td>
                  <span :class="['pill', t.tipo === 'receita' ? 'pill--in' : 'pill--out']">
                    {{ t.tipo === 'receita' ? 'Recebimento' : 'Pagamento' }}
                  </span>
                </td>
                <td>{{ t.categoria }}</td>
                <td class="ta-r mono" :class="t.tipo === 'receita' ? 'res-val--in' : 'res-val--out'">
                  {{ fmtMoeda(t.valor) }}
                </td>
                <td>
                  <span :class="['pill', pillStatus(t)]">{{ t.status }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, defineAsyncComponent } from 'vue'
import { fetchFluxoCaixa, type FluxoCaixaData, type FluxoResumo, type FluxoTransacao } from '@/api/fluxocaixa'
import type { DashboardParams } from '@/api/dashboard'
import AppSpinner from '@/components/ui/AppSpinner.vue'
import BotaoFoco from '@/components/ui/BotaoFoco.vue'
import ResumoMes from './ResumoMes.vue'
import ProximosVencimentos from './ProximosVencimentos.vue'
import { fmtMoeda, fmtCompacto } from '@/utils/formato'
import { filtrarTransacoes, classeStatus, resumirTransacoes } from '@/utils/fluxo'
import { aplicarSelecao, alternarNumero, temRecorte, type Selecao } from '@/utils/fluxocruzado'
import { porCategoria, topEntidades } from '@/utils/agregacao'
import { deveMostrar } from '@/utils/foco'
import { rotulosGraficos } from '@/utils/rotulos'
import { useUiStore } from '@/stores/ui'

// Assíncronos: as abas de contas montam este componente no layout 'lateral' e
// nunca precisam dos três, então não devem pagar o download deles.
const GraficoDonut = defineAsyncComponent(() => import('./GraficoDonut.vue'))
const GraficoBarrasH = defineAsyncComponent(() => import('./GraficoBarrasH.vue'))
const PivotDiario = defineAsyncComponent(() => import('./PivotDiario.vue'))

const props = withDefaults(defineProps<{
  grupoId: string
  filtros: DashboardParams
  mes: number
  /** Restringe a visão a um lado: abas Contas a Receber e Contas a Pagar. */
  tipo?: 'todos' | 'receita' | 'despesa'
  /**
   * Acrescenta a tabela intradia (categoria × dia), abaixo do calendário.
   *
   * É a única diferença entre as três abas que montam este componente — antes
   * era um enum `layout` que ligava sete coisas de uma vez, e seis delas
   * passaram a valer para todas.
   */
  pivoIntradia?: boolean
}>(), { tipo: 'todos', pivoIntradia: false })

const ui = useUiStore()

/**
 * Ids dos blocos focáveis desta aba. Reunidos num objeto porque aparecem no
 * template, no v-show e na classe de cada bloco — soltos como literais, um erro
 * de digitação esconderia o bloco em silêncio.
 */
const ID = {
  categorias: 'fc-categorias',
  clientes:   'fc-clientes',
  calendario: 'fc-calendario',
  resumo:     'fc-resumo',
  intradia:   'fc-intradia',
  transacoes: 'fc-transacoes',
} as const

/**
 * O bloco aparece? Sem foco, todos; com foco, só ele. Ver utils/foco.ts.
 *
 * No layout lateral (abas de contas) não há botão de foco, mas a função
 * continua valendo: `ui.foco` é nulo ali, então devolve sempre true.
 */
const mostrar = (id: string) => deveMostrar(ui.foco, id)


const umLadoSo   = computed(() => props.tipo !== 'todos')

const tituloCalendario = computed(() => ({
  todos:   'CALENDÁRIO FINANCEIRO',
  receita: 'CALENDÁRIO DE RECEBIMENTOS',
  despesa: 'CALENDÁRIO DE PAGAMENTOS',
}[props.tipo]))

/** Cliente ou fornecedor conforme o lado. Ver utils/rotulos.ts. */
const rotulos = computed(() => rotulosGraficos(props.tipo))

const rotuloCarregando = computed(() => ({
  todos:   'fluxo de caixa',
  receita: 'recebimentos',
  despesa: 'pagamentos',
}[props.tipo]))

const nomeMes = ['Janeiro','Fevereiro','Março','Abril','Maio','Junho',
                 'Julho','Agosto','Setembro','Outubro','Novembro','Dezembro']

const bruto          = ref<FluxoCaixaData | null>(null)
const carregando     = ref(false)
const erro           = ref('')
const busca          = ref('')
const filtroTipo     = ref('')
const filtroSituacao = ref('')

// ── Recorte cruzado ────────────────────────────────────────────────────────
// No layout 'lateral' só `selDias` chega a ser usado: os gráficos das abas de
// contas ficam no modo 'ocultar' e não emitem seleção. O comportamento daquelas
// abas é, portanto, o mesmo de antes.
const selDias       = ref<Set<number>>(new Set())
const selCategorias = ref<Set<string>>(new Set())
const selEntidades  = ref<Set<string>>(new Set())

const selecao = computed<Selecao>(() => ({
  dias: selDias.value,
  categorias: selCategorias.value,
  entidades: selEntidades.value,
}))
const haRecorte = computed(() => temRecorte(selecao.value))

function cliqueDia(dia: number, ev: MouseEvent) {
  selDias.value = alternarNumero(selDias.value, dia, ev.ctrlKey || ev.metaKey)
}

function limparRecorte() {
  selDias.value = new Set()
  selCategorias.value = new Set()
  selEntidades.value = new Set()
}

/**
 * Na visão de recebimentos o recorte por tipo é aplicado uma vez, aqui:
 * transações, resumo e próximos vencimentos passam a enxergar só receitas.
 * Filtrar em cada consumidor separadamente deixaria algum deles para trás — o
 * resumo já vem totalizado do servidor e precisa ser recalculado junto.
 */
const dados = computed<FluxoCaixaData | null>(() => {
  if (!bruto.value) return null
  if (!umLadoSo.value) return bruto.value

  const doLado = (t: FluxoTransacao) => t.tipo === props.tipo
  const transacoes = bruto.value.transacoes.filter(doLado)
  // Totaliza pela mesma função do dia selecionado e com a mesma ordem de ramos
  // do servidor. Escrito à mão aqui, o "Atrasado" cairia em a_receber e a linha
  // nova nunca apareceria nesta aba — que é justamente onde ela importa.
  const resumo = resumirTransacoes(transacoes, true)

  return {
    ...bruto.value,
    transacoes,
    resumo,
    proximos_vencimentos: bruto.value.proximos_vencimentos.filter(doLado),
  }
})

// ── As quatro visões do mesmo conjunto ─────────────────────────────────────
// Cada bloco ignora a própria dimensão. Ver utils/fluxocruzado.ts: sem isso,
// clicar numa categoria apagaria todas as outras do donut e o gráfico deixaria
// de servir de ponto de partida para o clique seguinte.
const todas = computed(() => dados.value?.transacoes ?? [])
const paraCalendario = computed(() => aplicarSelecao(todas.value, selecao.value, 'dias'))
const paraCategorias = computed(() => aplicarSelecao(todas.value, selecao.value, 'categorias'))
const paraEntidades  = computed(() => aplicarSelecao(todas.value, selecao.value, 'entidades'))
const recortadas     = computed(() => aplicarSelecao(todas.value, selecao.value))

const itensCategoria = computed(() => porCategoria(paraCategorias.value))
const itensEntidade  = computed(() => topEntidades(paraEntidades.value))

/** Descreve o recorte ativo em uma linha, para os subtítulos dos blocos. */
const rotuloRecorte = computed(() => {
  const partes: string[] = []
  const d = [...selDias.value].sort((a, b) => a - b)
  if (d.length === 1) partes.push(`Dia ${d[0]}`)
  else if (d.length > 1) partes.push(`${d.length} dias: ${d.join(', ')}`)
  if (selCategorias.value.size === 1) partes.push([...selCategorias.value][0])
  else if (selCategorias.value.size > 1) partes.push(`${selCategorias.value.size} categorias`)
  if (selEntidades.value.size === 1) partes.push([...selEntidades.value][0])
  else if (selEntidades.value.size > 1) partes.push(`${selEntidades.value.size} entidades`)
  return partes.length ? partes.join(' · ') : 'Mês inteiro'
})

const tituloResumo = computed(() =>
  haRecorte.value ? 'Resumo do recorte' : 'Resumo do mês')

// ── Calendário ─────────────────────────────────────────────────────────────
const diasNoMes = computed(() =>
  dados.value ? new Date(dados.value.ano, dados.value.mes, 0).getDate() : 0
)

/** Quantas células vazias antes do dia 1, para alinhar na coluna do dia da semana. */
const vaziosAntes = computed(() =>
  dados.value ? new Date(dados.value.ano, dados.value.mes - 1, 1).getDay() : 0
)

const diaDeHoje = computed(() => {
  const h = new Date()
  if (!dados.value) return -1
  return h.getFullYear() === dados.value.ano && h.getMonth() + 1 === dados.value.mes
    ? h.getDate() : -1
})

const diasDoMes = computed(() => {
  const base = Array.from({ length: diasNoMes.value }, (_, i) => ({
    dia: i + 1, entradas: 0, saidas: 0, temPrevisto: false, temLancamento: false,
  }))
  for (const t of paraCalendario.value) {
    const d = base[t.dia - 1]
    if (!d) continue
    if (t.tipo === 'receita') d.entradas += t.valor
    else d.saidas += t.valor
    if (!t.realizado) d.temPrevisto = true
    d.temLancamento = true
  }
  return base
})

// ── Resumo: recalculado no cliente quando há recorte ───────────────────────
// Sem recorte usa o total que veio do servidor, para não divergir por
// arredondamento do que o banco reportou.
const resumo = computed<FluxoResumo>(() => {
  if (!dados.value) return resumirTransacoes([])
  if (!haRecorte.value) return dados.value.resumo
  return resumirTransacoes(recortadas.value, umLadoSo.value)
})

// ── Listagem ───────────────────────────────────────────────────────────────
// O recorte cruzado entra antes; `filtrarTransacoes` segue cuidando de busca,
// tipo e situação. `dia: null` porque a dimensão de dia já foi aplicada.
const listagem = computed(() =>
  filtrarTransacoes(recortadas.value, {
    dia: null,
    tipo: filtroTipo.value,
    situacao: filtroSituacao.value,
    busca: busca.value,
  })
)

const pillStatus = classeStatus

// ── Carregamento ───────────────────────────────────────────────────────────
async function carregar() {
  if (!props.grupoId) return
  carregando.value = true
  erro.value = ''
  try {
    bruto.value = await fetchFluxoCaixa(props.grupoId, { ...props.filtros, mes: props.mes })
    // Recorte é uma lente sobre o mês exibido; carregar outro mês o invalida —
    // um dia ou uma categoria que não existe ali deixaria a tela vazia sem
    // motivo aparente.
    limparRecorte()
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message
    erro.value = msg || 'Erro ao carregar fluxo de caixa'
  } finally {
    carregando.value = false
  }
}

watch(() => [props.grupoId, props.filtros, props.mes], carregar, { deep: true, immediate: true })
</script>

<style scoped>
.fc { display: flex; flex-direction: column; gap: 16px; }

.fc-state {
  padding: 48px; text-align: center; gap: 10px;
  display: flex; align-items: center; justify-content: center;
  font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim);
}
.fc-state--erro { color: var(--danger); }

/* Fileiras de dois, como nas abas de contas: categorias e clientes na primeira,
   calendário e a coluna de resumo na segunda.
   Havia aqui uma variante de três colunas, com o calendário ao lado dos dois
   gráficos, e ela obrigava a comprimir o donut para 170px e as células do
   calendário para 38px. Quem precisa de espaço agora expande o bloco. */
.fc-graficos {
  display: grid; grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--sp-4); margin-bottom: var(--sp-4);
}
.fc-grid {
  display: grid; grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--sp-4); align-items: stretch; margin-bottom: var(--sp-4);
}
@media (max-width: 1100px) {
  .fc-graficos, .fc-grid { grid-template-columns: 1fr; }
}

/* Um bloco expandido é o único na tela, então a grade de duas colunas deixaria
   metade vazia à direita. O resto do visual do foco é global, em main.css. */
.fc-graficos:has(> .em-foco), .fc-grid:has(> .em-foco) { grid-template-columns: 1fr; }

/* O card de vencimentos absorve a altura que sobra, então a coluna lateral
   termina na mesma linha do calendário em qualquer viewport. */
.fc-lateral { display: flex; flex-direction: column; gap: var(--sp-4); min-height: 0; }
.fc-card--venc {
  flex: 1 1 auto; min-height: 210px;
  display: flex; flex-direction: column;
}

.fc-card {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--r); padding: var(--sp-5);
}
/* O card do calendario e coluna flex para que .cal-grid possa usar flex:1 e
   preencher a altura, em vez de a altura sair do aspect-ratio da celula. */
.fc-card--cal { display: flex; flex-direction: column; }
.fc-card-head {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: var(--sp-4); flex-wrap: wrap; margin-bottom: var(--sp-4);
}
.fc-card-title {
  font-family: var(--font-mono); font-size: var(--fs-md); font-weight: 600;
  color: var(--text);
}
.fc-card-sub { font-size: var(--fs-xs); color: var(--text-dim); margin-top: 2px; }
.fc-vazio { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim); padding: 16px 0; }

.fc-btn, .fc-input, .fc-select {
  background: var(--surface-2); color: var(--text-muted);
  border: 1px solid var(--border-strong); border-radius: 7px;
  padding: 5px 10px; font-family: var(--font-mono); font-size: var(--fs-xs); outline: none;
}
.fc-btn { cursor: pointer; transition: var(--transition); }
.fc-btn:hover { border-color: var(--primary); color: var(--primary); }
.fc-filtros { display: flex; gap: 6px; flex-wrap: wrap; align-items: center; }
/* Mesma razão do .gr-acoes nos gráficos: o .fc-card-head é space-between, e um
   segundo botão solto ali ficaria no meio do cabeçalho. */
.fc-card-acoes { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.fc-input { min-width: 200px; }

/* Faixa de recorte: própria, no topo da aba. */
.fc-recorte {
  display: flex; align-items: center; gap: var(--sp-3); flex-wrap: wrap;
  margin-bottom: calc(var(--sp-4) * -1 + 4px);
}
.fc-recorte-txt {
  font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: 600;
  color: var(--primary);
  background: var(--primary-weak); border-radius: 20px; padding: 2px 10px;
}
.fc-btn--limpar { border-color: var(--danger); color: var(--danger); }
.fc-btn--limpar:hover { border-color: var(--danger); color: var(--danger); background: var(--danger-weak); }

/* ── Calendário ── */
.cal-semana, .cal-grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 6px; }
.cal-semana {
  margin-bottom: 6px; font-family: var(--font-mono); font-size: var(--fs-xs);
  font-weight: 600; letter-spacing: .04em;
  color: var(--text-dim); text-align: center;
}
.cal-grid { flex: 1; grid-auto-rows: minmax(56px, 1fr); }
.cal-vazio { min-height: 56px; }
.cal-dia {
  position: relative; min-height: 56px;
  display: flex; flex-direction: column; align-items: flex-start; gap: 0;
  padding: 6px 8px; overflow: hidden;
  /* Borda transparente e nao --border-strong: com a celula colapsada em 64px de
     largura a grade fechada pesava demais. A cor aparece no hover e na selecao. */
  background: var(--surface-2); border: 1px solid transparent; border-radius: var(--r-sm);
  cursor: pointer; transition: border-color var(--transition), background var(--transition);
  text-align: left;
}
.cal-dia:hover:not(:disabled) { border-color: var(--primary); }
.cal-dia--vazio { opacity: 0.35; cursor: default; }
.cal-dia--sel { border-color: var(--primary); background: var(--primary-weak); }
.cal-dia--hoje .cal-num { color: var(--primary); font-weight: 700; }
.cal-num { font-family: var(--font-mono); font-size: var(--fs-sm); font-weight: 600; color: var(--text-muted); line-height: 1.3; }
.cal-val {
  font-family: var(--font-mono); font-size: var(--fs-xs); line-height: 1.2;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%;
}
.cal-val--in  { color: var(--success); }
.cal-val--out { color: var(--danger); }
/* Marca discreta de que o dia contém pendência, não só realizado. */
.cal-prev {
  position: absolute; top: 5px; right: 5px;
  width: 6px; height: 6px; border-radius: 50%;
  background: var(--warning);
}

/* ── Tabela ── */
.fc-scroll { overflow-x: auto; }
.fc-table { width: 100%; border-collapse: collapse; min-width: 720px; }
.fc-table th {
  font-family: var(--font-mono); font-size: var(--fs-xs); letter-spacing: 1px; color: var(--text-dim);
  text-align: left; padding: 9px 12px; border-bottom: 1px solid var(--border-strong); white-space: nowrap;
}
.fc-table td {
  padding: 8px 12px; font-size: var(--fs-xs); color: var(--text);
  border-bottom: 1px solid var(--border);
}
.fc-table tr:last-child td { border-bottom: none; }
.fc-table tr:hover td { background: var(--surface-2); }
.ta-r { text-align: right; }
.mono { font-family: var(--font-mono); font-size: var(--fs-xs); }

.res-val--in     { color: var(--success); }
.res-val--out    { color: var(--danger); }

.pill {
  display: inline-flex; padding: 2px 8px; border-radius: 20px;
  font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: 600; white-space: nowrap;
}
.pill--in  { background: var(--success-weak); color: var(--success); }
.pill--out  { background: var(--danger-weak); color: var(--danger); }
.pill--pend { background: var(--warning-weak); color: var(--warning); }
/* Atrasado usa a cor de perigo, e nao o ambar de pendente: vencido nao e a
   mesma coisa que "ainda vai vencer", e a diferenca precisa saltar na lista. */
.pill--atraso { background: var(--danger-weak); color: var(--danger); font-weight: 700; }
</style>
