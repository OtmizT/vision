<template>
  <div class="dash-root">

    <!-- Filtros teleportados para dentro da topbar.
         `defer` é obrigatório: #topbar-filters é renderizado por AppTopbar, um irmão
         na mesma árvore. O Vue monta os filhos antes de inserir o elemento pai no
         documento, então sem `defer` o querySelector do target retorna null. -->
    <Teleport defer to="#topbar-filters">
      <div class="tf-inner" @click.stop>
        <!-- Ano -->
        <div class="fi">
          <span class="fi-label">ANO</span>
          <select class="fi-select" v-model="filtros.ano" @change="onAnoChange">
            <option v-for="a in anosDisponiveis" :key="a" :value="a">{{ a }}</option>
          </select>
        </div>

        <!-- Mês: só nas abas mensais; Visão Geral e Resultado são anuais -->
        <div class="fi" v-if="abaMensal">
          <span class="fi-label">MÊS</span>
          <select class="fi-select" v-model.number="mesSelecionado">
            <option v-for="m in mesesDoAno" :key="m.v" :value="m.v">{{ m.n }}</option>
          </select>
        </div>

        <!-- Meses: recorte de exibição, não filtro de dados — por isso não dispara
             nova consulta. Serve a tabela do Resultado e os cards e gráficos da
             Visão Geral; nas abas mensais quem manda é o seletor de MÊS ao lado. -->
        <div class="fi" v-if="abaComMeses">
          <span class="fi-label">MESES</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('meses', $event)">
              {{ rotuloMeses }}<span class="chv">▾</span>
            </button>
          </div>
        </div>

        <!-- Inadimplência: só nas abas de contas, mesmo criterio do seletor de MÊS.
             Nas demais abas nem aparece, porque nao tem efeito nelas. -->
        <div class="fi" v-if="abaDeContas">
          <span class="fi-label">INADIMPLÊNCIA</span>
          <select class="fi-select" v-model="filtros.inadimplencia">
            <option :value="true">Considerar</option>
            <option :value="false">Ignorar</option>
          </select>
        </div>

        <!-- Empresas: só aparece quando há mais de uma no grupo -->
        <div class="fi" v-if="filtrosDisponiveis.empresas.length > 1">
          <span class="fi-label">EMPRESA</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('emp', $event)">
              {{ rotuloLista('empresas', filtrosDisponiveis.empresas.length) }}<span class="chv">▾</span>
            </button>
          </div>
        </div>

        <!-- Contas correntes: apenas das empresas selecionadas -->
        <div class="fi" v-if="filtrosDisponiveis.contas_correntes.length">
          <span class="fi-label">CONTAS</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('contas', $event)">
              {{ rotuloLista('contas_correntes', filtrosDisponiveis.contas_correntes.length) }}<span class="chv">▾</span>
            </button>
          </div>
        </div>

        <!-- Departamentos -->
        <div class="fi" v-if="filtrosDisponiveis.departamentos.length">
          <span class="fi-label">DEPARTAMENTO</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('dept', $event)">
              {{ rotuloLista('departamentos', filtrosDisponiveis.departamentos.length) }}<span class="chv">▾</span>
            </button>
          </div>
        </div>

        <!-- Categorias -->
        <div class="fi" v-if="filtrosDisponiveis.categorias.length">
          <span class="fi-label">CATEGORIA</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('cat', $event)">
              {{ rotuloCategorias }}<span class="chv">▾</span>
            </button>
          </div>
        </div>

        <!-- Cliente/Fornecedor com autocomplete -->
        <div class="fi fi--grow">
          <span class="fi-label">CLIENTE / FORNECEDOR</span>
          <input
            class="fi-input"
            v-model="filtros.cliente"
            placeholder="Buscar..."
            @input="onClienteInput"
            @focus="onClienteFocus($event)"
            @blur="onClienteBlur"
            autocomplete="off"
          />
        </div>

        <!-- Limpar -->
        <button class="fi-clear" @click="limparFiltros" title="Limpar filtros">✕</button>
      </div>
    </Teleport>

    <!-- Dropdowns fixos — renderizados no body via Teleport para escapar de overflow:hidden -->
    <Teleport to="body">
      <!-- Empresas -->
      <div v-if="dropdown === 'emp'" class="fd-fixed" :style="fdStyle" @click.stop>
        <div class="fd-acoes">
          <button class="fd-acao" @click="selecionarTodos('empresas')">Marcar todas</button>
        </div>
        <label v-for="e in filtrosDisponiveis.empresas" :key="e.id" class="chk-item">
          <input type="checkbox" :checked="marcado('empresas', e.id)"
                 @change="alternar('empresas', e.id, filtrosDisponiveis.empresas.map(x => x.id))" />
          {{ e.nome }}
        </label>
      </div>
      <!-- Contas correntes -->
      <div v-if="dropdown === 'contas'" class="fd-fixed" :style="fdStyle" @click.stop>
        <div class="fd-acoes">
          <button class="fd-acao" @click="selecionarTodos('contas_correntes')">Marcar todas</button>
        </div>
        <template v-for="g in gruposContas" :key="g.id">
          <div class="fd-grupo">
            <span class="fd-grupo-rot">{{ g.rotulo }} ({{ g.contas.length }})</span>
            <span class="fd-grupo-acoes">
              <button class="fd-acao" @click="alternarGrupoContas(g, true)">Marcar</button>
              <button class="fd-acao" @click="alternarGrupoContas(g, false)">Desmarcar</button>
            </span>
          </div>
          <label v-for="cc in g.contas" :key="cc.codigo" class="chk-item">
            <input type="checkbox" :checked="marcado('contas_correntes', cc.codigo)"
                   @change="alternar('contas_correntes', cc.codigo, todasAsContas())" />
            {{ cc.descricao }}
          </label>
        </template>
      </div>
      <!-- Departamentos -->
      <div v-if="dropdown === 'dept'" class="fd-fixed" :style="fdStyle" @click.stop>
        <div class="fd-acoes">
          <button class="fd-acao" @click="selecionarTodos('departamentos')">Marcar todos</button>
        </div>
        <label v-for="d in filtrosDisponiveis.departamentos" :key="d" class="chk-item">
          <input type="checkbox" :checked="marcado('departamentos', d)"
                 @change="alternar('departamentos', d, filtrosDisponiveis.departamentos)" />
          {{ d }}
        </label>
      </div>
      <!-- Categorias: lista de exclusão, com "Transferência" oculta por padrão -->
      <div v-if="dropdown === 'cat'" class="fd-fixed" :style="fdStyle" @click.stop>
        <div class="fd-acoes">
          <button class="fd-acao" @click="todasCategorias">Marcar todas</button>
          <button class="fd-acao" @click="padraoCategorias">Padrão</button>
        </div>
        <label v-for="c in filtrosDisponiveis.categorias" :key="c" class="chk-item">
          <input type="checkbox" :checked="categoriaMarcada(c)" @change="alternarCategoria(c)" />
          {{ c }}
        </label>
      </div>
      <!-- Meses do Resultado -->
      <div v-if="dropdown === 'meses'" class="fd-fixed" :style="fdStyle" @click.stop>
        <div class="fd-acoes">
          <button class="fd-acao" @click="todosOsMeses">Ano inteiro</button>
          <button class="fd-acao" @click="nenhumMes">Limpar</button>
        </div>
        <label v-for="m in mesesDoAno" :key="m.v" class="chk-item">
          <input type="checkbox" :checked="mesesVisiveisSet.has(m.v)" @change="alternarMes(m.v)" />
          {{ m.n }}
        </label>
      </div>

      <!-- Autocomplete cliente -->
      <div v-if="dropdown === 'cli' && clienteSugestoes.length" class="fd-fixed" :style="fdStyle" @click.stop>
        <div
          v-for="s in clienteSugestoes"
          :key="s"
          class="chk-item chk-item--click"
          @mousedown.prevent="selecionarCliente(s)"
        >
          {{ s }}
        </div>
      </div>
    </Teleport>

    <!-- Abas na faixa fixa do cabeçalho, logo abaixo dos filtros.
         Mesmo `defer` dos filtros: #appbar-tabs é renderizado pelo MainLayout e
         sem ele o querySelector do alvo devolve null na montagem. -->
    <Teleport defer to="#appbar-tabs">
      <div v-if="!ui.haFoco" class="dash-tabs">
        <button v-for="t in abas" :key="t.id"
                :class="['dash-tab', { active: aba === t.id }]"
                @click="aba = t.id">{{ t.label }}</button>
      </div>
    </Teleport>

    <!-- Loading / erro -->
    <div v-if="carregando && !dados" class="state-msg">
      <AppSpinner /> Carregando dashboard...
    </div>
    <div v-else-if="erro" class="state-msg state-msg--erro">{{ erro }}</div>

    <!-- Aba Resultado: componente carregado sob demanda para não pesar no bundle
         inicial do dashboard, que já é o maior da aplicação. -->
    <ResultadoPivot v-else-if="aba === 'resultado'" :grupo-id="grupoIDAtivo"
                    :filtros="filtrosAtivos" :meses="mesesVisiveis" />
    <!-- As três abas montam o MESMO componente, variando só o recorte por tipo e
         a presença da tabela intradia. Recebíveis e pagamentos passavam antes por
         um ContasPorTipo, que segurava os gráficos do lado de fora — e era por
         isso que o calendário deles não filtrava os gráficos. -->
    <FluxoCaixa v-else-if="aba === 'fluxo'" :grupo-id="grupoIDAtivo"
                :filtros="filtrosAtivos" :mes="mesSelecionado" pivo-intradia />
    <FluxoCaixa v-else-if="aba === 'receber'" :grupo-id="grupoIDAtivo"
                :filtros="filtrosAtivos" :mes="mesSelecionado" tipo="receita" />
    <FluxoCaixa v-else-if="aba === 'pagar'" :grupo-id="grupoIDAtivo"
                :filtros="filtrosAtivos" :mes="mesSelecionado" tipo="despesa" />

    <!-- Conteúdo -->
    <div v-else-if="dados" class="dash-content">

      <!--
        Cards KPI. Sem faixa "INDICADORES" acima: os quatro rótulos já dizem o
        que cada número é, e o título só empurrava o conteúdo para baixo.

        A altura não é fixada — vem do conteúdo, que difere por card: receita e
        despesa levam sparkline, resultado leva a barra de margem e saldo a
        contagem de contas. Fixar altura cortaria o mais alto ou deixaria folga
        nos outros.
      -->
      <div class="cards-row">
        <div class="kpi-card">
          <div class="kpi-header">
            <span class="kpi-label">Receita total</span>
            <span class="kpi-icon kpi-icon--green"><IconArrowUpRight /></span>
          </div>
          <div class="kpi-value kpi-value--green">{{ fmtExato(cardsRecorte.receita) }}</div>
          <div class="spark-wrap"><canvas ref="canvasSparkRec" /></div>
        </div>

        <div class="kpi-card">
          <div class="kpi-header">
            <span class="kpi-label">Despesa total</span>
            <span class="kpi-icon kpi-icon--red"><IconArrowDownRight /></span>
          </div>
          <div class="kpi-value kpi-value--red">{{ fmtDespesa(cardsRecorte.despesa) }}</div>
          <div class="spark-wrap"><canvas ref="canvasSparkDesp" /></div>
        </div>

        <div class="kpi-card">
          <div class="kpi-header">
            <span class="kpi-label">Resultado</span>
            <span class="kpi-icon kpi-icon--accent"><IconLineChart /></span>
          </div>
          <div class="kpi-value" :class="resultadoRecorte >= 0 ? 'kpi-value--green' : 'kpi-value--red'">
            {{ fmtExato(resultadoRecorte) }}
          </div>
          <div class="progress"><i :style="{ width: margemBarra }" /></div>
          <div class="kpi-foot">{{ margemTexto }}</div>
        </div>

        <div class="kpi-card">
          <div class="kpi-header">
            <span class="kpi-label">Saldo em contas</span>
            <span class="kpi-icon kpi-icon--cyan"><IconCreditCard /></span>
          </div>
          <!-- Mesma regra do card Resultado ao lado: cor pelo sinal, e os parenteses
               do negativo vem do proprio fmtExato. -->
          <div class="kpi-value"
               :class="saldoRecorte >= 0 ? 'kpi-value--green' : 'kpi-value--red'">
            {{ fmtExato(saldoRecorte) }}
          </div>
          <div class="kpi-foot">
            <span class="tag tag--ok">{{ contasNoFluxo }}</span> · {{ rotuloSaldo }}
          </div>
        </div>
      </div>

      <!-- Gráficos -->
      <div class="charts-row">
        <div class="chart-card chart-card--clicavel">
          <div class="chart-header">
            <div>
              <div class="chart-title">Receita vs Despesa</div>
              <div class="chart-sub">
                {{ rotuloRecorteGeral }} — {{ filtros.ano }} · clique numa barra para recortar
              </div>
            </div>
            <div class="chart-legend">
              <button v-if="haRecorteMeses" class="chart-limpar" @click="todosOsMeses">
                ✕ {{ rotuloMeses }}
              </button>
              <span class="leg-dot" style="background:var(--success)"></span>Receita
              <span class="leg-dot" style="background:var(--danger)"></span>Despesa
              <span class="leg-dot" style="background:var(--primary)"></span>Resultado
            </div>
          </div>
          <div class="chart-wrap"><canvas ref="canvasRecDesp"></canvas></div>
        </div>

        <div class="chart-card">
          <div class="chart-header">
            <div>
              <div class="chart-title">Resultado Acumulado</div>
              <div class="chart-sub">
                Do saldo em contas ao saldo projetado — {{ rotuloRecorteGeral }}
              </div>
            </div>
            <div class="chart-legend">
              <span class="leg-dot" style="background:var(--success)"></span>Aumento
              <span class="leg-dot" style="background:var(--danger)"></span>Redução
              <span class="leg-dot" style="background:var(--accent)"></span>Saldo e total
            </div>
          </div>
          <div class="chart-wrap"><canvas ref="canvasAcum"></canvas></div>
        </div>
      </div>

    </div>

    <div v-else class="state-msg">Nenhum dado disponível.</div>

  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch, nextTick, defineAsyncComponent } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Chart, registerables } from 'chart.js'
