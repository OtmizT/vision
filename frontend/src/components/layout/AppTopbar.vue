<template>
  <header class="topbar">
    <!-- Hamburger -->
    <button class="hamburger" @click="$emit('toggle-sidebar')" aria-label="Menu">
      <span /><span /><span />
    </button>

    <!-- Título -->
    <div class="topbar-title">
      {{ pageTitles[route.name as string] ?? 'VisiON' }}
      <span v-if="pageSubtitle">{{ pageSubtitle }}</span>
    </div>

    <!-- Grupo ativo. Sem isto, uma troca indevida de grupo passa despercebida —
         nada mais na tela indica de qual cliente são os dados exibidos. -->
    <div v-if="grupoAtivo" class="grupo-chip" :title="`Grupo ativo: ${grupoAtivo}`">
      <span class="grupo-dot" />
      <span class="grupo-nome">{{ grupoAtivo }}</span>
    </div>

    <!-- Direita -->
    <div class="topbar-right">
      <!-- Frescor dos dados. Substitui o "AO VIVO", que dava a impressão de
           tempo real: os dados vêm de views materializadas atualizadas a cada
           sync, então o que importa é QUANDO foi a última. Discreto de propósito
           — é informação de apoio, não estado de sistema. -->
      <span v-if="ultimaAtualizacao" class="sync-info" :title="tituloSync">
        {{ ultimaAtualizacao }}
      </span>

      <!-- Filtros: rotulado, não só ícone. É o controle mais usado do dashboard
           e um ícone solitário não comunica que há uma barra recolhida ali. -->
      <button
        v-if="temFiltros"
        class="filtro-btn"
        :class="{ 'filtro-btn--on': ui.filtrosAbertos }"
        :aria-expanded="ui.filtrosAbertos"
        :title="ui.filtrosAbertos ? 'Ocultar filtros' : 'Mostrar filtros'"
        @click="ui.toggleFiltros()"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
             stroke-linecap="round"><path d="M3 5h18M7 12h10M10 19h4" /></svg>
        <span class="filtro-txt">Filtros</span>
        <span class="filtro-chv">▾</span>
      </button>

      <!-- Theme toggle -->
      <button class="theme-btn" @click="ui.toggleTheme()" :title="ui.theme === 'dark' ? 'Modo claro' : 'Modo escuro'">
        <Sun  v-if="ui.theme === 'dark'" :size="16" />
        <Moon v-else                     :size="16" />
      </button>

      <!-- Logout -->
      <button class="logout-btn" @click="handleLogout" title="Sair do sistema">
        <LogOut :size="16" />
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { syncApi } from '@/api/sync'
import { Sun, Moon, LogOut } from '@lucide/vue'

defineEmits<{ 'toggle-sidebar': [] }>()

const route = useRoute()
const router = useRouter()
const ui    = useUiStore()
const auth  = useAuthStore()

// ── Frescor dos dados ──────────────────────────────────────────────────────
const syncEm = ref<Date | null>(null)

/**
 * Busca silenciosa: se falhar, o indicador simplesmente não aparece. É
 * informação de apoio no cabeçalho — não vale interromper a navegação com erro,
 * nem poluir o console em cada rota.
 */
async function carregarUltimaAtualizacao() {
  if (!auth.user?.grupo_id) { syncEm.value = null; return }
  try {
    const { data } = await syncApi.ultimaAtualizacao()
    const bruto = data?.data?.ultimo_sync_at ?? null
    syncEm.value = bruto ? new Date(bruto) : null
  } catch {
    syncEm.value = null
  }
}

// Recarrega ao trocar de grupo: o horário é por grupo, e manter o anterior
// mostraria o frescor do cliente errado.
watch(() => auth.user?.grupo_id, () => { carregarUltimaAtualizacao() })
onMounted(carregarUltimaAtualizacao)

/**
 * "hoje HH:MM" para o dia corrente, "ontem HH:MM" para o anterior e
 * "DD/MM HH:MM" além disso. Data cheia num rótulo discreto rouba atenção sem
 * acrescentar nada no caso comum, que é o sync do próprio dia.
 */
const ultimaAtualizacao = computed(() => {
  const d = syncEm.value
  if (!d || isNaN(d.getTime())) return null

  const hora = d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })
  const soData = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime()
  const dias = Math.round((soData(new Date()) - soData(d)) / 86400000)

  if (dias === 0) return `Sync hoje ${hora}`
  if (dias === 1) return `Sync ontem ${hora}`
  return `Sync ${d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })} ${hora}`
})

// O título traz a data completa, para quem precisar do valor exato.
const tituloSync = computed(() =>
  syncEm.value ? `Última sincronização: ${syncEm.value.toLocaleString('pt-BR')}` : ''
)

async function handleLogout() {
  if (confirm('Deseja realmente sair do sistema?')) {
    await auth.logout()   // não rejeita; limpa o estado local em qualquer cenário
    router.replace('/login')
  }
}

// O botão de filtros só faz sentido onde existe barra de filtros. Deriva da
// rota em vez de sondar o DOM do alvo do Teleport, que não é reativo.
const temFiltros = computed(() => route.name === 'Dashboard')

