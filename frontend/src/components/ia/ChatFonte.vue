<template>
  <!--
    "Mostre a conta".

    Toda resposta com número diz de onde ele veio: qual consulta e com quais
    filtros. É o recurso que faz alguém do financeiro confiar no valor — e o que
    permite conferir contra a tela quando os dois não batem.

    Fechado por padrão para não competir com a resposta; quem quer conferir,
    abre.
  -->
  <details class="cf">
    <summary class="cf-resumo">{{ rotulo }}</summary>
    <div class="cf-corpo">
      <div v-for="(valor, chave) in filtrosVisiveis" :key="chave" class="cf-linha">
        <span class="cf-chave">{{ chave }}</span>
        <span class="cf-valor">{{ valor }}</span>
      </div>
      <div v-if="!temFiltros" class="cf-linha cf-vazio">sem filtros</div>
    </div>
  </details>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FonteResposta } from '@/api/ia'

const props = defineProps<{ fonte: FonteResposta }>()

/** Nome técnico da ferramenta → o que ela faz, em português. */
const NOMES: Record<string, string> = {
  resumo_financeiro: 'Resumo financeiro do ano',
  abrir_por_categoria_ou_cliente: 'Abertura por categoria e cliente',
  fluxo_do_mes: 'Fluxo de caixa do mês',
  opcoes_de_filtro: 'Opções de filtro',
}

const rotulo = computed(() => NOMES[props.fonte.ferramenta] ?? props.fonte.ferramenta)

const filtrosVisiveis = computed(() => {
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(props.fonte.filtros ?? {})) {
    if (v === null || v === undefined || v === '') continue
    out[k] = String(v)
  }
  return out
})

const temFiltros = computed(() => Object.keys(filtrosVisiveis.value).length > 0)
</script>

<style scoped>
.cf {
  margin-top: 10px;
  border-top: 1px solid var(--border);
  padding-top: 7px;
}

.cf-resumo {
  font-size: var(--fs-xs);
  color: var(--text-dim);
  cursor: pointer;
  list-style: none;
  user-select: none;
}
/* Remove o triângulo nativo, que difere entre navegadores. */
.cf-resumo::-webkit-details-marker { display: none; }

.cf-resumo::before {
  content: '▸ ';
  display: inline-block;
  transition: transform var(--transition);
}
.cf[open] .cf-resumo::before { transform: rotate(90deg); }

.cf-resumo:hover { color: var(--text-muted); }

.cf-corpo {
  margin-top: 6px;
  padding-left: 14px;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.cf-linha {
  display: flex;
  gap: 8px;
  font-size: var(--fs-xs);
}

.cf-chave { color: var(--text-dim); }
.cf-valor { color: var(--text-muted); font-weight: 600; }
.cf-vazio { color: var(--text-dim); font-style: italic; }
</style>