import ChartDataLabels from 'chartjs-plugin-datalabels'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { fetchDashboard, fetchFiltros, type DashboardData, type FiltrosDisponiveis } from '@/api/dashboard'
import AppSpinner from '@/components/ui/AppSpinner.vue'
import { fmtMoeda, fmtMoedaExata, fmtCompacto } from '@/utils/formato'
import { coresGrafico, comAlfa } from '@/utils/tema'
import { cardsDoRecorte, saldoDoRecorte, cardResultado, margemDoRecorte } from '@/utils/visaogeral'
import { montarCascata, valorDoPasso, alinhamentoDoPasso } from '@/utils/cascata'
import { alternarNumero } from '@/utils/fluxocruzado'
import {
  IconArrowUpRight, IconArrowDownRight, IconLineChart, IconCreditCard,
} from '@/components/ui/icons'
import { agruparContas, aplicarGrupo, type GrupoContas } from '@/utils/contas'

Chart.register(...registerables, ChartDataLabels)

const auth   = useAuthStore()
const ui     = useUiStore()   // fonte reativa do tema para os gráficos
const router = useRouter()
const route  = useRoute()

const ResultadoPivot = defineAsyncComponent(
  () => import('@/components/dashboard/ResultadoPivot.vue')
)
const FluxoCaixa = defineAsyncComponent(
  () => import('@/components/dashboard/FluxoCaixa.vue')
)

