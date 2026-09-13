<template>
  <div v-if="mobileOpen" class="sidebar-overlay" @click="$emit('close')" />

  <aside
    :class="['sidebar', { 'sidebar--expanded': isExpanded, 'sidebar--mobile-open': mobileOpen }]"
    @mouseenter="hovering = true"
    @mouseleave="hovering = false"
  >
    <!--
      O logo é só a marca: não clica, não recebe foco, não fixa nada.

      Ele já foi um botão que alternava ui.sidebarPinned, e isso era um defeito:
      um clique num quadrado de 34px gravava o estado no localStorage e a barra
      ficava aberta para sempre — inclusive após recarregar, sem pista de como
      desfazer. Clicar num logo é gesto de "voltar ao início", não de fixar menu.
      Havia ainda um caminho por teclado: com tabindex, espaço também fixava.
    -->
    <div class="sidebar-top">
      <div class="logo-icon">
        <span class="logo-marca" aria-hidden="true" />
        <span class="logo-letra" aria-hidden="true">V</span>
      </div>
      <div class="logo-text">
        <div class="logo-name">Visi<span>ON</span></div>
        <div class="logo-sub">ADMIN PANEL</div>
      </div>
    </div>

    <nav class="nav-scroll">
      <template v-for="section in navSections" :key="section.label">
        <div class="nav-label">{{ section.label }}</div>
        <RouterLink
          v-for="item in section.items"
          :key="item.to"
          :to="item.to"
          custom
          v-slot="{ isActive, navigate }"
        >
          <div
            :class="['nav-item', { active: isActive }]"
            @click="navigate(); $emit('close')"
            role="link"
          >
            <div class="nav-icon">
              <component :is="item.icon" />
            </div>
            <span class="nav-text">{{ item.label }}</span>
          </div>
        </RouterLink>
      </template>
    </nav>

    <div class="sidebar-footer">
      <div class="user-card">
        <div class="user-avatar">{{ initials }}</div>
        <div class="user-info">
          <div class="user-name">{{ auth.user?.nome ?? '...' }}</div>
          <div class="user-role">{{ roleLabel }}</div>
          <!-- Sem este badge, o admin global dentro de um grupo se le apenas
               como "Admin Grupo", sem nenhuma pista de onde esta. -->
          <div class="user-contexto" :class="{ plataforma: auth.noContextoPlataforma }">
            {{ contextoLabel }}
          </div>
        </div>
        <button class="logout-mini" @click="handleLogout" title="Sair">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15M12 9l-3 3m0 0l3 3m-3-3h12.75" />
          </svg>
        </button>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import {
  IconGrid, IconBuilding, IconFactory,
  IconUsers, IconKey, IconSync, IconUser, IconDatabase
} from '@/components/ui/icons'
import { temDashboard, rotuloContexto } from '@/utils/navegacao'

const IconSwitch = {
  template: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 16V4m0 0L3 8m4-4l4 4M17 8v12m0 0l4-4m-4 4l-4-4"/></svg>`
}

defineProps<{ mobileOpen: boolean }>()
defineEmits<{ close: [] }>()

const auth     = useAuthStore()
const router   = useRouter()
const hovering = ref(false)

async function handleLogout() {
  if (confirm('Deseja realmente sair do sistema?')) {
    await auth.logout()   // não rejeita; limpa o estado local em qualquer cenário
    router.replace('/login')
  }
}

const isExpanded = computed(() => hovering.value)

const initials = computed(() => {
  const name = auth.user?.nome ?? ''
  return name.split(' ').map((w: string) => w[0]).slice(0, 2).join('').toUpperCase() || 'U'
})

const roleLabel = computed(() => {
  const labels: Record<string, string> = {
    admin_global: 'Admin Global',
    admin_grupo:  'Admin Grupo',
    viewer:       'Viewer'
  }
  return labels[auth.user?.role ?? 'viewer'] ?? ''
})

const contextoLabel = computed(() => rotuloContexto(auth.contexto, auth.nomeGrupoAtivo))

