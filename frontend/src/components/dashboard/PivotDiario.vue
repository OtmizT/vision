<template>
  <div class="pd">
    <div class="pd-barra">
      <div>
        <div class="pd-title">RESULTADO INTRADIA</div>
        <div class="pd-sub">{{ subtitulo }}</div>
      </div>
      <div class="pd-toolbar">
        <button class="pv-btn" @click="expandirAte(1)">Recolher</button>
        <button class="pv-btn" @click="expandirAte(2)">Cat. superior</button>
        <button class="pv-btn" @click="expandirAte(3)">Cat. final</button>
        <button class="pv-btn" @click="expandirAte(4)">Tudo</button>
        <button class="pv-btn" @click="ui.toggleCentavos()">
          {{ ui.mostrarCentavos ? 'Sem centavos' : 'Com centavos' }}
        </button>
        <button class="pv-btn" @click="restaurarColunas">Restaurar colunas</button>
        <BotaoFoco :id="ID_FOCO" rotulo="Expandir" />
      </div>
    </div>

    <p v-if="!arvore.length" class="pd-vazio">Nenhum lançamento no recorte atual.</p>

    <div v-else class="pv-scroll" ref="scrollEl"
         :style="alturaTabela ? { maxHeight: `${alturaTabela}px` } : undefined">
      <table class="pv-table" :style="{ minWidth: `${280 + diasNoMes * 76}px` }">
        <thead>
          <tr>
            <th class="pv-th-dim" :style="estiloCol('dim', 280)">
              DESCRIÇÃO
              <span class="pv-alca" @mousedown="iniciarArrasto('dim', $event, larguras['dim'] ?? 280)" />
            </th>
            <th v-for="i in indices" :key="i"
                :class="['pv-th-dia', { 'pv-previsto': i + 1 > diaCorte }]"
                :style="estiloCol(`d${i}`, 76)">
              {{ i + 1 }}
              <span class="pv-alca" @mousedown="iniciarArrasto(`d${i}`, $event, larguras[`d${i}`] ?? 76)" />
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
              <!-- Mesma regra da aba Resultado: só a categoria superior sai da
                   conta, porque é o nível em que tirar um bloco inteiro faz sentido. -->
              <button v-if="n.nivel === 2" class="pv-rotulo pv-rotulo--clicavel"
                      :title="foraDoResultado(n) ? 'Voltar a contar no resultado' : 'Não contar no resultado'"
                      @click="alternarNoResultado(n.tipo, n.rotulo)">{{ n.rotulo }}</button>
              <span v-else class="pv-rotulo">{{ n.rotulo }}</span>
            </td>
            <td v-for="i in indices" :key="i"
                :class="['pv-td-num', `pv-${n.tipo}`,
                         { 'pv-previsto': i + 1 > diaCorte, 'pv-zero': n.dias[i] === 0 }]">
              {{ fmtPorTipo(n.dias[i], n.tipo) }}
            </td>
            <td :class="['pv-td-total', `pv-${n.tipo}`]">
              {{ fmtPorTipo(n.total, n.tipo) }}
            </td>
          </tr>
        </tbody>
        <tfoot>
          <tr class="pv-tfoot">
            <td class="pv-td-dim">RESULTADO</td>
            <td v-for="i in indices" :key="i"
                :class="['pv-td-num', { 'pv-previsto': i + 1 > diaCorte,
                                        'pv-neg': resultado.dias[i] < 0,
                                        'pv-pos': resultado.dias[i] > 0 }]">
              {{ resultado.dias[i] === 0 ? '—' : fmt(resultado.dias[i]) }}
            </td>
            <td class="pv-td-total"
                :class="{ 'pv-neg': resultado.total < 0, 'pv-pos': resultado.total > 0 }">
              {{ fmt(resultado.total) }}
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import type { FluxoTransacao } from '@/api/fluxocaixa'
import { fmtNumero } from '@/utils/formato'
import { useUiStore } from '@/stores/ui'
import { chaveSuperior, larguraAoArrastar, alturaDisponivel } from '@/utils/resultado'
import { montarPivotDiario, resultadoDiario, type NoDiario } from '@/utils/pivotdiario'
import BotaoFoco from '@/components/ui/BotaoFoco.vue'

/**
 * Tabela categoria × dia do mês, no padrão da aba Resultado.
 *
 * Não faz requisição: as transações já vêm carregadas pela aba de Fluxo de
 * Caixa, e recebê-las por prop é o que faz a tabela reagir instantaneamente ao
 * recorte cruzado — uma ida ao servidor a cada clique num dia ou numa categoria
 * tornaria a interação inútil.
 */
const props = defineProps<{
  transacoes: FluxoTransacao[]
  ano: number
  mes: number
  /** Respiro abaixo da tabela. Maior que o da aba Resultado — ver medirAltura. */
  folga?: number
}>()

const ui = useUiStore()