const abas = [
  { id: 'geral',     label: 'VISÃO GERAL' },
  { id: 'resultado', label: 'RESULTADO'   },
  { id: 'fluxo',     label: 'FLUXO DE CAIXA' },
  { id: 'receber',   label: 'CONTAS A RECEBER' },
  { id: 'pagar',     label: 'CONTAS A PAGAR' },
] as const
const aba = ref<'geral' | 'resultado' | 'fluxo' | 'receber' | 'pagar'>('geral')

/** Abas mensais: são as que exibem o seletor de MÊS. */
const abasMensais = ['fluxo', 'receber', 'pagar']
const abaMensal = computed(() => abasMensais.includes(aba.value))

/** Abas que somam a inadimplencia e exibem o seletor correspondente. */
const abaDeContas = computed(() => aba.value === 'receber' || aba.value === 'pagar')

/** Abas que recortam por conjunto de meses (e não por um mês só). */
const abaComMeses = computed(() => aba.value === 'resultado' || aba.value === 'geral')

/**
 * Meses exibidos — no pivô do Resultado e, agora, na Visão Geral.
 *
 * Vive fora de `filtrosAtivos` de propósito: o ResultadoPivot observa `filtros`
 * com deep:true e refaz a consulta a cada mudança, e `carregar()` faz o mesmo.
 * Como o servidor já devolve os doze meses, recortar é decisão de exibição —
 * dentro dos filtros, cada clique num mês custaria uma ida ao banco à toa, e o
 * clique na barra do gráfico (que escreve aqui) deixaria de ser instantâneo.
 *
 * Compartilhado entre as duas abas de propósito: trocar de aba mantendo o
 * recorte é o que o usuário espera depois de escolher um trimestre.
 */
const mesesVisiveis = ref<number[]>([1,2,3,4,5,6,7,8,9,10,11,12])
const mesesVisiveisSet = computed(() => new Set(mesesVisiveis.value))

function alternarMes(m: number) {
  const s = new Set(mesesVisiveis.value)
  s.has(m) ? s.delete(m) : s.add(m)
  mesesVisiveis.value = [...s].sort((a, b) => a - b)
}
function todosOsMeses()  { mesesVisiveis.value = [1,2,3,4,5,6,7,8,9,10,11,12] }
function nenhumMes()     { mesesVisiveis.value = [] }

const rotuloMeses = computed(() =>
  mesesVisiveis.value.length === 12 ? 'Ano inteiro'
    : mesesVisiveis.value.length === 0 ? 'Nenhum mês'
    : `${mesesVisiveis.value.length} de 12`
)

/**
 * Mês só existe no Fluxo de Caixa. Visão Geral e Resultado são anuais — filtrar
 * por mês ali esvaziaria onze das doze colunas, então o seletor some fora da aba.
 */
const mesSelecionado = ref(new Date().getMonth() + 1)
const mesesDoAno = [
  { v: 1,  n: 'Janeiro' },   { v: 2,  n: 'Fevereiro' }, { v: 3,  n: 'Março' },
  { v: 4,  n: 'Abril' },     { v: 5,  n: 'Maio' },      { v: 6,  n: 'Junho' },
  { v: 7,  n: 'Julho' },     { v: 8,  n: 'Agosto' },    { v: 9,  n: 'Setembro' },
  { v: 10, n: 'Outubro' },   { v: 11, n: 'Novembro' },  { v: 12, n: 'Dezembro' },
]

// Grupo resolvido em carregar(); a aba Resultado consulta o mesmo grupo.
const grupoIDAtivo = ref('')

// Espelha os filtros da topbar no formato que a API espera, para que as duas abas
// mostrem sempre o mesmo recorte.
const filtrosAtivos = computed(() => ({
  ano:              filtros.ano,
  empresas:         filtros.empresas.length         ? filtros.empresas         : undefined,
  contas_correntes: filtros.contas_correntes.length ? filtros.contas_correntes : undefined,
  departamentos:    filtros.departamentos.length    ? filtros.departamentos    : undefined,
  categorias:       filtros.categorias.length       ? filtros.categorias       : undefined,
  cliente:          filtros.cliente || undefined,
  categorias_excluir: filtros.categorias_excluir.length ? filtros.categorias_excluir : undefined,
  // Enviado apenas nas abas de contas: Fluxo de Caixa usa o mesmo endpoint e
  // precisa continuar devolvendo exatamente o que devolvia antes.
  inadimplencia:    abaDeContas.value && filtros.inadimplencia ? true : undefined,
}))

// ── Estado ─────────────────────────────────────────────────────────────────
const dados      = ref<DashboardData | null>(null)
const carregando = ref(false)
const erro       = ref('')
const dropdown   = ref<string | null>(null)

const anoAtual = new Date().getFullYear()

/**
 * Inclui o ano seguinte: a matvw do ano corrente guarda provisões com
 * `ano >= ano atual`, e o executor de extrato coleta 1 ano à frente. Sem o
 * ano+1 na lista, esse dado ficava gravado e inalcançável pela tela.
 *
 * O ano seguinte vem parcialmente povoado — o horizonte termina no mesmo dia
 * do ano que vem, então os últimos meses ficam vazios. É fiel ao coletado.
 */
const anosDisponiveis = computed(() => {
  const anos = []
  for (let a = anoAtual + 1; a >= anoAtual - 5; a--) anos.push(a)
  return anos
})

/**
 * Categoria oculta por padrão. É lista de EXCLUSÃO: se fosse de inclusão, uma
 * categoria nova no Omie ficaria fora dos números em silêncio até alguém marcá-la.
 */
// Categoria FINAL, nao superior: o filtro passou a operar na final, e a superior
// "Transferência" nao existe mais como opcao. Consequencia esperada — as demais
// finais que estavam sob ela passam a aparecer e a somar.
const CATEGORIA_OCULTA_PADRAO = 'Transf. Inter Empresas/Contas'

const filtros = reactive({
  ano:               anoAtual,
  contas_correntes:  [] as string[],
  departamentos:     [] as string[],
  categorias:        [] as string[],
  empresas:          [] as string[],
  cliente:           '',
  categorias_excluir: [CATEGORIA_OCULTA_PADRAO] as string[],
  // Ligado por padrao: quem abre Contas a Pagar ou a Receber quer ver o que
  // esta vencido. Desligado, ninguem descobriria a opcao.
  inadimplencia:     true,
})

/**
 * Ordem da cascata. Cada filtro é restringido apenas pelos anteriores, e mudar um
 * limpa os seguintes — senão sobra uma conta selecionada que não pertence mais à
 * empresa escolhida e o resultado vem vazio sem explicação.
 */
const ordemCascata = ['empresas', 'contas_correntes', 'departamentos', 'cliente'] as const

function limparAbaixo(alterado: string) {
  const i = ordemCascata.indexOf(alterado as typeof ordemCascata[number])
  if (i < 0) return
  for (const campo of ordemCascata.slice(i + 1)) {
    if (campo === 'cliente') filtros.cliente = ''
    else filtros[campo] = []
  }
}

type CampoLista = 'empresas' | 'contas_correntes' | 'departamentos'

/**
 * Seleção vazia significa "todas". As caixas então aparecem MARCADAS nesse estado —
 * é o que de fato está sendo exibido, e deixá-las desmarcadas mostrando tudo seria
 * enganoso.
 */
function marcado(campo: CampoLista, valor: string): boolean {
  return filtros[campo].length === 0 || filtros[campo].includes(valor)
}

