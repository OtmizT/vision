<template>
  <div class="select-grupo-page">
    <div class="card">
      <div class="logo">
        <div class="logo-icon">
          <span class="logo-marca" aria-hidden="true" />
          <span class="logo-letra" aria-hidden="true">V</span>
        </div>
        <div class="logo-text">Visi<span>ON</span></div>
      </div>

      <h2 class="title">{{ podePlataforma ? 'Selecionar Contexto' : 'Selecionar Grupo' }}</h2>
      <p class="subtitle">{{ subtitulo }}</p>

      <div v-if="loading" class="loading">Carregando...</div>

      <div v-else-if="!podePlataforma && grupos.length === 0" class="empty">
        Nenhum grupo disponível.
      </div>

      <div v-else class="grupos-list">
        <!-- Plataforma primeiro: e o lugar onde o admin global trabalha. Sem
             esta opcao ele entraria num grupo e o painel dele ficaria sem
             caminho de volta que nao fosse deslogar. -->
        <button
          v-if="podePlataforma"
          class="grupo-btn plataforma-btn"
          :class="{ selected: selectedId === PLATAFORMA, loading: selecting === PLATAFORMA }"
          :disabled="!!selecting"
          @click="entrarNaPlataforma"
        >
          <div class="grupo-icon plataforma-icon">P</div>
          <div class="grupo-info">
            <div class="grupo-nome">Plataforma</div>
            <div class="grupo-slug">administracao de todos os grupos</div>
          </div>
          <div v-if="selecting === PLATAFORMA" class="spinner" />
        </button>

        <button
          v-for="g in grupos"
          :key="g.id"
          class="grupo-btn"
          :class="{ selected: selectedId === g.id, loading: selecting === g.id }"
          :disabled="!!selecting"
          @click="handleSelect(g.id)"
        >
          <div class="grupo-icon">{{ g.nome[0].toUpperCase() }}</div>
          <div class="grupo-info">
            <div class="grupo-nome">{{ g.nome }}</div>
            <div class="grupo-slug">{{ g.slug }}</div>
          </div>
          <div v-if="selecting === g.id" class="spinner" />
        </button>
      </div>

      <div v-if="error" class="error">{{ error }}</div>

      <button v-if="isTroca" class="cancel-btn" @click="cancelar">Cancelar</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore, type GrupoInfo } from '@/stores/auth'
import { destinoAposEntrar } from '@/utils/navegacao'

const auth     = useAuthStore()
const router   = useRouter()
const route    = useRoute()

// isTroca = usuário já autenticado quer trocar de grupo
const isTroca  = computed(() => !!auth.accessToken)

// Valor sentinela para o botao da Plataforma, que nao tem id de grupo.
const PLATAFORMA = '__plataforma__'

const grupos   = ref<GrupoInfo[]>([])
const loading  = ref(false)
const selecting = ref('')
const selectedId = ref('')
const error    = ref('')
const podePlataforma = ref(auth.podePlataforma)

const subtitulo = computed(() => {
  if (podePlataforma.value) return 'Escolha onde entrar: administrar a plataforma ou trabalhar em um grupo.'
  return isTroca.value ? 'Escolha o grupo para continuar' : 'Você pertence a múltiplos grupos. Escolha um para continuar.'
})

onMounted(async () => {
  if (isTroca.value) {
    // Autenticado trocando de contexto: o servidor diz quais sao os destinos.
    loading.value = true
    try {
      const ctxs = await auth.fetchContextos()
      grupos.value = ctxs.grupos
      podePlataforma.value = ctxs.pode_plataforma
    } catch {
      error.value = 'Erro ao carregar contextos.'
    } finally {
      loading.value = false
    }
  } else {
    // Fluxo de login: usa o que veio na resposta do /auth/login.
    grupos.value = auth.pendingGrupos
    podePlataforma.value = auth.podePlataforma
    if (grupos.value.length === 0 && !podePlataforma.value) {
      // Sem estado pendente — redireciona para login
      router.replace('/login')
    }
  }
})

async function entrarNaPlataforma() {
  await entrar(PLATAFORMA, () =>
    isTroca.value ? auth.trocaContexto('plataforma') : auth.selectContexto('plataforma'))
}

async function handleSelect(grupoID: string) {
  await entrar(grupoID, () =>
    isTroca.value ? auth.trocaGrupo(grupoID) : auth.selectGrupo(grupoID))
}

