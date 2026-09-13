<template>
  <div class="pv">
    <div v-if="carregando" class="pv-state"><AppSpinner /> Carregando resultado...</div>
    <div v-else-if="erro" class="pv-state pv-state--erro">{{ erro }}</div>
    <div v-else-if="!dados || !dados.linhas.length" class="pv-state">Nenhum dado no período.</div>

    <template v-else>
      <!-- No modo foco os controles ficam atrás de um botão, como os filtros:
           a barra ocupa altura que a tabela pode usar melhor. -->
      <div class="pv-barra">
        <button class="pv-btn pv-btn--modo" @click="controlesAbertos = !controlesAbertos"
                :aria-expanded="controlesAbertos">
          Exibição<span class="pv-chv">▾</span>
        </button>

        <BotaoFoco :id="ID_FOCO" rotulo="Modo foco" />

        <span v-if="dados.mes_corte <= 12" class="pv-legenda">
          <span class="pv-chip-prev" /> a partir de {{ nomeMes[dados.mes_corte - 1] }} são valores previstos
        </span>
      </div>

      <div v-show="controlesAbertos" class="pv-toolbar">
        <button class="pv-btn" @click="expandirAte(1)">Recolher tudo</button>
        <button class="pv-btn" @click="expandirAte(2)">Categoria superior</button>
        <button class="pv-btn" @click="expandirAte(3)">Categoria final</button>
        <button class="pv-btn" @click="expandirAte(4)">Expandir tudo</button>
        <button class="pv-btn" @click="ui.toggleCentavos()"
                :title="ui.mostrarCentavos ? 'Ocultar centavos' : 'Mostrar centavos'">
          {{ ui.mostrarCentavos ? 'Sem centavos' : 'Com centavos' }}
        </button>
        <button class="pv-btn" @click="restaurarColunas">Restaurar colunas</button>
      </div>

      <div class="pv-scroll" ref="scrollEl"
           :style="alturaTabela ? { maxHeight: `${alturaTabela}px` } : undefined">
        <table class="pv-table">
          <thead>
            <tr>
              <th class="pv-th-dim" :style="estiloCol('dim', 280)">
                DESCRIÇÃO
                <span class="pv-alca" @mousedown="iniciarArrasto('dim', $event, larguras['dim'] ?? 280)" />
              </th>
              <th v-for="i in indicesVisiveis" :key="i"
                  :class="['pv-th-mes', { 'pv-previsto': i + 1 >= dados.mes_corte }]"
                  :style="estiloCol(`m${i}`, 92)">
                {{ nomeMes[i] }}
                <span class="pv-alca" @mousedown="iniciarArrasto(`m${i}`, $event, larguras[`m${i}`] ?? 92)" />
              </th>
              <th class="pv-th-total">TOTAL</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="n in linhasVisiveis" :key="n.id"
                :class="['pv-tr', `pv-nivel-${n.nivel}`,
                         { 'pv-tr-folha': n.nivel === 4, 'pv-fora': foraDoResultado(n) }]">
              <td class="pv-td-dim" :style="{ paddingLeft: `${12 + (n.nivel - 1) * 18}px` }">
                <button v-if="n.temFilhos" class="pv-toggle" @click="alternar(n.id)">
                  {{ expandidos.has(n.id) ? '−' : '+' }}
                </button>
                <span v-else class="pv-toggle pv-toggle--vazio" />
                <!-- Só a categoria superior alterna: é o nível em que faz sentido
                     tirar um bloco inteiro da conta, como Transferência. -->
                <button v-if="n.nivel === 2" class="pv-rotulo pv-rotulo--clicavel"
                        :title="foraDoResultado(n) ? 'Voltar a contar no resultado' : 'Não contar no resultado'"
                        @click="alternarNoResultado(n.tipo, n.rotulo)">{{ n.rotulo }}</button>
                <span v-else class="pv-rotulo">{{ n.rotulo }}</span>
              </td>
              <td v-for="i in indicesVisiveis" :key="i"
                  :class="['pv-td-num', `pv-${n.tipo}`,
                           { 'pv-previsto': i + 1 >= dados.mes_corte, 'pv-zero': n.meses[i] === 0 }]">
                {{ fmtPorTipo(n.meses[i], n.tipo) }}
              </td>
              <!-- Soma os meses visíveis, não n.total: aquele guarda o ano fechado
                   e mentiria embaixo de colunas recortadas. -->
              <td :class="['pv-td-total', `pv-${n.tipo}`]">
                {{ fmtPorTipo(totalVisivel(n.meses, mesesVisiveis), n.tipo) }}
              </td>
            </tr>
          </tbody>
          <tfoot>
            <tr class="pv-tfoot">
              <td class="pv-td-dim">
                RESULTADO
                <span class="pv-nota" title="Receita menos despesa do período. Não inclui o saldo das contas correntes, por isso difere do card RESULTADO da Visão Geral.">?</span>
              </td>
              <td v-for="i in indicesVisiveis" :key="i"
                  :class="['pv-td-num', { 'pv-previsto': i + 1 >= dados.mes_corte,
                                          'pv-neg': resultado.meses[i] < 0,
                                          'pv-pos': resultado.meses[i] > 0 }]">
                {{ resultado.meses[i] === 0 ? '—' : fmt(resultado.meses[i]) }}
              </td>
              <td class="pv-td-total"
                  :class="{ 'pv-neg': resultado.total < 0, 'pv-pos': resultado.total > 0 }">
                {{ fmt(resultado.total) }}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { fetchPivot, type PivotData, type PivotLinha } from '@/api/pivot'