function alternar(campo: CampoLista, valor: string, todos: string[]) {
  const atual = filtros[campo]
  if (atual.length === 0) {
    // Saindo do "todas": materializa a lista completa menos o desmarcado.
    filtros[campo] = todos.filter(v => v !== valor)
  } else if (atual.includes(valor)) {
    filtros[campo] = atual.filter(v => v !== valor)
  } else {
    const proximo = [...atual, valor]
    // Marcar tudo volta a "todas" (array vazio), para que item novo entre sozinho.
    filtros[campo] = proximo.length === todos.length ? [] : proximo
  }
  limparAbaixo(campo)
  carregar()
}

/**
 * Ano é o topo da cascata: um departamento ou categoria que existia em 2026 pode não
 * existir em 2025, e a seleção obsoleta produziria tela vazia sem explicação. Limpar
 * é preferível — o custo é ter de remarcar ao comparar o mesmo recorte entre anos.
 * A exclusão de categorias é preservada, porque é preferência de exibição.
 */
function onAnoChange() {
  filtros.empresas         = []
  filtros.contas_correntes = []
  filtros.departamentos    = []
  filtros.categorias       = []
  filtros.cliente          = ''
  carregar()
}

// ── Contas correntes agrupadas pela marca do Omie ──────────────────────────
// filtrosDisponiveis é reactive, não ref: acessar .value aqui devolve undefined.
const gruposContas = computed(() => agruparContas(filtrosDisponiveis.contas_correntes))

function todasAsContas(): string[] {
  return filtrosDisponiveis.contas_correntes.map(c => c.codigo)
}

/** Marca ou desmarca de uma vez todas as contas de um grupo. */
function alternarGrupoContas(g: GrupoContas, marcar: boolean) {
  filtros.contas_correntes = aplicarGrupo(
    todasAsContas(), filtros.contas_correntes, g.contas.map(c => c.codigo), marcar,
  )
  limparAbaixo('contas_correntes')
  carregar()
}

/** Marcar todos e limpar têm o mesmo efeito: array vazio = sem filtro = todas. */
function selecionarTodos(campo: CampoLista) {
  filtros[campo] = []
  limparAbaixo(campo)
  carregar()
}

// ── Categorias: lista de exclusão, não de inclusão ─────────────────────────
function categoriaMarcada(valor: string): boolean {
  return !filtros.categorias_excluir.includes(valor)
}

function alternarCategoria(valor: string) {
  filtros.categorias_excluir = filtros.categorias_excluir.includes(valor)
    ? filtros.categorias_excluir.filter(v => v !== valor)
    : [...filtros.categorias_excluir, valor]
  carregar()
}

function todasCategorias() {
  filtros.categorias_excluir = []
  carregar()
}

function padraoCategorias() {
  filtros.categorias_excluir = [CATEGORIA_OCULTA_PADRAO]
  carregar()
}

/** Rótulo do gatilho: informa o estado sem obrigar a abrir o dropdown. */
function rotuloLista(campo: CampoLista, total: number): string {
  const n = filtros[campo].length
  return n === 0 ? 'Todas' : `${n} de ${total}`
}

const rotuloCategorias = computed(() => {
  const ex = filtros.categorias_excluir.length
  if (ex === 0) return 'Todas'
  if (ex === 1 && filtros.categorias_excluir[0] === CATEGORIA_OCULTA_PADRAO) {
    return 'Todas exc. Transf.'
  }
  return `${ex} oculta${ex > 1 ? 's' : ''}`
})

const filtrosDisponiveis = reactive<FiltrosDisponiveis>({
  contas_correntes: [],
  departamentos:    [],
  categorias:       [],
  clientes:         [],
  empresas:         [],
})

// Posição do dropdown fixo (calculada via getBoundingClientRect)
const fdPos = reactive({ top: 0, left: 0, right: 0, useRight: false })
const fdStyle = computed(() => fdPos.useRight
  ? { top: `${fdPos.top}px`, right: `${fdPos.right}px` }
  : { top: `${fdPos.top}px`, left: `${fdPos.left}px` }
)

// Autocomplete cliente
const clienteSugestoes = computed(() => {
  const q = filtros.cliente.trim().toLowerCase()
  if (!q || q.length < 2) return []
  return filtrosDisponiveis.clientes
    .filter(c => c.toLowerCase().includes(q))
    .slice(0, 20)
})

// ── Gráficos ───────────────────────────────────────────────────────────────
const canvasRecDesp    = ref<HTMLCanvasElement | null>(null)
const canvasAcum       = ref<HTMLCanvasElement | null>(null)
const canvasSparkRec   = ref<HTMLCanvasElement | null>(null)
const canvasSparkDesp  = ref<HTMLCanvasElement | null>(null)
let chartRecDesp:   Chart | null = null
let chartAcum:      Chart | null = null
let chartSparkRec:  Chart | null = null
let chartSparkDesp: Chart | null = null

// Formatação contábil: negativo entre parênteses. Ver src/utils/formato.ts.
const fmt  = fmtMoeda
const fmtExato = fmtMoedaExata   // cards de KPI: valor cheio, com centavos

/**
 * Despesa sempre entre parênteses, mesmo vindo positiva da API. Sem isso o card
 * de despesa e o de receita mostram dois números de aparência idêntica, e a
 * leitura rápida sugere que um soma ao outro.
 */
function fmtDespesa(v: number): string {
  return fmtMoedaExata(-Math.abs(v))
}
const fmtK = fmtCompacto

/**
 * Cores e fonte vindas dos tokens CSS, resolvidas para valores concretos.
 *
 * Não dá para passar uma referência de token ao Chart.js: o canvas descarta a string como
 * cor inválida. A tabela de duplas dark/light que havia aqui também saiu — era
 * uma terceira cópia da paleta, que não acompanhava mudança de token.
 */
function chartColors() {
  const c = coresGrafico()
  return {
    ...c,
    grid:          c.grade,
    label:         c.rotulo,
    bg:            c.fundo,
    tooltip:       c.tooltipFundo,
    tooltipBorder: c.tooltipBorda,
    tooltipText:   c.tooltipTexto,
  }
}

/**
 * Alfa do preenchimento das barras, nos dois gráficos da Visão Geral.
 *
 * Era 0.25 na receita e 0.20 na despesa, contra 0.70 no gráfico ao lado — lado a
 * lado, um parecia lavado em relação ao outro, e os dois em relação aos próprios
 * pontos de legenda, que são opacos. Um valor só, num lugar só, impede a
 * divergência de voltar.
 */
const ALFA_BARRA = 0.7
/** Barra fora do recorte: presente para comparação, mas claramente secundária. */
const ALFA_APAGADA = 0.18

/*
 * Espessura das barras nos dois gráficos.
 *
 * Precisam ser declaradas, e diferentes, para PARECEREM iguais: Receita vs
 * Despesa tem duas barras por mês e a cascata tem uma, então o mesmo par de
 * percentuais renderiza uma metade da largura da outra. Os valores abaixo foram
 * medidos no navegador, não estimados — ver a verificação da entrega.
 */
const PROPORCAO_PAR   = { categoryPercentage: 0.9,  barPercentage: 0.95 }
const PROPORCAO_UNICA = { categoryPercentage: 0.55, barPercentage: 0.9 }

