<template>
  <section class="gr-card">
    <div class="gr-head">
      <div>
        <div class="gr-title">{{ titulo }}</div>
        <div class="gr-sub">{{ subtitulo }}</div>
      </div>
      <div class="gr-acoes">
        <button v-if="itens.length" class="gr-acao" @click="alternarTodos">
          {{ rotuloAcao }}
        </button>
        <!-- Só aparece quando o pai pede: as abas de contas montam este mesmo
             componente sem focoId e seguem sem o botão. -->
        <BotaoFoco v-if="focoId" :id="focoId" />
      </div>
    </div>

    <p v-if="!itens.length" class="gr-vazio">Nada a exibir no período.</p>

    <div v-else class="gr-corpo">
      <div class="gr-canvas-wrap">
        <canvas ref="canvasEl" />
        <div class="gr-centro">
          <div class="gr-centro-rot">TOTAL</div>
          <div class="gr-centro-val">{{ totalTexto }}</div>
        </div>
      </div>

      <div class="gr-legenda">
        <button
          v-for="(it, i) in itens" :key="it.rotulo"
          :class="['leg-item', {
            'leg-item--apagado': haSelecao && !selecao.has(it.rotulo),
            'leg-item--ativo': selecao.has(it.rotulo),
          }]"
          @click="alternar(it.rotulo)"
        >
          <span class="leg-cor" :style="{ background: corDe(i) }" />
          <span class="leg-rot" :title="it.rotulo">{{ it.rotulo }}</span>
          <span class="leg-val">{{ fmtMoeda(it.valor) }}</span>
          <span class="leg-pct">{{ pct(it.valor) }}</span>
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { Chart } from 'chart.js'
import { fmtMoeda, fmtCompacto } from '@/utils/formato'
import { corDe, corDeAlpha } from '@/utils/paleta'
import { totalMarcados, type Agregado } from '@/utils/agregacao'
import { alternarRotulo } from '@/utils/fluxocruzado'
import BotaoFoco from '@/components/ui/BotaoFoco.vue'

const props = withDefaults(defineProps<{
  titulo: string
  subtitulo?: string
  itens: Agregado[]
  /**
   * Rótulos selecionados. Vazio = nenhum recorte.
   *
   * O clique RECORTA a tela inteira, não esconde a fatia. Houve um segundo modo
   * aqui, em que clicar removia o item do gráfico e do total, e as três abas que
   * montam este componente ficaram com gestos opostos no mesmo lugar. Restou um.
   */
  selecionados?: string[]
  /** Id do bloco no modo foco. Sem ele, o botão de expandir não existe. */
  focoId?: string
}>(), { subtitulo: '', selecionados: () => [] })

const emit = defineEmits<{ (e: 'update:selecionados', v: string[]): void }>()

const selecao   = computed(() => new Set(props.selecionados))
const haSelecao = computed(() => selecao.value.size > 0)

const canvasEl = ref<HTMLCanvasElement | null>(null)
let chart: Chart | null = null

/**
 * O total acompanha o recorte, e sem recorte é o período inteiro — assim o
 * número no centro sempre corresponde ao que a tela está mostrando abaixo.
 */
const totalGeral = computed(() => props.itens.reduce((s, i) => s + i.valor, 0))

const total = computed(() =>
  haSelecao.value ? totalMarcados(props.itens, selecao.value) : totalGeral.value
)

const rotuloAcao = computed(() => haSelecao.value ? 'Limpar seleção' : 'Todas as categorias')

/**
 * fmtCompacto devolve string vazia para zero — correto em rotulo de grafico,
 * onde "R$ 0" sobre barra sem altura so suja. No centro do donut o vazio parece
 * falha de renderizacao, entao aqui zero e escrito.
 */
const totalTexto = computed(() => total.value === 0 ? 'R$ 0' : fmtCompacto(total.value))

/**
 * Percentual sobre o TOTAL GERAL, não sobre o total do recorte.
 *
 * Dividindo pelo recorte, a categoria escolhida apareceria como 100% e as
 * demais somariam mais de 100% entre si — cada uma dividida pelo valor da
 * escolhida. A fatia de cada categoria no período não muda por causa do
 * recorte, e o percentual precisa dizer isso.
 */
function pct(valor: number): string {
  return totalGeral.value === 0 ? '—' : `${((valor / totalGeral.value) * 100).toFixed(1)}%`
}

function alternar(rotulo: string) {
  emit('update:selecionados', [...alternarRotulo(selecao.value, rotulo)])
}

function alternarTodos() {
  emit('update:selecionados', [])
}