import type { DashboardParams } from '@/api/dashboard'
import AppSpinner from '@/components/ui/AppSpinner.vue'
import BotaoFoco from '@/components/ui/BotaoFoco.vue'
import { fmtNumero } from '@/utils/formato'
import { useUiStore } from '@/stores/ui'
import { calcularResultado, chaveSuperior, larguraAoArrastar, alturaDisponivel, totalVisivel } from '@/utils/resultado'

const props = withDefaults(defineProps<{
  grupoId: string
  filtros: DashboardParams
  /** Recorte de exibição, fora de `filtros` para não disparar nova consulta. */
  meses?: number[]
}>(), { meses: () => [1,2,3,4,5,6,7,8,9,10,11,12] })

const mesesVisiveis = computed(() => new Set(props.meses))
/** Índices 0-11 das colunas exibidas, na ordem do ano. */
const indicesVisiveis = computed(() =>
  Array.from({ length: 12 }, (_, i) => i).filter(i => mesesVisiveis.value.has(i + 1)))

const ui = useUiStore()

/** Id deste bloco no modo foco. Ver utils/foco.ts. */
const ID_FOCO = 'resultado'
const emFoco = computed(() => ui.emFoco(ID_FOCO))

const nomeMes = ['Jan','Fev','Mar','Abr','Mai','Jun','Jul','Ago','Set','Out','Nov','Dez']

const dados      = ref<PivotData | null>(null)
const carregando = ref(false)
const erro       = ref('')
const expandidos = ref<Set<string>>(new Set())

/**
 * Categorias superiores fora da conta do RESULTADO. Estado de sessão, como a
 * legenda do donut: é uma lente de análise, não uma preferência — persistir faria
 * o usuário voltar dias depois a um resultado alterado sem lembrar por quê.
 */
const excluidosDoResultado = ref<Set<string>>(new Set())

// Controles recolhidos por padrão no modo foco, abertos fora dele: quem entra no
// foco quer área de tabela, quem está fora espera os botões onde sempre estiveram.
const controlesAbertos = ref(true)

/**
 * Larguras arrastadas, por coluna. Sessão apenas — o usuário dispensou persistir,
 * e "Restaurar colunas" cobre o arrependimento dentro da sessão.
 *
 * Chave 'dim' para a descrição e o índice do mês para as demais. O TOTAL fica de
 * fora: é a âncora de leitura à direita.
 */
const larguras = ref<Record<string, number>>({})