function buildChartRecDesp() {
  if (!canvasRecDesp.value || !dados.value) return
  chartRecDesp?.destroy()
  const c  = chartColors()
  const ms = dados.value.grafico_mensal

  // O gráfico NÃO se filtra pela própria dimensão: os doze meses continuam na
  // tela e o recorte aparece como esmaecimento. Recortando junto, o clique
  // deixaria uma barra só e não haveria como comparar nem voltar.
  const dentro = (i: number) => !haRecorteMeses.value || mesesVisiveisSet.value.has(ms[i].mes)
  const pinta = (cor: string) => ms.map((_, i) => comAlfa(cor, dentro(i) ? ALFA_BARRA : ALFA_APAGADA))
  const borda = (cor: string) => ms.map((_, i) => dentro(i) ? cor : comAlfa(cor, 0.35))

  chartRecDesp = new Chart(canvasRecDesp.value, {
    type: 'bar',
    data: {
      labels: ms.map(m => m.mes_nome),
      datasets: [
        {
          label: 'Receita', type: 'bar',
          data: ms.map(m => m.receita),
          backgroundColor: pinta(c.receita),
          borderColor: borda(c.receita), borderWidth: 1.5, borderRadius: 4, order: 2,
          ...PROPORCAO_PAR,
        },
        {
          label: 'Despesa', type: 'bar',
          data: ms.map(m => m.despesa),
          backgroundColor: pinta(c.despesa),
          borderColor: borda(c.despesa), borderWidth: 1.5, borderRadius: 4, order: 2,
          ...PROPORCAO_PAR,
        },
        {
          label: 'Resultado', type: 'line',
          data: ms.map(m => m.resultado_mes),
          borderColor: c.linha, borderWidth: 2.5,
          backgroundColor: comAlfa(c.linha, 0.12), fill: true, tension: 0.4,
          pointRadius: 4, pointBackgroundColor: c.linha,
          pointBorderColor: c.bg, pointBorderWidth: 2, order: 1,
        },
      ],
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      // Clique na barra recorta a tela. `index` e não `nearest`: o alvo é o mês,
      // não a série, então clicar na receita ou na despesa de março dá no mesmo.
      onClick: (ev, _els, chart) => {
        // `ev`, e não `ev.native`: o evento normalizado já carrega x/y relativos
        // à área do gráfico, calculados pelo próprio Chart.js. Com o nativo ele
        // os recalcula a partir de `event.target` e do getBoundingClientRect —
        // uma volta ao DOM desnecessária, e que falha se o alvo não estiver lá.
        const alvo = chart.getElementsAtEventForMode(
          ev as unknown as Event, 'index', { intersect: false }, false)
        if (!alvo.length) return
        const mes = ms[alvo[0].index]?.mes
        if (!mes) return
        const e = ev.native as MouseEvent
        mesesVisiveis.value = [...alternarNumero(
          mesesVisiveisSet.value, mes, e.ctrlKey || e.metaKey)].sort((a, b) => a - b)
        // Conjunto vazio significaria "nenhum mês" e esvaziaria os cards; aqui o
        // gesto de desmarcar o único mês quer dizer "voltar ao ano inteiro".
        if (!mesesVisiveis.value.length) todosOsMeses()
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: c.tooltip, borderColor: c.tooltipBorder, borderWidth: 1,
          titleColor: c.tick, bodyColor: c.tooltipText, padding: 12,
          callbacks: { label: ctx => ` ${ctx.dataset.label}: ${fmtK(ctx.parsed.y)}` },
        },
        datalabels: {
          display: (ctx) => (ctx.parsed?.y ?? 0) !== 0,
          font: { family: c.fonte, size: 9, weight: 'bold' },
          formatter: (v: number) => fmtK(v),
          anchor: 'end',
          align:  'top',
          color:  (ctx) => {
            if (ctx.dataset.label === 'Receita')   return c.receita
            if (ctx.dataset.label === 'Despesa')   return c.despesa
            if (ctx.dataset.label === 'Resultado') return c.linha
            return c.label
          },
          offset: 2,
        },
      },
      scales: {
        x: {
          grid: { display: false },
          ticks: { color: c.tick, font: { family: c.fonte, size: 10 } },
          border: { display: false },
        },
        y: {
          grid: { color: c.grid },
          ticks: { color: c.tick, font: { family: c.fonte, size: 10 }, callback: v => fmtK(Number(v)) },
          border: { display: false },
        },
      },
      layout: { padding: { top: 24 } },
    },
  })
}

/**
 * Sparkline de um card: só a linha, sem eixo, grade, legenda ou rótulo.
 *
 * O contêiner `.spark-wrap` tem altura fixa e o canvas é absoluto dentro dele.
 * Isso não é estética: com `responsive: true` o Chart.js dimensiona pelo pai, e
 * num pai de altura automática o canvas realimenta o próprio tamanho e cresce
 * sem limite — no mockup chegou a 431×2644px antes de eu fechar a caixa.
 */
function buildSpark(
  canvas: HTMLCanvasElement | null,
  serie: number[],
  cor: string,
): Chart | null {
  if (!canvas || serie.length < 2) return null
  return new Chart(canvas, {
    type: 'line',
    data: {
      labels: serie.map((_, i) => i),
      datasets: [{
        data: serie,
        borderColor: cor,
        backgroundColor: comAlfa(cor, 0.14),
        borderWidth: 1.75,
        fill: true,
        tension: 0.38,
        pointRadius: 0,
      }],
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      // Sem interação: a sparkline é ilustrativa e o número exato já está acima.
      events: [],
      plugins: { legend: { display: false }, tooltip: { enabled: false }, datalabels: { display: false } },
      scales: { x: { display: false }, y: { display: false } },
      layout: { padding: 0 },
    },
  })
}

function buildSparklines() {
  if (!dados.value) return
  chartSparkRec?.destroy()
  chartSparkDesp?.destroy()
  const c  = coresGrafico()
  const ms = dados.value.grafico_mensal
  chartSparkRec  = buildSpark(canvasSparkRec.value,  ms.map(m => m.receita), c.receita)
  chartSparkDesp = buildSpark(canvasSparkDesp.value, ms.map(m => m.despesa), c.despesa)
}

/**
 * Cards recomputados pelo recorte de meses.
 *
 * Os valores prontos em `dados.cards` continuam existindo e valem o ano inteiro;
 * com um recorte ativo eles deixariam de corresponder ao que os gráficos abaixo
 * mostram. A aritmética está em utils/visaogeral.ts, testada — número errado num
 * card continua parecendo um número.
 */
const cardsRecorte = computed(() =>
  cardsDoRecorte(dados.value?.grafico_mensal ?? [], mesesVisiveisSet.value))

/** Saldo do último mês marcado. Ver saldoDoRecorte: saldo é instante, não soma. */
const saldoRecorte = computed(() =>
  saldoDoRecorte(dados.value?.saldos_mensais ?? [], mesesVisiveisSet.value))

const resultadoRecorte = computed(() => cardResultado(cardsRecorte.value, saldoRecorte.value))

/** Há recorte quando nem todos os doze meses estão marcados. */
const haRecorteMeses = computed(() => mesesVisiveis.value.length !== 12)

/** Descreve o recorte em uma linha, para os subtítulos dos gráficos. */
const rotuloRecorteGeral = computed(() => {
  const ms = mesesVisiveis.value
  if (ms.length === 12) return 'Ano inteiro'
  if (ms.length === 0)  return 'Nenhum mês'
  if (ms.length === 1)  return mesesDoAno[ms[0] - 1].n
  return `${ms.length} meses`
})

/**
 * Rodapé do card de saldo. Diz a QUE DATA o saldo se refere — sem isso o número
 * muda ao trocar de mês sem nada na tela explicando por quê.
 */
const rotuloSaldo = computed(() => {
  const ms = mesesVisiveis.value
  if (!ms.length) return 'sem período'
  const ultimo = ms[ms.length - 1]
  return `saldo ao fim de ${mesesDoAno[ultimo - 1].n.toLowerCase()}`
})

/**
 * Margem sobre a receita. Receita zero devolve 0 em vez de dividir: o período
 * pode não ter faturamento e uma divisão por zero pintaria "NaN%" no card.
 */