const navSections = computed(() => {
  const role = auth.user?.role
  const sections: Array<{ label: string; items: Array<{ to: string; label: string; icon: unknown }> }> = []

  // O Dashboard existe dentro de um grupo, nao na plataforma: la se administra
  // o produto, nao se consome dado financeiro de cliente nenhum. A decisao e
  // por CONTEXTO, e nao por papel — o mesmo admin global tem Dashboard quando
  // entra num grupo. Ver utils/navegacao.ts, a mesma funcao que decide a rota
  // inicial: se divergirem, o item some do menu e a URL continua aberta.
  if (temDashboard(auth.contexto)) {
    sections.push({
      label: 'PRINCIPAL',
      items: [{ to: '/', label: 'Dashboard', icon: IconGrid }]
    })
  }

  const adminItems: Array<{ to: string; label: string; icon: unknown }> = []
  if (role === 'admin_global') {
    adminItems.push({ to: '/grupos', label: 'Grupos', icon: IconBuilding })
    adminItems.push({ to: '/admin/sync-control', label: 'Sync Control', icon: IconSync })
  }
  if (role === 'admin_grupo') {
    adminItems.push({ to: '/empresas',   label: 'Empresas',   icon: IconFactory })
    adminItems.push({ to: '/usuarios',   label: 'Usuarios',   icon: IconUsers })
    adminItems.push({ to: '/permissoes', label: 'Permissoes', icon: IconKey })
  }
  if (adminItems.length) {
    sections.push({ label: 'ADMINISTRACAO', items: adminItems })
  }

  const systemItems: Array<{ to: string; label: string; icon: unknown }> = []
  if (role === 'admin_grupo') {
    systemItems.push({ to: '/sync', label: 'Sync', icon: IconSync })
  }
  if (role === 'admin_global') {
    systemItems.push({ to: '/omie-config', label: 'Config Omie', icon: IconKey })
  }
  if (role === 'admin_global' || role === 'admin_grupo') {
    systemItems.push({ to: '/sql-explorer', label: 'SQL Explorer', icon: IconDatabase })
  }
  // "Trocar Contexto" e nao "Trocar Grupo": para o admin global, Plataforma e
  // um destino como qualquer outro, e ele pode ter um grupo so.
  if (auth.meusGrupos.length > 1 || auth.podePlataforma) {
    systemItems.push({ to: '/select-grupo', label: 'Trocar Contexto', icon: IconSwitch })
  }
  systemItems.push({ to: '/perfil', label: 'Perfil', icon: IconUser })
  sections.push({ label: 'SISTEMA', items: systemItems })

  return sections
})
</script>

<style scoped>
.sidebar-overlay {
  position: fixed; inset: 0;
  background: var(--overlay);
  z-index: 200;
  backdrop-filter: blur(2px);
}

.sidebar {
  position: fixed; left: 0; top: 0; bottom: 0;
  width: var(--sidebar-w);
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border);
  backdrop-filter: blur(24px);
  z-index: 300;
  display: flex; flex-direction: column;
  transition: width 0.28s cubic-bezier(0.4,0,0.2,1), transform 0.28s cubic-bezier(0.4,0,0.2,1), box-shadow 0.28s;
  overflow: hidden;
}

.sidebar--expanded { width: var(--sidebar-w-expanded); box-shadow: 4px 0 40px rgba(0,0,0,0.25); }

@media (max-width: 768px) {
  .sidebar { transform: translateX(-100%); width: var(--sidebar-w-expanded) !important; }
  .sidebar--mobile-open { transform: translateX(0); box-shadow: 4px 0 40px rgba(0,0,0,0.35); }
}

.sidebar-top {
  padding: 16px 0; border-bottom: 1px solid var(--border);
  display: flex; align-items: center;
  min-height: 64px; flex-shrink: 0;
  padding-left: 15px;
}

.logo-icon {
  width: 34px; height: 34px; flex-shrink: 0;
  display: grid; place-items: center;
}
.logo-icon .logo-marca { width: 34px; height: 34px; }
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
    background: linear-gradient(135deg, var(--primary-line), var(--accent-line));
    font-family: var(--font-display); font-size: var(--fs-md); font-weight: 700;
    color: #fff; box-shadow: 0 2px 10px var(--primary-weak);
  }
}

.logo-text {
  opacity: 0; transform: translateX(-8px);
  transition: opacity 0.2s, transform 0.2s;
  white-space: nowrap; margin-left: 12px; overflow: hidden;
}
.sidebar--expanded .logo-text,
.sidebar--mobile-open .logo-text { opacity: 1; transform: translateX(0); }