/**
 * Altura da tabela medida, e não estimada.
 *
 * Havia um `calc(100vh - 240px)` aqui. O 240 era um chute do que fica acima, mas
 * isso muda em três eixos independentes — filtros abertos ou fechados, modo foco
 * (que remove a faixa de abas) e controles recolhidos. No foco o número fixo
 * desperdiçava quase 80px justamente onde se quer área.
 *
 * innerHeight e não 100vh: em navegador móvel o vh ignora a barra de endereço e
 * estoura a tela.
 */
const scrollEl = ref<HTMLElement | null>(null)
const alturaTabela = ref(0)

function medirAltura() {
  const el = scrollEl.value
  if (!el) return
  alturaTabela.value = alturaDisponivel(el.getBoundingClientRect().top, window.innerHeight)
}

// Recalcula quando algo ACIMA da tabela muda de altura. Sem isto, abrir os
// filtros empurraria a tabela para fora da janela sem que ela encolhesse.
watch(() => [ui.filtrosAbertos, ui.foco, controlesAbertos.value, dados.value, props.meses],
  () => nextTick(medirAltura))

onMounted(() => {
  nextTick(medirAltura)
  window.addEventListener('resize', medirAltura)
})
onBeforeUnmount(() => window.removeEventListener('resize', medirAltura))

function restaurarColunas() {
  larguras.value = {}
}

/**
 * Arrasto na divisa do cabeçalho. Os listeners vão na `window`, não no elemento:
 * o ponteiro sai do `th` durante o movimento, e presos ao elemento o arrasto
 * morreria no meio.
 */
function iniciarArrasto(chave: string, ev: MouseEvent, larguraAtual: number) {
  ev.preventDefault()
  const x0 = ev.clientX
  const mover = (e: MouseEvent) => {
    larguras.value = { ...larguras.value, [chave]: larguraAoArrastar(larguraAtual, e.clientX - x0) }
  }
  const soltar = () => {
    window.removeEventListener('mousemove', mover)
    window.removeEventListener('mouseup', soltar)
    document.body.style.userSelect = ''
  }
  // Sem isto o arrasto seleciona o texto do cabeçalho junto.
  document.body.style.userSelect = 'none'
  window.addEventListener('mousemove', mover)
  window.addEventListener('mouseup', soltar)
}

const estiloCol = (chave: string, padrao: number) => ({
  width: `${larguras.value[chave] ?? padrao}px`,
})
watch(emFoco, foco => { controlesAbertos.value = !foco })

function alternarNoResultado(tipo: string, categoriaSuperior: string) {
  const k = chaveSuperior(tipo, categoriaSuperior)
  const s = new Set(excluidosDoResultado.value)
  s.has(k) ? s.delete(k) : s.add(k)
  excluidosDoResultado.value = s
}

const foraDoResultado = (n: No) =>
  n.nivel === 2 && excluidosDoResultado.value.has(chaveSuperior(n.tipo, n.rotulo))

/**
 * Recalculado na tela porque o servidor não conhece as exclusões do usuário. Com
 * nenhuma exclusão o valor é o mesmo que o backend envia — ver utils/resultado.ts.
 */
const resultado = computed(() =>
  calcularResultado(dados.value?.linhas ?? [], excluidosDoResultado.value, mesesVisiveis.value)
)

// ── Montagem da árvore ─────────────────────────────────────────────────────
// O backend entrega as folhas já pivotadas por mês. Os níveis superiores são
// somados aqui: não há ida ao servidor a cada expansão.
interface No {
  id: string
  nivel: 1 | 2 | 3 | 4
  rotulo: string
  meses: number[]
  total: number
  temFilhos: boolean
  paiId: string | null
  // Herdado da raiz da arvore. Receita e despesa chegam ambas como magnitude
  // positiva, entao o sinal nao distingue as duas — quem distingue e o tipo.
  tipo: 'receita' | 'despesa'
}

const zeros = () => Array(12).fill(0)

