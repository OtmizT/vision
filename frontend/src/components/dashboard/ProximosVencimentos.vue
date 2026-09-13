<template>
  <div class="pvv">
    <div class="fc-card-title">Próximos vencimentos</div>
    <div class="fc-card-sub">A partir de hoje, em qualquer mês</div>
    <p v-if="!itens.length" class="fc-vazio">Nada previsto adiante.</p>
    <div v-else class="venc-lista">
      <div v-for="(t, i) in itens" :key="i" class="venc-item">
        <div class="venc-data">{{ t.data.slice(0, 5) }}</div>
        <div class="venc-desc">
          <span class="venc-nome">{{ t.descricao }}</span>
          <span class="venc-cat">{{ t.categoria }}</span>
        </div>
        <div class="venc-val" :class="t.tipo === 'receita' ? 'res-val--in' : 'res-val--out'">
          {{ fmtMoeda(t.valor) }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { fmtMoeda } from '@/utils/formato'
import type { FluxoTransacao } from '@/api/fluxocaixa'

/**
 * Lista de provisões futuras. Extraída de FluxoCaixa.vue junto com o resumo,
 * pelo mesmo motivo: a aba Fluxo de Caixa a exibe dentro do bloco recolhível, e
 * as abas de contas na coluna lateral.
 *
 * Ignora de propósito o recorte cruzado e o mês selecionado — é um aviso do que
 * vem, e recortá-lo por um dia do mês passado o deixaria sempre vazio.
 */
defineProps<{ itens: FluxoTransacao[] }>()
</script>

<style scoped>
.pvv { display: flex; flex-direction: column; min-height: 0; flex: 1; }

.fc-card-title {
  font-family: var(--font-mono); font-size: var(--fs-md); font-weight: 600;
  color: var(--text);
}
.fc-card-sub { font-size: var(--fs-xs); color: var(--text-dim); margin-top: 2px; }
.fc-vazio { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim); padding: 16px 0; }

/* A lista rola em vez de esticar o card com o volume de títulos. A rolagem fica
   na lista, não no card, para o título continuar visível.

   flex-basis 150px, não auto: é o basis que a grade usa para dimensionar a
   linha. Com basis auto a lista inteira entrava na conta e esticava o
   calendário junto. O grow faz a lista preencher a altura que sobrar. */
.venc-lista { flex: 1 1 150px; min-height: 0; overflow-y: auto; padding-right: 6px; }
.venc-item {
  display: grid; grid-template-columns: 38px 1fr auto; gap: 8px; align-items: center;
  padding: 5px 0; border-bottom: 1px solid var(--border);
}
.venc-item:last-child { border-bottom: none; }
.venc-data { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim); }
.venc-desc { min-width: 0; display: flex; flex-direction: column; }
.venc-nome {
  font-size: var(--fs-xs); line-height: 1.3; color: var(--text);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.venc-cat {
  font-family: var(--font-mono); font-size: var(--fs-xs); line-height: 1.3; color: var(--text-dim);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.venc-val { font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: 600; white-space: nowrap; }
.res-val--in  { color: var(--success); }
.res-val--out { color: var(--danger); }
</style>