/** Id deste bloco no modo foco. Ver utils/foco.ts. */
const ID_FOCO = 'fc-intradia'

const diasNoMes = computed(() => new Date(props.ano, props.mes, 0).getDate())
const indices = computed(() => Array.from({ length: diasNoMes.value }, (_, i) => i))

/**
 * Último dia já ocorrido. Análogo ao `mes_corte` do pivô anual: dali para frente
 * o que aparece é provisão. Fora do mês corrente não há corte — mês passado é
 * todo realizado, mês futuro é todo previsto.
 */
const diaCorte = computed(() => {
  const h = new Date()
  if (h.getFullYear() === props.ano && h.getMonth() + 1 === props.mes) return h.getDate()
  const passado = props.ano < h.getFullYear()
    || (props.ano === h.getFullYear() && props.mes < h.getMonth() + 1)
  return passado ? diasNoMes.value : 0
})

const subtitulo = computed(() => {
  const n = props.transacoes.length
  return `${n} lançamento${n === 1 ? '' : 's'} · categoria × dia`
})

const expandidos = ref<Set<string>>(new Set())
const excluidosDoResultado = ref<Set<string>>(new Set())
const larguras = ref<Record<string, number>>({})

const arvore = computed(() => montarPivotDiario(props.transacoes, diasNoMes.value))

const resultado = computed(() =>
  resultadoDiario(arvore.value, excluidosDoResultado.value, diasNoMes.value))

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
    arvore.value.filter(n => n.nivel < nivel && n.temFilhos).map(n => n.id))
}

function alternarNoResultado(tipo: string, categoriaSuperior: string) {
  const k = chaveSuperior(tipo, categoriaSuperior)
  const s = new Set(excluidosDoResultado.value)
  s.has(k) ? s.delete(k) : s.add(k)
  excluidosDoResultado.value = s
}

const foraDoResultado = (n: NoDiario) =>
  n.nivel === 2 && excluidosDoResultado.value.has(chaveSuperior(n.tipo, n.rotulo))

// Abre até a categoria superior, como a aba Resultado. Reaplica quando a árvore
// muda de forma — um recorte pode fazer surgir categorias que não estavam ali.
watch(arvore, () => {
  if (!expandidos.value.size) expandirAte(2)
}, { immediate: true })

// ── Colunas redimensionáveis ───────────────────────────────────────────────
function restaurarColunas() {
  larguras.value = {}
}

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
  document.body.style.userSelect = 'none'
  window.addEventListener('mousemove', mover)
  window.addEventListener('mouseup', soltar)
}

const estiloCol = (chave: string, padrao: number) => ({
  width: `${larguras.value[chave] ?? padrao}px`,
})

// ── Altura medida ──────────────────────────────────────────────────────────
/**
 * Diferente da aba Resultado, aqui a tabela NÃO é o último elemento da página:
 * a listagem de transações vem abaixo. A folga maior que a de lá existe para que
 * a primeira faixa da listagem apareça — sem isso a tabela encosta na dobra e
 * nada indica que há mais conteúdo.
 *
 * 90px foi medido, não estimado: com 150 a tabela caía no piso de 260px numa
 * tela de 1080 e sobravam 131px de área vazia abaixo dela.
 */
const scrollEl = ref<HTMLElement | null>(null)
const alturaTabela = ref(0)

function medirAltura() {
  const el = scrollEl.value
  if (!el) return
  alturaTabela.value = alturaDisponivel(
    el.getBoundingClientRect().top, window.innerHeight, props.folga ?? 90, 260)
}

watch(() => [ui.filtrosAbertos, ui.foco, arvore.value, linhasVisiveis.value.length],
  () => nextTick(medirAltura))

onMounted(() => {
  nextTick(medirAltura)
  window.addEventListener('resize', medirAltura)
})
onBeforeUnmount(() => window.removeEventListener('resize', medirAltura))

// ── Formatação ─────────────────────────────────────────────────────────────
const fmt = (v: number) => fmtNumero(v, ui.mostrarCentavos ? 2 : 0)

/** Despesa sempre entre parênteses, receita nunca — ver ResultadoPivot.vue. */
const fmtPorTipo = (v: number, tipo: NoDiario['tipo']) =>
  v === 0 ? '—' : fmt(tipo === 'despesa' ? -Math.abs(v) : Math.abs(v))
</script>

<style scoped>
/* O visual é o mesmo da aba Resultado, de propósito: é a mesma leitura, num
   recorte de tempo diferente. Os nomes de classe `pv-*` foram mantidos para que
   a semelhança seja evidente ao comparar os dois arquivos. */
.pd { display: flex; flex-direction: column; gap: 10px; }

.pd-barra {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: var(--sp-4); flex-wrap: wrap;
}
.pd-title {
  font-family: var(--font-body); font-size: var(--fs-md); font-weight: 600;
  color: var(--text);
}
.pd-sub { font-size: var(--fs-xs); color: var(--text-dim); margin-top: 2px; }
.pd-vazio {
  font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text-dim);
  padding: 24px 0; text-align: center;
}
.pd-toolbar { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }

.pv-btn {
  background: var(--surface-2); color: var(--text-muted);
  border: 1px solid var(--border-strong); border-radius: 7px;
  padding: 5px 11px; font-family: var(--font-body); font-size: var(--fs-xs);
  letter-spacing: 0.5px; cursor: pointer; transition: var(--transition);
}
.pv-btn:hover { border-color: var(--primary); color: var(--primary); }

/* A altura é o que faz o cabeçalho grudar: `sticky` se ancora no ancestral
   rolável mais próximo, e sem altura própria este contêiner nunca rolaria por
   dentro. Ver a nota longa em ResultadoPivot.vue. O valor que vale é o inline,
   medido por alturaDisponivel; este é o fallback. */
.pv-scroll {
  overflow: auto;
  max-height: max(260px, calc(100vh - 420px));
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
}
.pv-table { width: 100%; border-collapse: collapse; }

.pv-table th {
  position: sticky; top: 0; z-index: 2;
  background: var(--surface);
  font-family: var(--font-body); font-size: var(--fs-xs); letter-spacing: 1px;
  color: var(--text-dim); font-weight: 600;
  padding: 10px 12px; border-bottom: 1px solid var(--border-strong);
  white-space: nowrap;
}

/*
 * Primeira coluna presa à esquerda — a diferença que 31 colunas impõem sobre as
 * 12 do pivô anual. Rolando até o fim do mês, sem isto se perde qual categoria
 * está sendo lida, e o número deixa de significar algo.
 *
 * O canto (th da descrição) precisa de z-index acima dos dois eixos, senão o
 * cabeçalho dos dias passa por cima dele ao rolar na horizontal. Fundo opaco é
 * obrigatório: sem ele o conteúdo rolado aparece por baixo.
 */
.pv-th-dim {
  text-align: left; left: 0; z-index: 3;
  border-right: 1px solid var(--border-strong);
}
.pv-td-dim {
  position: sticky; left: 0; z-index: 1;
  background: var(--surface);
  border-right: 1px solid var(--border-strong);
  padding: 7px 12px; font-size: var(--fs-xs); color: var(--text);
  white-space: nowrap; display: flex; align-items: center; gap: 6px;
}
.pv-tr:hover .pv-td-dim { background: var(--surface-2); }
.pv-tfoot .pv-td-dim { z-index: 3; }

.pv-alca {
  position: absolute; top: 0; right: 0; width: 6px; height: 100%;
  cursor: col-resize; user-select: none;
}
.pv-alca:hover { background: var(--primary); opacity: 0.5; }

.pv-th-dia   { text-align: right; }
.pv-th-total { text-align: right; border-left: 1px solid var(--border-strong); }

.pv-td-num, .pv-td-total {
  padding: 7px 12px; text-align: right;
  font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text-muted);
  white-space: nowrap;
}
.pv-td-total { border-left: 1px solid var(--border); font-weight: 600; color: var(--text); }

.pv-receita { color: var(--success); }
.pv-despesa { color: var(--danger); }
.pv-zero { color: var(--text-dim) !important; opacity: 0.45; }
.pv-previsto { background: var(--warning-weak); }

/* Rodapé RESULTADO: cor pelo SINAL, ao contrário do corpo, onde ela vem do tipo.
   O !important é necessário — `.pv-tfoot td` fixa color com especificidade
   (0,2,1) e uma classe sozinha (0,1,0) perderia a disputa. */
.pv-neg { color: var(--danger) !important; }
.pv-pos { color: var(--success) !important; }

.pv-fora .pv-rotulo { text-decoration: line-through; }
.pv-fora .pv-td-num, .pv-fora .pv-td-total { opacity: 0.45; }
.pv-rotulo--clicavel {
  background: none; border: none; padding: 0; font: inherit; color: inherit;
  cursor: pointer; text-align: left;
}
.pv-rotulo--clicavel:hover { text-decoration: underline; }

.pv-tr { border-bottom: 1px solid var(--border); }
.pv-tr:hover td { background: var(--surface-2); }
.pv-nivel-1 .pv-td-dim { font-weight: 700; letter-spacing: 0.5px; background: var(--surface-2); }
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
/* A célula do rodapé precisa voltar a table-cell. Como as do corpo, ela é flex
   por causa do botão de expandir — mas o rodapé não tem botão, e com display:flex
   ela deixava de grudar no fundo: os números do RESULTADO ficavam parados e só o
   rótulo subia com a rolagem, o que só acontece quando os dois eixos de sticky
   incidem na mesma célula. Medido, não suposto. */
.pv-tfoot .pv-td-dim {
  font-size: var(--fs-xs); letter-spacing: 1px;
  display: table-cell; vertical-align: middle;
}
</style>
