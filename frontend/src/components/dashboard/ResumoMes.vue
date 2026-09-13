<template>
  <div class="rm">
    <div class="fc-card-title">{{ titulo }}</div>
    <div class="fc-card-sub">{{ subtitulo }}</div>

    <template v-if="tipo !== 'despesa'">
      <div class="res-linha">
        <span class="res-rot">Recebido</span>
        <span class="res-val res-val--in">{{ fmtMoeda(resumo.recebido) }}</span>
      </div>
      <div class="res-linha">
        <span class="res-rot">A receber<span class="res-prev">previsto</span></span>
        <span class="res-val res-val--in">{{ fmtMoeda(resumo.a_receber) }}</span>
      </div>
      <!-- So aparece quando ha atraso: com a inadimplencia desligada o
           valor e zero e a linha nao deve ocupar espaco. -->
      <div v-if="resumo.atrasado_receber > 0" class="res-linha">
        <span class="res-rot">A receber<span class="res-atraso">atrasado</span></span>
        <span class="res-val res-val--atraso">{{ fmtMoeda(resumo.atrasado_receber) }}</span>
      </div>
    </template>
    <template v-if="tipo !== 'receita'">
      <div class="res-linha">
        <span class="res-rot">Pago</span>
        <span class="res-val res-val--out">{{ fmtMoeda(resumo.pago) }}</span>
      </div>
      <div class="res-linha">
        <span class="res-rot">A pagar<span class="res-prev">previsto</span></span>
        <span class="res-val res-val--out">{{ fmtMoeda(resumo.a_pagar) }}</span>
      </div>
      <div v-if="resumo.atrasado_pagar > 0" class="res-linha">
        <span class="res-rot">A pagar<span class="res-atraso">atrasado</span></span>
        <span class="res-val res-val--atraso">{{ fmtMoeda(resumo.atrasado_pagar) }}</span>
      </div>
    </template>
    <div class="res-linha res-linha--total">
      <span class="res-rot">{{ tipo === 'todos' ? 'Resultado' : 'Total' }}</span>
      <span class="res-val" :class="resumo.resultado < 0 ? 'res-val--out' : 'res-val--in'">
        {{ fmtMoeda(resumo.resultado) }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { fmtMoeda } from '@/utils/formato'
import type { FluxoResumo } from '@/api/fluxocaixa'

/**
 * Bloco de resumo do fluxo de caixa.
 *
 * Extraído de FluxoCaixa.vue porque a aba Fluxo de Caixa passou a exibi-lo em
 * dois lugares diferentes — no bloco recolhível e, antes, na coluna lateral das
 * abas de contas. Duplicar o markup faria as duas versões divergirem, e a linha
 * de atrasado, que é recente, é justamente o tipo de detalhe que ficaria só numa.
 */
defineProps<{
  resumo: FluxoResumo
  tipo: 'todos' | 'receita' | 'despesa'
  titulo: string
  subtitulo: string
}>()
</script>

<style scoped>
.fc-card-title {
  font-family: var(--font-mono); font-size: var(--fs-md); font-weight: 600;
  color: var(--text);
}
.fc-card-sub { font-size: var(--fs-xs); color: var(--text-dim); margin-top: 2px; }

.res-linha {
  display: flex; align-items: center; justify-content: space-between;
  padding: 7px 0; border-bottom: 1px solid var(--border);
}
.res-linha--total { border-bottom: none; border-top: 1px solid var(--border-strong); margin-top: 4px; padding-top: 10px; }
.res-rot { font-size: var(--fs-xs); color: var(--text-muted); display: flex; align-items: center; gap: 6px; }
.res-prev {
  font-family: var(--font-mono); font-size: var(--fs-xs); letter-spacing: 0.5px;
  padding: 1px 5px; border-radius: 10px;
  background: var(--warning-weak); color: var(--warning);
}
/* Atrasado usa a cor de perigo, e nao o ambar do previsto: vencido nao e a
   mesma coisa que "ainda vai vencer". Mesma distincao ja feita na pill da
   listagem. */
.res-atraso {
  font-family: var(--font-mono); font-size: var(--fs-xs); letter-spacing: 0.5px;
  padding: 1px 5px; border-radius: 10px;
  background: var(--danger-weak); color: var(--danger);
}
.res-val { font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: 600; }
.res-val--in     { color: var(--success); }
.res-val--out    { color: var(--danger); }
.res-val--atraso { color: var(--danger); font-weight: 700; }
.res-linha--total .res-val { font-size: var(--fs-base); }
</style>
