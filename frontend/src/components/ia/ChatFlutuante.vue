<template>
  <!--
    Teleport para o body, como o AppModal: dentro do MainLayout o painel herdaria
    o contexto de empilhamento da topbar e ficaria por baixo dela.
  -->
  <Teleport to="body">
    <!-- Botão flutuante. Some quando o grupo não tem o recurso ligado. -->
    <Transition name="fab">
      <button
        v-if="ia.disponivel && !ia.aberto"
        class="ia-fab"
        aria-label="Abrir o assistente"
        title="Pergunte ao VisiON"
        @click="ia.alternar()"
      >
        <img src="/chat-assistente.png" alt="" class="ia-fab-icone" />
      </button>
    </Transition>

    <Transition name="painel">
      <aside v-if="ia.disponivel && ia.aberto" class="ia-painel" role="dialog" aria-label="Assistente VisiON">
        <header class="ia-header">
          <img src="/chat-assistente.png" alt="" class="ia-header-icone" />
          <div class="ia-header-txt">
            <div class="ia-header-nome">Assistente</div>
            <div class="ia-header-sub">{{ nomeGrupo || 'dados do grupo' }}</div>
          </div>
          <button
            v-if="!ia.vazio"
            class="ia-acao"
            title="Limpar conversa"
            @click="limpar"
          >Limpar</button>
          <button class="ia-acao ia-fechar" title="Fechar (ESC)" @click="ia.fechar()">✕</button>
        </header>

        <div ref="corpoEl" class="ia-corpo">
          <div v-if="ia.carregando" class="ia-aviso">Carregando a conversa…</div>

          <div v-else-if="ia.vazio" class="ia-vazio">
            <img src="/chat-assistente.png" alt="" class="ia-vazio-icone" />
            <p class="ia-vazio-titulo">Pergunte ao VisiON</p>
            <p class="ia-vazio-texto">
              Respondo sobre os dados financeiros deste grupo. Os números vêm das
              mesmas consultas que montam o dashboard.
            </p>
            <div class="ia-sugestoes">
              <button
                v-for="s in SUGESTOES"
                :key="s"
                class="ia-sugestao"
                @click="enviar(s)"
              >{{ s }}</button>
            </div>
          </div>

          <template v-else>
            <ChatMensagem v-for="(m, i) in ia.mensagens" :key="m.id ?? i" :msg="m" />
          </template>

          <div v-if="ia.enviando" class="ia-pensando">
            <span class="ia-ponto" /><span class="ia-ponto" /><span class="ia-ponto" />
          </div>
        </div>

        <footer class="ia-rodape">
          <textarea
            ref="inputEl"
            v-model="texto"
            class="ia-input"
            rows="1"
            placeholder="Pergunte ao VisiON"
            :disabled="ia.enviando"
            @keydown.enter.exact.prevent="enviar()"
            @input="ajustarAltura"
          />
          <button
            class="ia-enviar"
            :disabled="!texto.trim() || ia.enviando"
            aria-label="Enviar"
            @click="enviar()"
          >
            <svg viewBox="0 0 24 24" width="17" height="17" fill="none"
                 stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M22 2 11 13M22 2l-7 20-4-9-9-4 20-7z" />
            </svg>
          </button>
        </footer>
      </aside>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onBeforeUnmount, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useIaStore } from '@/stores/ia'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import ChatMensagem from './ChatMensagem.vue'

const ia = useIaStore()
const auth = useAuthStore()
const ui = useUiStore()
const route = useRoute()

const texto = ref('')
const corpoEl = ref<HTMLElement | null>(null)
const inputEl = ref<HTMLTextAreaElement | null>(null)

const SUGESTOES = [
  'Como foi o resultado deste mês?',
  'Quais categorias mais pesaram na despesa?',
  'Tem algo vencendo nos próximos dias?',
]

const nomeGrupo = computed(() => auth.nomeGrupoAtivo)

/*
 * Contexto da tela.
 *
 * Vai como PADRÃO das ferramentas: sem ele, "e no mês passado?" não teria a
 * partir de quê, e a resposta viria sobre o mês corrente do servidor — que
 * pode não ser o que a pessoa está olhando.
 *
 * O período vem da store de UI, que o DashboardView mantém em dia. Em rotas
 * sem filtro de período ele fica na data de hoje, que é o padrão razoável.
 */