function desenhar() {
  if (!canvasEl.value) return
  chart?.destroy()

  // Nenhuma fatia sai do donut: ele é o ponto de partida do próximo clique, e
  // remover as outras deixaria o usuário sem como comparar ou voltar. O recorte
  // aparece como esmaecimento.
  chart = new Chart(canvasEl.value, {
    type: 'doughnut',
    data: {
      labels: props.itens.map(i => i.rotulo),
      datasets: [{
        data: props.itens.map(i => i.valor),
        backgroundColor: props.itens.map((i, idx) => {
          const esmaecida = haSelecao.value && !selecao.value.has(i.rotulo)
          return esmaecida ? corDeAlpha(idx, 0.18) : corDe(idx)
        }),
        borderColor: 'transparent',
        borderWidth: 0,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      cutout: '68%',
      plugins: {
        legend: { display: false },   // a legenda lateral é interativa e substitui esta
        datalabels: { display: false },
        tooltip: {
          callbacks: { label: c => ` ${c.label}: ${fmtMoeda(Number(c.raw))}` },
        },
      },
    },
  })
}

watch(() => [props.itens, props.selecionados], () => nextTick(desenhar), { deep: true })
onMounted(() => nextTick(desenhar))
onBeforeUnmount(() => chart?.destroy())
</script>

<style scoped>
.gr-card {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--r); padding: var(--sp-5);
}
.gr-head {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: var(--sp-4); margin-bottom: var(--sp-4); flex-wrap: wrap;
}
.gr-title {
  font-family: var(--font-body); font-size: var(--fs-md); font-weight: 600;
  color: var(--text);
}
.gr-sub { font-size: var(--fs-xs); color: var(--text-dim); margin-top: 2px; }
.gr-vazio { font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text-dim); padding: 24px 0; }
.gr-acao {
  background: var(--surface-2); color: var(--text-muted);
  border: 1px solid var(--border-strong); border-radius: 7px;
  padding: 4px 9px; font-family: var(--font-body); font-size: var(--fs-xs);
  cursor: pointer; transition: var(--transition); white-space: nowrap;
}
.gr-acao:hover { border-color: var(--primary); color: var(--primary); }
/* Agrupa os botões do cabeçalho. Sem o wrapper, o segundo botão viraria um
   terceiro filho do .gr-head, que é space-between, e ficaria solto no meio. */
.gr-acoes { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }

.gr-corpo {
  display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: var(--sp-5); align-items: center;
}
@media (max-width: 640px) { .gr-corpo { grid-template-columns: 1fr; } }

.gr-canvas-wrap {
  position: relative; width: 100%;
  aspect-ratio: 1 / 1; max-height: 300px; margin: 0 auto;
}
/* O canvas sai do fluxo: no fluxo, a altura que o Chart.js atribui a ele
   realimenta a altura do contêiner e vence o aspect-ratio — o donut saía
   245x300 em vez de quadrado. Absoluto, quem manda na caixa é o aspect-ratio. */
.gr-canvas-wrap canvas { position: absolute; inset: 0; }
.gr-centro {
  position: absolute; inset: 0; display: flex; flex-direction: column;
  align-items: center; justify-content: center; pointer-events: none;
}
.gr-centro-rot { font-family: var(--font-body); font-size: var(--fs-xs); letter-spacing: 1.5px; color: var(--text-dim); }
.gr-centro-val {
  font-family: var(--font-body); font-feature-settings: "tnum" 1;
  font-size: var(--fs-xl); font-weight: 700; letter-spacing: -.02em; color: var(--text);
}

.gr-legenda { display: flex; flex-direction: column; gap: 2px; max-height: 300px; overflow-y: auto; }
.leg-item {
  display: grid; grid-template-columns: 11px minmax(0, 1fr) auto 52px;
  gap: 9px; align-items: center;
  background: none; border: none; padding: 4px 6px; border-radius: 6px;
  cursor: pointer; text-align: left; transition: var(--transition);
}
.leg-item:hover { background: var(--surface-2); }
/* Realça o escolhido em vez de riscar. Sem line-through de propósito: o item
   não foi removido da conta, só não é o foco do recorte. */
.leg-item--apagado { opacity: 0.45; }
.leg-item--ativo { background: var(--primary-weak); }
.leg-item--ativo .leg-rot { color: var(--text); font-weight: 600; }
.leg-cor { width: 11px; height: 11px; border-radius: 3px; }
.leg-rot {
  font-size: var(--fs-sm); color: var(--text-muted);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.leg-val { font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text); white-space: nowrap; }
.leg-pct { font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text-dim); text-align: right; }
</style>