const margemPct = computed(() => {
  const m = margemDoRecorte(resultadoRecorte.value, cardsRecorte.value.receita)
  return m === null ? 0 : m * 100
})

// A barra é limitada a 0–100%: resultado acima da receita (por conta do saldo
// somado) passaria de 100 e vazaria da caixa.
const margemBarra = computed(() => `${Math.max(0, Math.min(100, margemPct.value))}%`)

const margemTexto = computed(() => {
  const r = cardsRecorte.value.receita
  if (r <= 0) return 'Sem receita no período'
  return `Margem de ${margemPct.value.toFixed(1).replace('.', ',')}% sobre a receita`
})

/**
 * Contas marcadas como fluxo de caixa no Omie (cFluxoCaixa = 'S'), que são as
 * que compõem o saldo do card. Conta vazia é a que o extrato ainda não trouxe;
 * fica fora para o número não prometer mais do que o saldo representa.
 */
const contasNoFluxo = computed(() => {
  const n = filtrosDisponiveis.contas_correntes.filter(c => c.fluxo_caixa === 'S').length
  return `${n} ${n === 1 ? 'conta' : 'contas'}`
})

/**
 * Cascata do resultado: do saldo em contas ao saldo projetado.
 *
 * Substituiu barras ancoradas no zero mais uma linha de acumulado. Ali a relação
 * entre um mês e o seguinte ficava por conta do leitor; aqui cada mês parte de
 * onde o anterior parou, e a escada É o acumulado — por isso a linha saiu, em vez
 * de ficar sobreposta dizendo a mesma coisa.
 *
 * Barra flutuante no Chart.js é `data: [[base, topo]]`. Não precisa de plugin.
 *
 * A aritmética dos degraus está em utils/cascata.ts, testada: um degrau fora do
 * lugar ainda parece um gráfico correto, e não dá para conferir no canvas.
 */
function buildChartAcum() {
  if (!canvasAcum.value || !dados.value) return
  chartAcum?.destroy()
  const c = chartColors()

  const passos = montarCascata(
    dados.value.cards.saldo_inicial,
    dados.value.grafico_resultado_acumulado,
    mesesVisiveisSet.value,
  )

  // Abertura e total em --accent, a cor distinta que marca posição (e não
  // variação) — é o papel que a barra Total tem num waterfall.
  const corDoPasso = (t: string) =>
    t === 'aumento' ? c.receita : t === 'reducao' ? c.despesa : c.acento


  chartAcum = new Chart(canvasAcum.value, {
    type: 'bar',
    data: {
      labels: passos.map(p => p.rotulo),
      datasets: [{
        label: 'Cascata',
        data: passos.map(p => [p.de, p.ate] as [number, number]),
        backgroundColor: passos.map(p => comAlfa(corDoPasso(p.tipo), ALFA_BARRA)),
        borderColor:     passos.map(p => corDoPasso(p.tipo)),
        borderWidth: 1.5, borderRadius: 4,
        ...PROPORCAO_UNICA,
      }],
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: c.tooltip, borderColor: c.tooltipBorder, borderWidth: 1,
          titleColor: c.tick, bodyColor: c.tooltipText, padding: 12,
          callbacks: {
            // O tooltip mostra o VALOR do passo, não o par base/topo que o
            // Chart.js traria por padrão — o par não significa nada para quem lê.
            label: ctx => {
              const p = passos[ctx.dataIndex]
              return p.tipo === 'abertura' || p.tipo === 'total'
                ? ` ${p.rotulo}: ${fmtK(p.ate)}`
                : ` ${p.rotulo}: ${fmtK(p.valor)}  (acumulado ${fmtK(p.ate)})`
            },
          },
        },
        datalabels: {
          // Degrau de altura zero é informação (o mês não mexeu no saldo), mas
          // um rótulo "R$ 0" flutuando ali só sujaria.
          display: ctx => passos[ctx.dataIndex].tipo !== 'aumento'
            || passos[ctx.dataIndex].valor !== 0,
          font: { family: c.fonte, size: 12, weight: 'bold' },
          formatter: (_v, ctx) => fmtK(valorDoPasso(passos[ctx.dataIndex])),
          anchor: 'end',
          // Acima quando positivo, abaixo quando negativo, nunca dentro da
          // barra. A regra e o porquê estão em utils/cascata.ts, com testes —
          // é o tipo de posição que não dá para conferir olhando o canvas.
          align: ctx => alinhamentoDoPasso(passos[ctx.dataIndex]),
          color: ctx => corDoPasso(passos[ctx.dataIndex].tipo),
          // 2px encostava o texto na barra; com o corner radius de 4px parecia
          // estar dentro dela.
          offset: 6,
        },
      },
      scales: {
        x: {
          grid: { display: false },
          ticks: { color: c.tick, font: { family: c.fonte, size: 10 } },
          border: { display: false },
        },
        y: {
          grid: { color: c.grid },
          ticks: { color: c.tick, font: { family: c.fonte, size: 10 }, callback: v => fmtK(Number(v)) },
          border: { display: false },
        },
      },
      // Folga para o rótulo, que agora é maior e fica 6px fora da barra. Sem
      // ela, o de cima e o de baixo saem cortados na borda do gráfico.
      layout: { padding: { top: 32, bottom: 32 } },
    },
  })
}

// Dispara imediatamente com o valor atual de auth.user (immediate: true).
// Se user já existe ao montar → chama carregar() de imediato.
// Se user ainda é null (edge case de timing) → aguarda até ser definido.
watch(() => auth.user, (user) => {
  if (user && !dados.value && !carregando.value) carregar()
}, { immediate: true })

// Reconstrói após dados atualizados (flush:post garante DOM pronto)
watch(dados, () => {
  setTimeout(() => {
    buildChartRecDesp()
    buildChartAcum()
    buildSparklines()
  }, 50)
}, { flush: 'post' })

/*
 * Reconstrói ao mudar o recorte de meses.
 *
 * Os cards são computeds e se atualizam sozinhos; os gráficos, não — vivem num
 * canvas e só mudam quando alguém os reconstrói. Sem este watcher, marcar um mês
 * atualizava os quatro cards e deixava a cascata e o realce das barras mostrando
 * o recorte anterior — as duas metades da tela discordando, sem erro nenhum.
 *
 * Só faz sentido na Visão Geral: na aba Resultado o recorte vai por prop ao
 * componente do pivô, que cuida da própria renderização. Não há sparkline aqui
 * de propósito — ela desenha a série do ano inteiro, que é o pano de fundo
 * contra o qual o recorte é lido.
 */
watch(mesesVisiveis, () => {
  if (aba.value !== 'geral' || !dados.value) return
  nextTick(() => { buildChartRecDesp(); buildChartAcum() })
})

// Reconstrói ao trocar tema.
//
// Observa ui.theme, não document.documentElement.getAttribute('data-theme'):
// getAttribute não tem dependência reativa, então o watcher anterior NUNCA
// disparava e os gráficos mantinham as cores do tema antigo até a próxima
// recarga. Só apareceu quando fomos medir.
watch(
  () => ui.theme,
  () => { if (dados.value) { nextTick(() => { buildChartRecDesp(); buildChartAcum(); buildSparklines() }) } }
)

// Reconstrói ao voltar da aba Resultado.
//
// As abas usam cadeia v-if, então ir para Resultado DESMONTA o .dash-content e
// destrói os <canvas>. Ao voltar, novos canvas são criados, mas as instâncias do
// Chart.js ainda apontam para os antigos — e o gráfico aparece vazio.
//
// Antes só o watch de `dados` reconstruía, e trocar de aba não altera os dados.
// Era por isso que recarregar a página ou limpar os filtros "consertava": ambos
// mudam `dados`.
watch(aba, novaAba => {
  if (novaAba !== 'geral' || !dados.value) return
  nextTick(() => {
    buildChartRecDesp()
    buildChartAcum()
    buildSparklines()
  })
})