const arvore = computed<No[]>(() => {
  if (!dados.value) return []

  // Acumula por nível preservando a ordem de chegada (o SQL já ordena).
  const acc = new Map<string, No>()
  const ordem: string[] = []

  const somar = (id: string, nivel: No['nivel'], rotulo: string, paiId: string | null, l: PivotLinha) => {
    let n = acc.get(id)
    if (!n) {
      n = { id, nivel, rotulo, meses: zeros(), total: 0, temFilhos: nivel < 4, paiId,
            tipo: l.tipo as No['tipo'] }
      acc.set(id, n)
      ordem.push(id)
    }
    for (let i = 0; i < 12; i++) n.meses[i] += l.meses[i]
    n.total += l.total
  }

  for (const l of dados.value.linhas) {
    const i1 = `1|${l.tipo}`
    const i2 = `${i1}|${l.categoria_superior}`
    const i3 = `${i2}|${l.categoria_final}`
    const i4 = `${i3}|${l.cliente}`
    somar(i1, 1, l.tipo.toUpperCase(), null, l)
    somar(i2, 2, l.categoria_superior, i1, l)
    somar(i3, 3, l.categoria_final,    i2, l)
    somar(i4, 4, l.cliente,            i3, l)
  }

  // Reordena em profundidade para que cada nó apareça sob o seu pai.
  const porPai = new Map<string | null, No[]>()
  for (const id of ordem) {
    const n = acc.get(id)!
    const lista = porPai.get(n.paiId) ?? []
    lista.push(n)
    porPai.set(n.paiId, lista)
  }

  // Receita antes de despesa. O SQL ordena por `tipo`, e alfabeticamente "despesa"
  // vem primeiro — invertido em relação à leitura de um resultado.
  const pesoTipo = (rotulo: string) => (rotulo.toUpperCase() === 'RECEITA' ? 0 : 1)
  const raiz = porPai.get(null)
  if (raiz) raiz.sort((a, b) => pesoTipo(a.rotulo) - pesoTipo(b.rotulo))
  const saida: No[] = []
  const descer = (paiId: string | null) => {
    for (const n of porPai.get(paiId) ?? []) {
      saida.push(n)
      descer(n.id)
    }
  }
  descer(null)
  return saida
})

// Um nó aparece se todos os seus ancestrais estiverem expandidos.
const linhasVisiveis = computed(() =>
  arvore.value.filter(n => {
    let pai = n.paiId
    while (pai) {
      if (!expandidos.value.has(pai)) return false
      pai = arvore.value.find(x => x.id === pai)?.paiId ?? null
    }
    return true
  })
)

function alternar(id: string) {
  const s = new Set(expandidos.value)
  s.has(id) ? s.delete(id) : s.add(id)
  expandidos.value = s
}

function expandirAte(nivel: number) {
  expandidos.value = new Set(
    arvore.value.filter(n => n.nivel < nivel && n.temFilhos).map(n => n.id)
  )
}

// Notação contábil: negativo entre parênteses. Ver src/utils/formato.ts.
const fmt = (v: number) => fmtNumero(v, ui.mostrarCentavos ? 2 : 0)

/**
 * Despesa sempre entre parênteses, receita nunca. A matvw guarda as duas como
 * magnitude POSITIVA, então formatar pelo sinal deixava gasto e faturamento com
 * a mesma aparência — que é o que o usuário pediu para separar.
 *
 * A linha RESULTADO continua pelo sinal: ali o negativo é negativo de verdade.
 */
const fmtPorTipo = (v: number, tipo: No['tipo']) =>
  v === 0 ? '—' : fmt(tipo === 'despesa' ? -Math.abs(v) : Math.abs(v))

// ── Carregamento ───────────────────────────────────────────────────────────
async function carregar() {
  if (!props.grupoId) return
  carregando.value = true
  erro.value = ''
  try {
    dados.value = await fetchPivot(props.grupoId, props.filtros)
    expandirAte(2) // abre o primeiro nível: receita/despesa visíveis, resto fechado
  } catch (e: any) {
    erro.value = e?.response?.data?.message ?? 'Erro ao carregar resultado'
  } finally {
    carregando.value = false
  }
}

watch(() => [props.grupoId, props.filtros], carregar, { deep: true, immediate: true })
</script>

<style scoped>
.pv { display: flex; flex-direction: column; gap: 12px; }

