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

      <h2 class="title">Selecionar Grupo</h2>
      <p class="subtitle">{{ isTroca ? 'Escolha o grupo para continuar' : 'Você pertence a múltiplos grupos. Escolha um para continuar.' }}</p>

      <div v-if="loading" class="loading">Carregando grupos...</div>

      <div v-else-if="grupos.length === 0" class="empty">
        Nenhum grupo disponível.
      </div>

      <div v-else class="grupos-list">
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

const grupos   = ref<GrupoInfo[]>([])
const loading  = ref(false)
const selecting = ref('')
const selectedId = ref('')
const error    = ref('')

onMounted(async () => {
  if (isTroca.value) {
    // Busca grupos do usuário autenticado
    loading.value = true
    try {
      grupos.value = await auth.fetchGrupos()
    } catch {
      error.value = 'Erro ao carregar grupos.'
    } finally {
      loading.value = false
    }
  } else {
    // Usa grupos salvos do flow de login
    grupos.value = auth.pendingGrupos
    if (grupos.value.length === 0) {
      // Sem estado pendente — redireciona para login
      router.replace('/login')
    }
  }
})

async function handleSelect(grupoID: string) {
  if (selecting.value) return
  selecting.value = grupoID
  selectedId.value = grupoID
  error.value = ''

  try {
    if (isTroca.value) {
      await auth.trocaGrupo(grupoID)
    } else {
      await auth.selectGrupo(grupoID)
    }
    // destinoAposEntrar recusa redirect externo e cai na tela do papel quando
    // nao ha redirect — '/' mandaria o admin global para um 403.
    router.push(destinoAposEntrar(auth.user?.role, route.query.redirect as string))
  } catch (e: unknown) {
    error.value = 'Erro ao selecionar grupo. Tente novamente.'
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
  font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-dim); margin-top: 2px;
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