// ── Carregamento ───────────────────────────────────────────────────────────
async function carregar() {
  if (!auth.user) {
    try { await auth.fetchMe() } catch {
      erro.value = 'Sessão inválida. Faça login novamente.'
      return
    }
  }
  // Só admin_global cai no primeiro grupo da lista — ele não tem grupo próprio e a
  // escolha é arbitrária por natureza. Para quem tem vários grupos, adivinhar seria
  // exibir dados de um cliente que a pessoa não selecionou: manda escolher.
  let grupoID = auth.user?.grupo_id
  if (!grupoID) {
    if (auth.isAdminGlobal) {
      grupoID = auth.meusGrupos[0]?.id
    } else if (auth.meusGrupos.length > 1) {
      router.push({ name: 'SelectGrupo', query: { redirect: route.fullPath } })
      return
    } else {
      grupoID = auth.meusGrupos[0]?.id
    }
  }
  if (!grupoID) {
    erro.value = 'Nenhum grupo associado. Faça login novamente.'
    return
  }
  grupoIDAtivo.value = grupoID

  carregando.value = true
  erro.value = ''

  try {
    // Opções e dados em paralelo. As opções vêm do endpoint próprio, em cascata:
    // antes as cinco consultas DISTINCT rodavam junto com a agregação pesada a cada
    // mudança de filtro.
    const [res, opcoes] = await Promise.all([
      fetchDashboard(grupoID, filtrosAtivos.value),
      fetchFiltros(grupoID, filtrosAtivos.value),
    ])

    dados.value = res
    Object.assign(filtrosDisponiveis, opcoes)

  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message
    erro.value = msg || 'Erro ao carregar dashboard'
  } finally {
    carregando.value = false
  }
}

let debounceTimer: ReturnType<typeof setTimeout>
function debouncedCarregar() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(carregar, 400)
}

function limparFiltros() {
  filtros.contas_correntes = []
  filtros.departamentos    = []
  filtros.categorias       = []
  filtros.empresas         = []
  filtros.cliente          = ''
  filtros.ano              = anoAtual
  // Volta ao padrão, não a "tudo visível": Transferência segue oculta.
  filtros.categorias_excluir = [CATEGORIA_OCULTA_PADRAO]
  carregar()
}

function toggleDropdown(key: string, event: MouseEvent) {
  if (dropdown.value === key) { dropdown.value = null; return }
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const spaceRight = window.innerWidth - rect.right
  fdPos.top      = rect.bottom + 4
  fdPos.left     = rect.left
  fdPos.right    = spaceRight < 270 ? window.innerWidth - rect.right : 0
  fdPos.useRight = spaceRight < 270
  dropdown.value = key
}

function closeDropdown() { dropdown.value = null }

function onClienteInput() {
  debouncedCarregar()
  // Mostra autocomplete se tiver 2+ chars
  if (filtros.cliente.trim().length >= 2) {
    dropdown.value = 'cli'
  } else {
    dropdown.value = null
  }
}

function onClienteFocus(event: FocusEvent) {
  if (filtros.cliente.trim().length >= 2) {
    const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
    fdPos.top      = rect.bottom + 4
    fdPos.left     = rect.left
    fdPos.useRight = false
    dropdown.value = 'cli'
  }
}

function onClienteBlur() {
  // Delay para permitir o mousedown na sugestão disparar antes do blur fechar
  setTimeout(() => { if (dropdown.value === 'cli') dropdown.value = null }, 150)
}

function selecionarCliente(nome: string) {
  filtros.cliente = nome
  dropdown.value  = null
  carregar()
}

/*
 * Esc sai do modo foco.
 *
 * Vive aqui por ser ancestral das duas abas que têm blocos focáveis, então um
 * listener cobre as duas. Antes a única saída era o botão dentro do próprio
 * modo: quem expandisse um bloco sem querer ficava sem rota de fuga óbvia.
 */
function aoTeclar(e: KeyboardEvent) {
  if (e.key === 'Escape' && ui.haFoco) ui.sairDoFoco()
}

onMounted(() => {
  document.addEventListener('click', closeDropdown)
  window.addEventListener('keydown', aoTeclar)
})

onBeforeUnmount(() => {
  chartRecDesp?.destroy()
  chartAcum?.destroy()
  chartSparkRec?.destroy()
  chartSparkDesp?.destroy()
  clearTimeout(debounceTimer)
  document.removeEventListener('click', closeDropdown)
  window.removeEventListener('keydown', aoTeclar)
})
</script>

<style scoped>
.dash-root { display: flex; flex-direction: column; min-height: 100%; }

/* ── Filtros na topbar (via Teleport) ──────────────────────────────────── */
.tf-inner {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  flex-wrap: nowrap;
  /* Rola em vez de recortar: na faixa própria há largura de sobra, e com clip
     os últimos filtros ficavam inalcançáveis em tela estreita. Os dropdowns
     usam position:fixed e escapam deste contêiner de qualquer forma. */
  overflow-x: auto;
  scrollbar-width: thin;
  padding: 10px 0;
  flex: 1;
}

/* Chip com o rótulo INLINE, não empilhado acima do controle: numa faixa fixa a
   altura é o recurso caro, e o rótulo em cima custava ~22px por filtro. */
.fi {
  display: inline-flex; align-items: center; gap: 7px;
  height: 34px; padding: 0 11px; flex-shrink: 0;
  border: 1px solid var(--border); border-radius: var(--r-sm);
  background: var(--surface);
  transition: border-color var(--transition), background var(--transition);
}
.fi:hover { border-color: var(--primary); background: var(--surface-2); }
.fi--grow { flex: 1; min-width: 150px; }

.fi-label {
  font-size: var(--fs-xs);
  font-weight: 600;
  letter-spacing: .06em;
  color: var(--text-dim);
  text-transform: uppercase;
  white-space: nowrap;
}

.fi-select,
.fi-input {
  font-size: var(--fs-sm);
  padding: 0;
  border-radius: 0;
  border: none;
  background: transparent;
  color: var(--text);
  outline: none;
  cursor: pointer;
  appearance: none;
  height: auto;
  white-space: nowrap;
  min-width: 0;
}
.fi-select:focus, .fi-input:focus { border-color: var(--primary); }
.fi-input::placeholder { color: var(--text-dim); }

.fi-multi { position: relative; }

.fi-trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-width: 80px;
  cursor: pointer;
  text-align: left;
}
.chv { font-size: var(--fs-xs); color: var(--text-dim); margin-left: 4px; }

/* Dropdown fixo — Teleportado para body, escapa de qualquer overflow:hidden */
.fd-fixed {
  position: fixed;
  min-width: 220px;
  max-width: 320px;
  max-height: 260px;
  overflow-y: auto;
  background: var(--surface);
  border: 1px solid var(--border-strong);
  border-radius: 8px;
  box-shadow: 0 8px 32px rgba(0,0,0,0.4);
  z-index: 9999;
  padding: 4px 0;
}

.fd-acoes {
  display: flex; gap: 6px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--border-strong);
  position: sticky; top: 0;
  background: var(--surface);
}
.fd-acao {
  flex: 1;
  background: var(--surface-2); color: var(--text-muted);
  border: 1px solid var(--border-strong); border-radius: 6px;
  padding: 4px 8px;
  font-family: var(--font-mono); font-size: var(--fs-xs); letter-spacing: 0.5px;
  cursor: pointer; transition: var(--transition); white-space: nowrap;
}
.fd-acao:hover { border-color: var(--primary); color: var(--primary); }