.pv-state {
  padding: 48px; text-align: center;
  font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text-dim);
  display: flex; align-items: center; justify-content: center; gap: 10px;
}
.pv-state--erro { color: var(--danger); }

.pv-toolbar { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.pv-btn {
  background: var(--surface-2); color: var(--text-muted);
  border: 1px solid var(--border-strong); border-radius: 7px;
  padding: 5px 11px; font-family: var(--font-body); font-size: var(--fs-xs);
  letter-spacing: 0.5px; cursor: pointer; transition: var(--transition);
}
.pv-btn:hover { border-color: var(--primary); color: var(--primary); }

.pv-legenda {
  margin-left: auto; display: flex; align-items: center; gap: 6px;
  font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text-dim);
}
.pv-chip-prev {
  width: 10px; height: 10px; border-radius: 3px;
  background: var(--warning-weak); border: 1px solid var(--warning);
}

/* A tabela rola dentro do próprio container — a página nunca rola na horizontal. */
/*
 * A altura aqui é o que faz o cabeçalho grudar. `position: sticky` se ancora no
 * ancestral rolável mais próximo, e não na janela — e este contêiner virou esse
 * ancestral por ter overflow-x: auto (com um eixo diferente de `visible`, o
 * outro passa a `auto`). Sem altura própria ele crescia até caber a tabela
 * inteira, nunca rolava por dentro, e o `top: 0` do th grudava no topo do
 * contêiner, que subia junto com a página. Parecia que o sticky não existia.
 *
 * Ancorar na janela em vez disso exigiria um `top` igual à altura do cabeçalho
 * da aplicação, que varia com a barra de filtros aberta ou fechada — erraria a
 * cada vez que ela fosse alternada.
 *
 * O desconto de 240px cobre o pior caso do que fica acima: cabeçalho com
 * filtros abertos, respiro da página e a barra de botões de expandir.
 *
 * O piso de 320px vai dentro do max-height, e não em min-height: como piso do
 * teto, ele só garante altura mínima em tela baixa. Em min-height ele forçaria
 * 320px sempre, deixando faixa vazia embaixo quando a tabela está recolhida —
 * que é justamente o estado inicial dela.
 */
.pv-scroll {
  overflow: auto;
  /* Fallback até a primeira medição, e rede de segurança se ela falhar. O valor
     que vale é o inline, calculado por alturaDisponivel. */
  max-height: max(320px, calc(100vh - 240px));
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
}
.pv-table { width: 100%; border-collapse: collapse; min-width: 900px; }

.pv-table th {
  position: sticky; top: 0; z-index: 2;
  background: var(--surface);
  font-family: var(--font-body); font-size: var(--fs-xs); letter-spacing: 1px;
  color: var(--text-dim); font-weight: 600;
  padding: 10px 12px; border-bottom: 1px solid var(--border-strong);
  white-space: nowrap;
}
/* min-width fixo saiu: a largura agora vem do estilo inline, e um minimo em CSS
   venceria o arrasto para a esquerda. O piso vive em larguraAoArrastar. */
.pv-th-dim   { text-align: left; }

/* Alca de arrasto na divisa, com o cursor indicando a acao. Nao precisa de
   position:relative no th: ele ja e sticky por causa do cabecalho fixo, e sticky
   e posicionado — logo, ancora filhos absolutos. */
.pv-alca {
  position: absolute; top: 0; right: 0; width: 6px; height: 100%;
  cursor: col-resize; user-select: none;
}
.pv-alca:hover { background: var(--primary); opacity: 0.5; }

/* Barra de modo: fica sempre visivel, mesmo com os controles recolhidos. */
.pv-barra {
  display: flex; align-items: center; gap: var(--sp-2);
  flex-wrap: wrap; margin-bottom: var(--sp-2);
}
.pv-btn--modo { font-weight: 600; }
.pv-btn--sair {
  background: var(--danger-weak); color: var(--danger);
  border-color: var(--danger); font-weight: 700;
}
.pv-chv { margin-left: 5px; font-size: var(--fs-xs); }
.pv-th-mes   { text-align: right; }
.pv-th-total { text-align: right; border-left: 1px solid var(--border-strong); }