function contextoDaTela() {
  return {
    ano: ui.periodo.ano,
    mes: ui.periodo.mes,
    aba: String(route.name ?? ''),
  }
}

async function enviar(pergunta?: string) {
  const p = (pergunta ?? texto.value).trim()
  if (!p || ia.enviando) return

  texto.value = ''
  ajustarAltura()
  await ia.perguntar(p, contextoDaTela())
}

async function limpar() {
  await ia.limpar()
  inputEl.value?.focus()
}

/** O textarea cresce com o texto, até um teto — senão o campo come o painel. */
function ajustarAltura() {
  const el = inputEl.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 120) + 'px'
}

/** Rola para a última mensagem. Sem isso a resposta nova nasce fora da vista. */
async function rolarParaFim() {
  await nextTick()
  const el = corpoEl.value
  if (el) el.scrollTop = el.scrollHeight
}

watch(() => ia.mensagens.length, rolarParaFim)
watch(() => ia.enviando, rolarParaFim)

watch(() => ia.aberto, async aberto => {
  if (!aberto) return
  await rolarParaFim()
  inputEl.value?.focus()
})

/*
 * ESC fecha.
 *
 * O SyncDrawer tem um botão escrito "Fechar (ESC)" e nenhum listener em lugar
 * nenhum do projeto — a tecla nunca funcionou. Aqui funciona.
 */
function aoTeclar(e: KeyboardEvent) {
  if (e.key === 'Escape' && ia.aberto) ia.fechar()
}

onMounted(async () => {
  window.addEventListener('keydown', aoTeclar)
  await ia.verificarDisponibilidade()
  /*
   * O painel lembra que estava aberto, então depois de um F5 ele volta aberto —
   * e só `alternar()` carregava a conversa. O resultado era um painel aberto
   * mostrando "Pergunte ao VisiON" com a conversa inteira intacta no servidor.
   *
   * Precisa vir DEPOIS do await: a disponibilidade é o portão, e no momento do
   * onMounted ela ainda é false.
   */
  if (ia.aberto && ia.disponivel) await ia.carregarConversa()
})

onBeforeUnmount(() => window.removeEventListener('keydown', aoTeclar))

// Trocar de grupo troca o interlocutor: a disponibilidade e a conversa são
// por grupo, e manter a conversa anterior na tela mostraria números de um
// cliente no contexto de outro.
watch(() => auth.user?.grupo_id, async () => {
  ia.mensagens.length = 0
  await ia.verificarDisponibilidade()
  if (ia.aberto && ia.disponivel) await ia.carregarConversa()
})
</script>

<style scoped>
/* ── Botão flutuante ── */
.ia-fab {
  position: fixed;
  right: 22px;
  bottom: 22px;
  width: 56px;
  height: 56px;
  border-radius: 50%;
  border: 1px solid var(--border);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
  cursor: pointer;
  display: grid;
  place-items: center;
  padding: 0;
  z-index: 500;
  transition: transform var(--transition), box-shadow var(--transition);
}
.ia-fab:hover { transform: translateY(-2px) scale(1.04); }
.ia-fab-icone { width: 38px; height: 38px; display: block; }

/* ── Painel ── */
.ia-painel {
  position: fixed;
  top: 0; right: 0; bottom: 0;
  width: 400px;
  background: var(--bg);
  border-left: 1px solid var(--border);
  box-shadow: var(--shadow-lateral);
  display: flex;
  flex-direction: column;
  /* Acima do SyncDrawer (400) e abaixo do AppModal (1000): um modal aberto
     precisa cobrir o chat, e o chat precisa cobrir o drawer. */
  z-index: 600;
}

@media (max-width: 560px) {
  .ia-painel { width: 100%; }
}

