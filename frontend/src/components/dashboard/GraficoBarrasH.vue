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

    <div v-else class="bar-lista">
      <button
        v-for="(it, i) in itens" :key="it.rotulo"
        :class="['bar-item', {
          'bar-item--apagado': haSelecao && !selecao.has(it.rotulo),
          'bar-item--ativo': selecao.has(it.rotulo),
        }]"
        @click="alternar(it.rotulo)"
      >
        <span class="bar-nome" :title="it.rotulo">{{ it.rotulo }}</span>
        <span class="bar-trilho">
          <span
            class="bar-preenchimento"
            :style="{ width: largura(it), background: corDe(i) }"
          />
        </span>
        <span class="bar-valor">{{ fmtMoeda(it.valor) }}</span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { fmtMoeda } from '@/utils/formato'
import { corDe } from '@/utils/paleta'
import type { Agregado } from '@/utils/agregacao'
import { alternarRotulo } from '@/utils/fluxocruzado'
import BotaoFoco from '@/components/ui/BotaoFoco.vue'

const props = withDefaults(defineProps<{
  titulo: string
  subtitulo?: string
  itens: Agregado[]
  /** Rótulos selecionados. Vazio = nenhum recorte, e todas as barras contam. */
  selecionados?: string[]
  /** Id do bloco no modo foco. Sem ele, o botão de expandir não existe. */
  focoId?: string
}>(), { subtitulo: '', selecionados: () => [] })

const emit = defineEmits<{ (e: 'update:selecionados', v: string[]): void }>()

// Barras em CSS e não em Chart.js: são poucas, precisam de nome à esquerda e
// valor à direita alinhados, e cada linha é clicável. Um canvas dificultaria
// as três coisas sem nenhum ganho.
const selecao   = computed(() => new Set(props.selecionados))
const haSelecao = computed(() => selecao.value.size > 0)

const rotuloAcao = computed(() => haSelecao.value ? 'Limpar seleção' : 'Todos')

/*
 * A régua considera TODAS as barras, inclusive as fora do recorte.
 *
 * Elas continuam na tela — o clique recorta, não esconde —, e reescalar a cada
 * clique faria o mesmo comprimento representar valores diferentes de um
 * instante para o outro, que é o pior defeito possível num gráfico de barras.
 */
const maiorVisivel = computed(() => Math.max(0, ...props.itens.map(i => i.valor)))

function largura(it: Agregado): string {
  if (maiorVisivel.value <= 0) return '0%'
  return `${Math.max(2, (it.valor / maiorVisivel.value) * 100)}%`
}

function alternar(rotulo: string) {
  emit('update:selecionados', [...alternarRotulo(selecao.value, rotulo)])
}

function alternarTodos() {
  emit('update:selecionados', [])
}
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

.bar-lista { display: flex; flex-direction: column; gap: 3px; }
.bar-item {
  display: grid; grid-template-columns: minmax(0, 150px) minmax(0, 1fr) auto;
  gap: var(--sp-3); align-items: center;
  background: none; border: none; padding: 3px 5px; border-radius: 6px;
  cursor: pointer; text-align: left; transition: var(--transition);
}
.bar-item:hover { background: var(--surface-2); }
/* Realça o escolhido, sem riscar os demais: eles não saíram da conta, só não
   são o foco do recorte. */
.bar-item--apagado { opacity: 0.45; }
.bar-item--ativo { background: var(--primary-weak); }
.bar-item--ativo .bar-nome { color: var(--text); font-weight: 600; }
.bar-nome {
  font-size: var(--fs-sm); color: var(--text-muted);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.bar-trilho {
  height: 14px; background: var(--surface-2); border-radius: 5px; overflow: hidden;
}
.bar-preenchimento {
  display: block; height: 100%; border-radius: 5px;
  transition: width 0.25s ease;
}
.bar-valor {
  font-family: var(--font-body); font-size: var(--fs-xs); font-weight: 600;
  color: var(--text); white-space: nowrap;
}
</style>