.pv-td-dim {
  padding: 7px 12px; font-size: var(--fs-xs); color: var(--text);
  white-space: nowrap; display: flex; align-items: center; gap: 6px;
}
.pv-td-num, .pv-td-total {
  padding: 7px 12px; text-align: right;
  font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text-muted);
  white-space: nowrap;
}
.pv-td-total { border-left: 1px solid var(--border); font-weight: 600; color: var(--text); }
.pv-zero { color: var(--text-dim); opacity: 0.45; }
/* Negativo em vermelho, somado aos parênteses da notação contábil. !important
   porque .pv-td-total define cor própria e a especificidade empata. */
/* Rodapé RESULTADO: cor pelo SINAL, ao contrário das linhas do corpo, onde ela
   vem do tipo. Aqui o negativo é negativo de verdade.

   O !important nas duas não é preguiça: `.pv-tfoot td` fixa color com
   especificidade (0,2,1), e uma classe sozinha (0,1,0) perderia a disputa e não
   pintaria nada. Zero fica de fora — é exibido como "—" e não teve resultado. */
.pv-neg { color: var(--danger) !important; }
.pv-pos { color: var(--success) !important; }
/* Cor pelo TIPO nas linhas: receita e faturamento, despesa e gasto, e as duas
   chegam como magnitude positiva. O rodape segue usando .pv-neg, porque ali o
   negativo e negativo de verdade. */
/* Mesmo tratamento da legenda do donut: o item continua legivel e com os proprios
   valores, so avisa que saiu da conta. */
.pv-fora .pv-rotulo { text-decoration: line-through; }
.pv-fora .pv-td-num, .pv-fora .pv-td-total { opacity: 0.45; }
.pv-rotulo--clicavel {
  background: none; border: none; padding: 0; font: inherit; color: inherit;
  cursor: pointer; text-align: left;
}
.pv-rotulo--clicavel:hover { text-decoration: underline; }

.pv-receita { color: var(--success); }
.pv-despesa { color: var(--danger); }
/* .pv-zero vence as duas: celula sem valor nao deve puxar atencao com cor. */
.pv-zero { color: var(--text-dim) !important; }
.pv-previsto { background: var(--warning-weak); }

.pv-tr { border-bottom: 1px solid var(--border); }
.pv-tr:hover td { background: var(--surface-2); }
.pv-nivel-1 .pv-td-dim { font-weight: 700; letter-spacing: 0.5px; }
.pv-nivel-1 td { background: var(--surface-2); }
.pv-nivel-2 .pv-td-dim { font-weight: 600; }
.pv-tr-folha .pv-td-dim { color: var(--text-muted); }

.pv-toggle {
  width: 16px; height: 16px; flex-shrink: 0;
  display: inline-flex; align-items: center; justify-content: center;
  background: var(--surface-2); border: 1px solid var(--border-strong); border-radius: 4px;
  color: var(--text-muted); font-family: var(--font-body); font-size: var(--fs-xs); line-height: 1;
  cursor: pointer; transition: var(--transition); padding: 0;
}
.pv-toggle:hover { border-color: var(--primary); color: var(--primary); }
.pv-toggle--vazio { background: none; border: none; cursor: default; }

.pv-rotulo { overflow: hidden; text-overflow: ellipsis; }

.pv-tfoot td {
  position: sticky; bottom: 0;
  background: var(--surface); border-top: 1px solid var(--border-strong);
  font-weight: 700; color: var(--text);
  font-family: var(--font-body); font-size: var(--fs-xs); padding: 10px 12px;
}
.pv-tfoot .pv-td-dim { font-size: var(--fs-xs); letter-spacing: 1px; }
.pv-nota {
  display: inline-flex; align-items: center; justify-content: center;
  width: 13px; height: 13px; margin-left: 6px;
  border: 1px solid var(--border-strong); border-radius: 50%;
  font-size: var(--fs-xs); color: var(--text-dim); cursor: help; font-weight: 400;
}
</style>