.ia-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--border);
  background: var(--surface);
  flex-shrink: 0;
}
.ia-header-icone { width: 30px; height: 30px; flex-shrink: 0; }
.ia-header-txt { flex: 1; min-width: 0; }
.ia-header-nome { font-size: var(--fs-sm); font-weight: 600; color: var(--text); }
.ia-header-sub {
  font-size: var(--fs-xs); color: var(--text-dim);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

.ia-acao {
  background: none;
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-dim);
  font-size: var(--fs-xs);
  padding: 4px 9px;
  cursor: pointer;
  transition: var(--transition);
}
.ia-acao:hover { color: var(--text); border-color: var(--border-strong); }
.ia-fechar { padding: 4px 8px; }

.ia-corpo {
  flex: 1;
  overflow-y: auto;
  padding: 16px 14px;
}

.ia-aviso { color: var(--text-dim); font-size: var(--fs-xs); text-align: center; padding: 20px 0; }

/* ── Estado vazio ── */
.ia-vazio { text-align: center; padding: 28px 6px; }
.ia-vazio-icone { width: 64px; height: 64px; opacity: .85; margin-bottom: 12px; }
.ia-vazio-titulo {
  font-size: var(--fs-md); font-weight: 600; color: var(--text); margin: 0 0 6px;
}
.ia-vazio-texto {
  font-size: var(--fs-xs); color: var(--text-dim);
  line-height: 1.55; margin: 0 auto 18px; max-width: 300px;
}

.ia-sugestoes { display: flex; flex-direction: column; gap: 7px; }
.ia-sugestao {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  color: var(--text-muted);
  font-size: var(--fs-xs);
  padding: 9px 12px;
  cursor: pointer;
  text-align: left;
  transition: var(--transition);
}
.ia-sugestao:hover {
  border-color: var(--primary);
  color: var(--text);
  background: var(--primary-weak);
}

/* ── "Pensando" ── */
.ia-pensando { display: flex; gap: 4px; padding: 6px 4px 2px; }
.ia-ponto {
  width: 6px; height: 6px; border-radius: 50%;
  background: var(--text-dim);
  animation: pulsar 1.2s ease-in-out infinite;
}
.ia-ponto:nth-child(2) { animation-delay: .18s; }
.ia-ponto:nth-child(3) { animation-delay: .36s; }
@keyframes pulsar {
  0%, 100% { opacity: .25; transform: translateY(0); }
  50%      { opacity: 1;   transform: translateY(-3px); }
}

/* ── Rodapé ── */
.ia-rodape {
  display: flex;
  gap: 8px;
  align-items: flex-end;
  padding: 11px 14px;
  border-top: 1px solid var(--border);
  background: var(--surface);
  flex-shrink: 0;
}

.ia-input {
  flex: 1;
  background: var(--surface-2);
  border: 1px solid var(--border-strong);
  border-radius: var(--r-sm);
  color: var(--text);
  font-family: var(--font-body);
  font-size: var(--fs-sm);
  padding: 9px 11px;
  resize: none;
  outline: none;
  max-height: 120px;
  transition: border-color var(--transition);
}
.ia-input:focus { border-color: var(--primary); }
.ia-input::placeholder { color: var(--text-dim); }
.ia-input:disabled { opacity: .6; }

.ia-enviar {
  width: 36px; height: 36px;
  border-radius: var(--r-sm);
  border: none;
  background: var(--primary);
  color: var(--text-oncolor);
  cursor: pointer;
  display: grid;
  place-items: center;
  flex-shrink: 0;
  transition: var(--transition);
}
.ia-enviar:hover:not(:disabled) { background: var(--primary-hover); }
.ia-enviar:disabled { opacity: .4; cursor: default; }

/* ── Transições ── */
.painel-enter-active, .painel-leave-active { transition: transform .26s cubic-bezier(.4, 0, .2, 1); }
.painel-enter-from, .painel-leave-to { transform: translateX(100%); }

.fab-enter-active, .fab-leave-active { transition: transform .2s, opacity .2s; }
.fab-enter-from, .fab-leave-to { transform: scale(.7); opacity: 0; }

@media (prefers-reduced-motion: reduce) {
  .painel-enter-active, .painel-leave-active,
  .fab-enter-active, .fab-leave-active { transition: none; }
  .ia-ponto { animation: none; }
}
</style>