/* Cabeçalho de grupo de contas correntes. */
.fd-grupo {
  display: flex; align-items: center; justify-content: space-between; gap: 8px;
  padding: 8px 12px 5px;
  border-top: 1px solid var(--border);
}
.fd-grupo:first-child { border-top: none; }
.fd-grupo-rot {
  font-family: var(--font-mono); font-size: var(--fs-xs); letter-spacing: 1px;
  color: var(--text-dim); font-weight: 600; white-space: nowrap;
}
.fd-grupo-acoes { display: flex; gap: 4px; }
.fd-grupo-acoes .fd-acao { flex: 0 0 auto; padding: 3px 7px; font-size: var(--fs-xs); }

.chk-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  font-size: var(--fs-xs);
  color: var(--text-muted);
  cursor: pointer;
  transition: background 0.15s;
}
.chk-item:hover { background: var(--surface-2); }
.chk-item input { accent-color: var(--primary); cursor: pointer; flex-shrink: 0; }
.chk-item--click { user-select: none; }
.chk-item--click:hover { background: var(--surface-2); color: var(--primary); }

.fi-clear {
  padding: 4px 8px;
  height: 28px;
  border-radius: 6px;
  border: 1px solid var(--border-strong);
  background: var(--surface-2);
  color: var(--text-dim);
  cursor: pointer;
  font-size: var(--fs-xs);
  align-self: flex-end;
  flex-shrink: 0;
  transition: var(--transition);
}
.fi-clear:hover { border-color: var(--danger); color: var(--danger); }

/* ── Estados ───────────────────────────────────────────────────────────── */
.state-msg {
  display: flex; align-items: center; gap: 10px;
  justify-content: center; padding: 60px 24px;
  font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim);
}
.state-msg--erro { color: var(--danger); }

/* ── Conteúdo ──────────────────────────────────────────────────────────── */
.dash-content { display: flex; flex-direction: column; gap: 20px; }

/* As abas vivem na faixa fixa do cabeçalho (Teleport para #appbar-tabs). Sem
   borda inferior nem margem próprias: quem desenha a separação é o .appbar. */
.dash-tabs {
  display: flex; gap: var(--sp-1);
  overflow-x: auto;
  scrollbar-width: none;
}
.dash-tabs::-webkit-scrollbar { display: none; }
.dash-tab {
  background: none; border: none;
  border-bottom: 2px solid transparent;
  padding: 10px var(--sp-4); flex-shrink: 0;
  font-size: var(--fs-sm); font-weight: 600; letter-spacing: .02em;
  color: var(--text-muted); cursor: pointer; transition: var(--transition);
  white-space: nowrap;
}
.dash-tab:hover { color: var(--text); }
.dash-tab.active { color: var(--primary); border-bottom-color: var(--primary); }

.section-title {
  display: flex; align-items: center; gap: 8px;
  font-family: var(--font-mono); font-size: var(--fs-xs);
  color: var(--text-dim); letter-spacing: 2px; text-transform: uppercase;
  margin-bottom: 4px;
}
.section-title::before {
  content: ''; width: 18px; height: 1.5px;
  background: var(--primary); opacity: 0.7;
}

/* ── Cards KPI ─────────────────────────────────────────────────────────── */
.cards-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
}
@media (max-width: 1100px) { .cards-row { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 560px)  { .cards-row { grid-template-columns: 1fr; } }

.kpi-card {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--r); padding: 13px var(--sp-4);
  box-shadow: var(--shadow-sm);
  transition: background var(--transition), border-color var(--transition);
}
/* Sem faixa colorida de 2px no topo: com o icone ja marcando a cor de cada
   indicador, a faixa repetia a mesma informacao. Sem hover com translateY
   tambem -- o card nao e clicavel, e o movimento sugeria que fosse. */

.kpi-header { display: flex; align-items: center; gap: var(--sp-2); margin-bottom: var(--sp-2); }
.kpi-label {
  margin-right: auto;
  font-size: var(--fs-xs); font-weight: 600;
  letter-spacing: .06em; text-transform: uppercase; color: var(--text-dim);
}

.kpi-icon {
  width: 28px; height: 28px; border-radius: 8px;
  display: grid; place-items: center; flex-shrink: 0;
}
.kpi-icon :deep(svg) { width: 15px; height: 15px; stroke-width: 1.9; fill: none; }
.kpi-icon--green  { background: var(--success-weak); color: var(--success); }
.kpi-icon--red    { background: var(--danger-weak);  color: var(--danger); }
.kpi-icon--accent { background: var(--primary-weak); color: var(--primary); }
.kpi-icon--cyan   { background: var(--accent-weak);  color: var(--accent); }

/* Valor cheio, com centavos. A Geist Mono ja alinha por ser monoespacada; o
   `tnum` cobre o caso de a familia cair no fallback.
   fs-xl e nao fs-2xl: "R$ 1.171.900,00" tem 15 caracteres e a 36px nao caberia
   na coluna do grid de 4 cards. */
.kpi-value {
  font-family: var(--font-mono);
  font-feature-settings: "tnum" 1;
  font-size: var(--fs-xl);
  font-weight: 600; line-height: 1.15; letter-spacing: -.02em;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.kpi-value--green { color: var(--success); }
.kpi-value--red   { color: var(--danger); }

.kpi-foot {
  display: flex; align-items: center; gap: var(--sp-2);
  margin-top: 6px; font-size: var(--fs-xs); color: var(--text-muted);
}

/* O canvas precisa de um contêiner com altura definida: com responsive:true o
   Chart.js dimensiona pelo pai, e sem essa caixa a sparkline cresce sem limite. */
.spark-wrap { margin-top: 6px; height: 30px; position: relative; }
.spark-wrap canvas { position: absolute; inset: 0; }

.progress {
  margin-top: 8px; height: 5px; border-radius: var(--r-pill);
  background: var(--surface-2); overflow: hidden;
}
.progress i {
  display: block; height: 100%; border-radius: var(--r-pill);
  background: linear-gradient(90deg, var(--primary-line), var(--accent-line));
}

.tag {
  display: inline-flex; align-items: center; gap: 5px;
  font-size: var(--fs-xs); font-weight: 600;
  padding: 3px 9px; border-radius: var(--r-pill);
}
.tag--ok { background: var(--success-weak); color: var(--success); }

/* ── Gráficos ──────────────────────────────────────────────────────────── */
/* Uma coluna: cada gráfico ocupa a largura inteira. Lado a lado, as doze barras
   de Receita vs Despesa ficavam estreitas demais para serem alvo de clique, e a
   cascata não tinha espaço para os degraus mais o Total. */
.charts-row {
  display: grid;
  grid-template-columns: 1fr;
  gap: 18px;
}

.chart-card {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--r); padding: 20px 22px;
}
.chart-header {
  display: flex; align-items: flex-start; justify-content: space-between;
  margin-bottom: 16px; flex-wrap: wrap; gap: 8px;
}
.chart-title { font-size: var(--fs-base); font-weight: 700; }
.chart-sub   { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim); margin-top: 2px; }

.chart-legend {
  display: flex; align-items: center; gap: 10px;
  font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-muted); flex-wrap: wrap;
}
.leg-dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 3px; }

.chart-wrap { position: relative; height: 320px; }
/* O gráfico de Receita vs Despesa recorta a tela ao clique — o cursor precisa
   dizer isso antes de o usuário descobrir por acidente. */
.chart-card--clicavel .chart-wrap { cursor: pointer; }

.chart-limpar {
  background: var(--primary-weak); color: var(--primary);
  border: 1px solid var(--primary); border-radius: 20px;
  padding: 2px 10px; margin-right: var(--sp-3);
  font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: 600;
  cursor: pointer; transition: var(--transition); white-space: nowrap;
}
.chart-limpar:hover { background: var(--primary); color: var(--text-oncolor); }
</style>
