<template>
  <div :class="['cm', `cm--${msg.papel}`, { 'cm--erro': msg.erro }]">
    <div class="cm-bolha">
      <!--
        v-html com conteúdo do assistente.

        O HTML vem de renderizarMarkdown, que converte Markdown e sanitiza com
        lista de permissão estrita. É o ÚNICO v-html do projeto, e a razão está
        em utils/markdown.ts: nomes de categoria vêm do ERP preenchidos pelo
        cliente, então o texto que chega aqui não é confiável.

        A mensagem do usuário NÃO passa por markdown — ela é dele, e renderizar
        markdown no que ele digitou faria um asterisco virar formatação.
      -->
      <div v-if="msg.papel === 'assistente'" class="cm-texto" v-html="html" />
      <div v-else class="cm-texto cm-texto--user">{{ msg.conteudo }}</div>

      <ChatGrafico v-if="msg.grafico" :spec="msg.grafico" />

      <ChatFonte v-if="msg.fonte" :fonte="msg.fonte" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { renderizarMarkdown } from '@/utils/markdown'
import type { MensagemChat } from '@/api/ia'
import ChatGrafico from './ChatGrafico.vue'
import ChatFonte from './ChatFonte.vue'

const props = defineProps<{ msg: MensagemChat }>()

const html = computed(() => renderizarMarkdown(props.msg.conteudo))
</script>

<style scoped>
.cm {
  display: flex;
  margin-bottom: 14px;
}

.cm--usuario { justify-content: flex-end; }
.cm--assistente { justify-content: flex-start; }

.cm-bolha {
  max-width: 88%;
  padding: 10px 13px;
  border-radius: var(--r);
  font-size: var(--fs-sm);
  line-height: 1.55;
}

.cm--usuario .cm-bolha {
  background: var(--primary);
  color: var(--text-oncolor);
  border-bottom-right-radius: 4px;
}

.cm--assistente .cm-bolha {
  background: var(--surface-2);
  color: var(--text);
  border: 1px solid var(--border);
  border-bottom-left-radius: 4px;
}

/* Falha fica visível como falha, e não como resposta do assistente. */
.cm--erro .cm-bolha {
  background: var(--danger-weak);
  border-color: var(--danger-weak);
  color: var(--danger);
}

/* Texto do usuário preserva as quebras que ele digitou. */
.cm-texto--user { white-space: pre-wrap; word-break: break-word; }

/* ── Conteúdo vindo do markdown ──
   Os seletores são :deep porque o HTML é injetado por v-html e não carrega o
   atributo de escopo do componente. */
.cm-texto :deep(p) { margin: 0 0 8px; }
.cm-texto :deep(p:last-child) { margin-bottom: 0; }
.cm-texto :deep(strong) { font-weight: 600; color: var(--text); }
.cm-texto :deep(ul),
.cm-texto :deep(ol) { margin: 6px 0 8px; padding-left: 20px; }
.cm-texto :deep(li) { margin-bottom: 3px; }
.cm-texto :deep(code) {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 1px 5px;
  font-size: var(--fs-xs);
}
.cm-texto :deep(pre) {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  padding: 10px;
  overflow-x: auto;
  margin: 8px 0;
}
.cm-texto :deep(pre code) { border: none; background: none; padding: 0; }
.cm-texto :deep(a) { color: var(--primary-line); text-decoration: underline; }
.cm-texto :deep(h1),
.cm-texto :deep(h2),
.cm-texto :deep(h3),
.cm-texto :deep(h4) {
  font-size: var(--fs-base);
  font-weight: 600;
  margin: 10px 0 6px;
}
.cm-texto :deep(blockquote) {
  border-left: 2px solid var(--border-strong);
  padding-left: 10px;
  margin: 8px 0;
  color: var(--text-muted);
}

/*
 * Tabela dentro de um painel de 400px transborda com facilidade — três colunas
 * de valores já não cabem. O scroll horizontal fica NA tabela, e não na página:
 * sem isso o painel inteiro passa a rolar de lado.
 */
.cm-texto :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 8px 0;
  font-size: var(--fs-xs);
  display: block;
  overflow-x: auto;
}
.cm-texto :deep(th),
.cm-texto :deep(td) {
  border: 1px solid var(--border);
  padding: 5px 8px;
  text-align: left;
  white-space: nowrap;
}
.cm-texto :deep(th) {
  background: var(--surface);
  font-weight: 600;
  color: var(--text-muted);
}
</style>