async function entrar(marca: string, acao: () => Promise<void>) {
  if (selecting.value) return
  selecting.value = marca
  selectedId.value = marca
  error.value = ''

  try {
    await acao()
    // destinoAposEntrar recusa redirect externo e cai na tela do contexto
    // quando nao ha redirect — '/' mandaria quem entrou na plataforma para um
    // 403, e ROTA_PLATAFORMA faria o mesmo com quem entrou num grupo.
    router.push(destinoAposEntrar(auth.contexto, route.query.redirect as string))
  } catch {
    error.value = 'Erro ao entrar. Tente novamente.'
    selectedId.value = ''
  } finally {
    selecting.value = ''
  }
}

function cancelar() {
  router.back()
}
</script>

<style scoped>
.select-grupo-page {
  min-height: 100vh;
  background: var(--bg);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.card {
  width: 100%;
  max-width: 440px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 40px 32px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logo-icon {
  width: 36px; height: 36px;
  display: grid; place-items: center;
}
.logo-icon .logo-marca { width: 36px; height: 36px; }
/* A logo entra como MASCARA e nao como imagem: o PNG tem a forma no canal
   alfa, e a cor vem do token, entao a marca acompanha os dois temas com um
   arquivo so. O lilas claro do arquivo sobre o fundo claro ficaria invisivel. */
.logo-marca {
  display: block;
  background: var(--primary);
  -webkit-mask: url('/logo-vision.png') center / contain no-repeat;
          mask: url('/logo-vision.png') center / contain no-repeat;
}
/* Sem suporte a mascara, volta ao quadrado com gradiente e a letra. */
@supports not ((mask-image: url('/logo-vision.png')) or (-webkit-mask-image: url('/logo-vision.png'))) {
  .logo-marca { display: none; }
  .logo-letra { display: grid; }
}
.logo-letra { display: none; place-items: center; }
@supports not ((mask-image: url('/logo-vision.png')) or (-webkit-mask-image: url('/logo-vision.png'))) {
  .logo-icon {
    border-radius: 10px;
    background: linear-gradient(135deg, var(--primary), var(--primary-line));
    font-size: var(--fs-lg); font-weight: 800; color: var(--text-oncolor);
  }
}

.logo-text {
  font-size: var(--fs-lg); font-weight: 800; color: var(--text);
}
.logo-text span { color: var(--primary); }

.title {
  font-size: var(--fs-lg); font-weight: 700; color: var(--text);
  margin: 0;
}

.subtitle {
  font-size: var(--fs-sm); color: var(--text-muted);
  margin: 0; line-height: 1.5;
}

.loading, .empty {
  color: var(--text-dim); font-size: var(--fs-sm); text-align: center; padding: 16px 0;
}

.grupos-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.grupo-btn {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  border-radius: 10px;
  border: 1px solid var(--border);
  background: var(--surface-2);
  cursor: pointer;
  color: var(--text);
  text-align: left;
  transition: var(--transition);
  position: relative;
}

.grupo-btn:hover:not(:disabled) {
  border-color: var(--primary);
  background: var(--primary-weak);
}

.grupo-btn.selected {
  border-color: var(--primary);
  background: var(--primary-weak);
}

.grupo-btn:disabled {
  opacity: 0.6;
  cursor: default;
}

/* A plataforma se destaca de proposito: e o contexto em que uma acao alcanca
   todos os clientes de uma vez. */
.plataforma-btn { border-color: var(--primary-line); }
.plataforma-icon { background: linear-gradient(135deg, var(--primary), var(--primary-line)); }

.grupo-icon {
  width: 36px; height: 36px; border-radius: 8px;
  background: linear-gradient(135deg, var(--warning), var(--primary-line));
  display: flex; align-items: center; justify-content: center;
  font-size: var(--fs-md); font-weight: 800; color: var(--text-oncolor);
  flex-shrink: 0;
}

.grupo-info { flex: 1; min-width: 0; }

.grupo-nome {
  font-size: var(--fs-base); font-weight: 600; color: var(--text);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

.grupo-slug {
  font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim); margin-top: 2px;
}

.spinner {
  width: 16px; height: 16px; border-radius: 50%;
  border: 2px solid var(--border-strong);
  border-top-color: var(--primary);
  animation: spin 0.7s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

.error {
  font-size: var(--fs-sm); color: var(--danger);
  background: var(--danger-weak);
  border: 1px solid var(--danger-weak);
  border-radius: 8px;
  padding: 10px 14px;
}

.cancel-btn {
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text-muted);
  padding: 10px;
  cursor: pointer;
  font-size: var(--fs-sm);
  transition: var(--transition);
}
.cancel-btn:hover { background: var(--surface-2); color: var(--text); }
</style>
