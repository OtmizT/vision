import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { rotaInicial } from '@/utils/navegacao'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    // ── Público ──────────────────────────────────────────────
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true }
    },
    {
      path: '/select-grupo',
      name: 'SelectGrupo',
      component: () => import('@/views/SelectGrupoView.vue'),
      meta: { selectGrupo: true }
    },

    // ── Autenticado ──────────────────────────────────────────
    {
      path: '/',
      component: () => import('@/components/layout/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        // Dashboard: a tela principal do produto. Fica de fora do admin
        // global, que administra a plataforma e nao le dados de grupo.
        // Sem esta restricao o item sumia do menu e a URL seguia aberta.
        {
          path: '',
          name: 'Dashboard',
          component: () => import('@/views/DashboardView.vue'),
          meta: { roles: ['admin_grupo', 'viewer'] }
        },

        // Grupos — admin_global apenas
        {
          path: 'grupos',
          name: 'Grupos',
          component: () => import('@/views/grupos/GruposView.vue'),
          meta: { roles: ['admin_global'] }
        },
        {
          path: 'grupos/:grupoId/empresas',
          name: 'GrupoEmpresas',
          component: () => import('@/views/empresas/EmpresasView.vue'),
          meta: { roles: ['admin_global'] }
        },

        // Empresas — admin_grupo apenas (admin_global acessa via /grupos/:id/empresas)
        {
          path: 'empresas',
          name: 'MinhasEmpresas',
          component: () => import('@/views/empresas/EmpresasView.vue'),
          meta: { roles: ['admin_grupo'] }
        },

        // Usuários
        {
          path: 'usuarios',
          name: 'Usuarios',
          component: () => import('@/views/usuarios/UsuariosView.vue'),
          meta: { roles: ['admin_grupo'] }
        },

        // Permissões
        {
          path: 'permissoes',
          name: 'Permissoes',
          component: () => import('@/views/permissoes/PermissoesView.vue'),
          meta: { roles: ['admin_grupo'] }
        },

        // Sync
        {
          path: 'sync',
          name: 'Sync',
          component: () => import('@/views/sync/SyncView.vue'),
          meta: { roles: ['admin_grupo'] }
        },
        {
          path: 'sync/:empresaId',
          name: 'SyncEmpresa',
          component: () => import('@/views/sync/SyncEmpresaView.vue'),
          meta: { roles: ['admin_grupo'] }
        },

        // Omie Config — admin_global apenas
        {
          path: 'omie-config',
          name: 'OmieConfig',
          component: () => import('@/views/admin/OmieConfigView.vue'),
          meta: { roles: ['admin_global'] }
        },
        {
          path: 'admin/sync-control',
          name: 'SyncControlCenter',
          component: () => import('@/views/admin/SyncControlCenter.vue'),
          meta: { roles: ['admin_global'] }
        },

        // SQL Explorer — admin_global e admin_grupo
        {
          path: 'sql-explorer',
          name: 'SqlExplorer',
          component: () => import('@/views/sql/SqlExplorerView.vue'),
          meta: { roles: ['admin_global', 'admin_grupo'] }
        },

        // Perfil — todos
        {
          path: 'perfil',
          name: 'Perfil',
          component: () => import('@/views/PerfilView.vue')
        },

        // 403
        {
          path: '403',
          name: 'Forbidden',
          component: () => import('@/views/ForbiddenView.vue')
        }
      ]
    },

    // Catch-all
    { path: '/:pathMatch(.*)*', redirect: '/' }
  ]
})

// ── Guards ──────────────────────────────────────────────────
router.beforeEach(async to => {
  const auth = useAuthStore()

  // Rota de seleção de grupo
  if (to.meta.selectGrupo) {
    // Permite acesso se tem preAuthToken (fluxo login multi-grupo) OU se já autenticado (fluxo troca-grupo)
    if (!auth.needsGroupSelect && !auth.isAuthenticated) return { name: 'Login' }
    return true
  }

  // Rota pública
  if (to.meta.public) {
    if (auth.needsGroupSelect) return { name: 'SelectGrupo' }
    // Quem ja esta autenticado vai para a tela do proprio papel — nem todo
    // mundo tem Dashboard. Ver utils/navegacao.ts.
    if (auth.isAuthenticated)  return rotaInicial(auth.user?.role)
    return true
  }

  // Seleção de grupo pendente — bloqueia acesso a rotas protegidas até selecionar
  if (auth.needsGroupSelect) return { name: 'SelectGrupo' }

  // Requer autenticação
  if (to.meta.requiresAuth !== false) {
    if (!auth.isAuthenticated) return { name: 'Login' }

    // Carrega dados do usuário se ainda não carregou.
    // ensureLoaded() é deduplicado — compartilha a chamada em voo com App.vue.
    if (!auth.user) {
      await auth.ensureLoaded()
      if (!auth.user) return { name: 'Login' }
    }

    // Verifica role
    const requiredRoles = to.meta.roles as string[] | undefined
    if (requiredRoles && !requiredRoles.includes(auth.user!.role)) {
      return { name: 'Forbidden' }
    }
  }

  return true
})

export default router