const pageTitles: Record<string, string> = {
  Dashboard:    'Dashboard',
  Grupos:       'Grupos',
  GrupoEmpresas:'Empresas',
  MinhasEmpresas:'Empresas',
  Usuarios:     'Usuários',
  Permissoes:   'Permissões',
  Sync:         'Sincronização',
  Perfil:       'Perfil',
  Forbidden:    'Acesso Negado',
}

const pageSubtitle = computed(() => {
  if (route.name === 'Dashboard') return 'Visão Geral'
  if (route.name === 'Sync') return 'Motor ETL'
  return null
})

// Nome do grupo cujo contexto está ativo no token. Resolvido a partir do
// grupo_id das claims cruzado com a lista de grupos do usuário.
const grupoAtivo = computed(() => {
  const gid = auth.user?.grupo_id
  if (!gid) return null
  return auth.meusGrupos.find(g => g.id === gid)?.nome ?? null
})
</script>

<style scoped>
/* Sem sticky nem fundo próprio: quem fixa e pinta é o .appbar do MainLayout,
   que envolve topbar + filtros + abas num único bloco. Manter sticky aqui
   criaria um segundo contexto de fixação dentro do primeiro. */
.topbar {
  padding: 0 var(--sp-6);
  height: var(--topbar-h);
  display: flex; align-items: center; gap: 14px;
  overflow: visible;
}
@media (max-width: 1023px) { .topbar { padding: 0 var(--sp-5); } }
@media (max-width: 767px)  { .topbar { padding: 0 var(--sp-4); } }

.hamburger {
  width: 38px; height: 38px; border-radius: 10px;
  border: 1px solid var(--border-strong); background: var(--surface-2);
  cursor: pointer; display: none; align-items: center;
  justify-content: center; flex-direction: column; gap: 4px; flex-shrink: 0;
  transition: var(--transition);
}
.hamburger:hover { border-color: var(--primary); }
.hamburger span { display: block; width: 16px; height: 1.5px; background: var(--text-muted); border-radius: 2px; }

@media (max-width: 768px) { .hamburger { display: flex; } }

.topbar-title {
  font-size: var(--fs-md); font-weight: 700; flex-shrink: 0;
  color: var(--text);
}
.topbar-title span { color: var(--primary); margin-left: 6px; }

.grupo-chip {
  display: flex; align-items: center; gap: 6px;
  flex-shrink: 0;
  padding: 4px 10px;
  border: 1px solid var(--border-strong);
  border-radius: 20px;
  background: var(--surface-2);
  max-width: 200px;
}
.grupo-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: var(--primary); flex-shrink: 0;
}
.grupo-nome {
  font-family: var(--font-mono); font-size: var(--fs-xs);
  letter-spacing: 0.5px; color: var(--text-muted);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

@media (max-width: 900px) { .grupo-chip { max-width: 120px; } }

/* O alvo dos filtros saiu da topbar e virou uma faixa própria no MainLayout:
   dentro da topbar eles disputavam largura com o título e o chip de grupo. */
.topbar-right { display: flex; align-items: center; gap: 10px; margin-left: auto; }

/* ── Botão de filtros ── */
.filtro-btn {
  display: inline-flex; align-items: center; gap: 7px;
  height: 34px; padding: 0 11px;
  border: 1px solid var(--primary); border-radius: var(--r-sm);
  background: var(--primary-weak); color: var(--primary);
  font-size: var(--fs-sm); font-weight: 600; white-space: nowrap;
  cursor: pointer; transition: var(--transition);
}
.filtro-btn:hover { background: var(--primary); color: var(--text-oncolor); }
.filtro-btn--on   { background: var(--primary); color: var(--text-oncolor); }
.filtro-btn svg { width: 16px; height: 16px; }
.filtro-chv { font-size: var(--fs-xs); line-height: 1; transition: transform var(--transition); }
.filtro-btn--on .filtro-chv { transform: rotate(180deg); }
@media (max-width: 900px) { .filtro-txt { display: none; } }

.sync-info {
  font-family: var(--font-mono); font-size: var(--fs-xs);
  color: var(--text-dim); white-space: nowrap;
}
@media (max-width: 640px) { .sync-info { display: none; } }

.theme-btn {
  width: 38px; height: 38px; border-radius: 10px;
  border: 1px solid var(--border-strong); background: var(--surface-2);
  cursor: pointer; display: flex; align-items: center; justify-content: center;
  color: var(--text-muted); transition: var(--transition); flex-shrink: 0;
}
.theme-btn:hover { border-color: var(--primary); color: var(--primary); }
.theme-btn svg { width: 16px; height: 16px; }

.logout-btn {
  width: 38px; height: 38px; border-radius: 10px;
  border: 1px solid var(--border-strong); background: var(--surface-2);
  cursor: pointer; display: flex; align-items: center; justify-content: center;
  color: var(--text-dim); transition: var(--transition); flex-shrink: 0;
}
.logout-btn:hover { border-color: var(--danger); color: var(--danger); background: var(--danger-weak); }
.logout-btn svg { width: 16px; height: 16px; }
</style>