.logo-name { font-size: var(--fs-md); font-weight: 800; color: var(--text); }
.logo-name span { color: var(--primary); }
.logo-sub { font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-dim); letter-spacing: 2px; margin-top: 1px; }

.nav-scroll { flex: 1; overflow-y: auto; overflow-x: hidden; padding: 10px 8px; }
.nav-scroll::-webkit-scrollbar { width: 3px; }
.nav-scroll::-webkit-scrollbar-thumb { background: var(--border-strong); border-radius: 3px; }

.nav-label {
  font-family: var(--font-display); font-size: var(--fs-xs); letter-spacing: 2px; color: var(--text-dim);
  padding: 8px 8px 4px; white-space: nowrap; overflow: hidden;
  opacity: 0; max-height: 0; transition: opacity 0.2s, max-height 0.2s;
}
.sidebar--expanded .nav-label,
.sidebar--mobile-open .nav-label { opacity: 1; max-height: 30px; }

.nav-item {
  position: relative;
  display: flex; align-items: center; padding: 0 8px; height: 42px;
  border-radius: var(--r-sm); cursor: pointer; color: var(--text-muted);
  transition: var(--transition); margin-bottom: 1px;
  white-space: nowrap; overflow: hidden;
  text-decoration: none;
}
.nav-item:hover { background: var(--surface-2); color: var(--text); }
/* Ativo por barra lateral + fundo fraco, como no padrão visual. A borda de 1px
   que havia antes some: com a sidebar colapsada em 64px ela virava um retângulo
   fechado em volta do ícone, pesado demais para um estado de navegação. */
.nav-item.active { background: var(--sidebar-active); color: var(--primary); font-weight: 600; }
.nav-item.active::before {
  content: ''; position: absolute; left: 0; top: 50%; transform: translateY(-50%);
  width: 3px; height: 18px; border-radius: 0 3px 3px 0; background: var(--primary);
}

.nav-icon { width: 32px; height: 32px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; border-radius: 8px; }
.nav-icon :deep(svg) { width: 20px; height: 20px; }

.nav-text { font-size: var(--fs-sm); font-weight: 600; margin-left: 6px; opacity: 0; width: 0; transition: opacity 0.18s, width 0.18s; }
.sidebar--expanded .nav-text,
.sidebar--mobile-open .nav-text { opacity: 1; width: auto; }

.sidebar-footer { padding: 16px 10px; border-top: 1px solid var(--border); flex-shrink: 0; }

.user-card { display: flex; align-items: center; gap: 10px; padding: 8px; border-radius: 10px; background: var(--surface-2); overflow: hidden; }

.user-avatar {
  width: 30px; height: 30px; flex-shrink: 0; border-radius: 8px;
  background: linear-gradient(135deg, var(--primary), var(--primary-line));
  display: flex; align-items: center; justify-content: center;
  font-size: var(--fs-xs); font-weight: 800; color: var(--text-oncolor);
}

.user-info { opacity: 0; width: 0; overflow: hidden; transition: opacity 0.18s, width 0.18s; }
.sidebar--expanded .user-info,
.sidebar--mobile-open .user-info { opacity: 1; width: auto; }

.user-name { font-size: var(--fs-xs); font-weight: 700; white-space: nowrap; }
.user-role { font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-dim); }

.user-contexto {
  margin-top: 3px;
  font-size: var(--fs-xs);
  font-weight: 600;
  color: var(--text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
/* A plataforma se destaca de proposito: e o contexto em que uma acao alcanca
   todos os clientes de uma vez. */
.user-contexto.plataforma {
  color: var(--primary);
}

.logout-mini {
  background: transparent; border: none; cursor: pointer;
  color: var(--text-dim); padding: 4px; border-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  transition: var(--transition); opacity: 0; width: 0; overflow: hidden;
}
.sidebar--expanded .logout-mini,
.sidebar--mobile-open .logout-mini { opacity: 1; width: 28px; margin-left: auto; }
.logout-mini:hover { color: var(--danger); background: var(--danger-weak); }
.logout-mini svg { width: 14px; height: 14px; }
</style>
