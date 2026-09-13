<template>
  <button
    :class="['bfoco', { 'bfoco--sair': ativo }]"
    :title="ativo ? 'Voltar à visão completa (Esc)' : 'Expandir para a página inteira'"
    :aria-pressed="ativo"
    @click="ui.alternarFoco(id)"
  >
    <span class="bfoco-ic">{{ ativo ? '✕' : '⤢' }}</span>
    <span v-if="rotulo" class="bfoco-tx">{{ ativo ? 'Sair' : rotulo }}</span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'

/**
 * Botão de expandir um bloco para a página inteira, na dinâmica do Power BI.
 *
 * Componente único em vez de markup repetido nos cinco blocos: são cinco cópias
 * do mesmo par de estados (⤢ / ✕) e do mesmo título, e a primeira manutenção as
 * faria divergir.
 *
 * O botão NÃO decide se o bloco aparece — isso é do pai, que precisa esconder os
 * irmãos. Aqui só mora o gesto.
 */
const props = defineProps<{
  /** Identificador do bloco. Único dentro da tela. */
  id: string
  /** Texto ao lado do ícone. Sem ele, o botão é só o ícone. */
  rotulo?: string
}>()

const ui = useUiStore()
const ativo = computed(() => ui.emFoco(props.id))
</script>

<style scoped>
/* Mesma linguagem dos botões de ação já existentes (.gr-acao, .pv-btn): fundo
   surface-2, borda forte, cantos de 7px. Repetido e não importado porque o CSS
   dos dois é scoped nos respectivos componentes. */
.bfoco {
  display: inline-flex; align-items: center; gap: 5px;
  background: var(--surface-2); color: var(--text-muted);
  border: 1px solid var(--border-strong); border-radius: 7px;
  padding: 4px 9px;
  font-family: var(--font-mono); font-size: var(--fs-xs);
  cursor: pointer; transition: var(--transition); white-space: nowrap;
}
.bfoco:hover { border-color: var(--primary); color: var(--primary); }

/* Sair usa a cor de perigo, como o "✕ Sair do modo foco" da aba Resultado já
   fazia: é a única saída visível e precisa se destacar do resto da barra. */
.bfoco--sair {
  background: var(--danger-weak); color: var(--danger);
  border-color: var(--danger); font-weight: 700;
}
.bfoco--sair:hover { border-color: var(--danger); color: var(--danger); }

.bfoco-ic { line-height: 1; }
</style>
