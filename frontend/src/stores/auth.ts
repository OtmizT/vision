import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api/client'
import { contextoOuGrupo, type Contexto } from '@/utils/navegacao'

export interface User {
  id:       string
  grupo_id: string
  nome:     string
  email:    string
  role:     'admin_global' | 'admin_grupo' | 'viewer'
  /*
   * Onde a pessoa está: administrando a plataforma ou dentro de um cliente.
   *
   * Vem do JWT, via /auth/me, e não do banco — o papel gravado em usuarios.role
   * diz o que a conta PODE ser, e o contexto diz o que ela é agora. Com um
   * admin global que também entra em grupos, os dois divergem o tempo todo.
   */
  contexto: Contexto
}

export interface GrupoInfo {
  id:          string
  nome:        string
  slug:        string
  schema_name: string
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken  = ref(localStorage.getItem('access_token') || '')
  const refreshToken = ref(localStorage.getItem('refresh_token') || '')
  const user         = ref<User | null>(null)

  // Estado de seleção pendente de grupo (multi-grupo no login)
  const preAuthToken  = ref(localStorage.getItem('pre_auth_token') || '')
  const pendingGrupos = ref<GrupoInfo[]>(JSON.parse(localStorage.getItem('pending_grupos') || '[]'))
  const meusGrupos    = ref<GrupoInfo[]>(JSON.parse(localStorage.getItem('meus_grupos') || '[]'))

  // Quem pode escolher Plataforma na tela de seleção. Vem do servidor (o login
  // e /auth/contextos respondem), e não de um `role === 'admin_global'` aqui:
  // a lista de destinos é decidida num lugar só.
  const podePlataforma = ref(localStorage.getItem('pode_plataforma') === '1')

  const isAuthenticated  = computed(() => !!accessToken.value)
  const needsGroupSelect = computed(() => !!preAuthToken.value && !accessToken.value)
  /** Contexto ativo. Ausente é lido como 'grupo' — omissão não promove. */
  const contexto = computed<Contexto>(() => contextoOuGrupo(user.value?.contexto))
  const noContextoPlataforma = computed(() => contexto.value === 'plataforma')
  /** Nome do grupo ativo, para o badge. */
  const nomeGrupoAtivo = computed(() =>
    meusGrupos.value.find(g => g.id === user.value?.grupo_id)?.nome ?? '')
  const isAdminGlobal    = computed(() => user.value?.role === 'admin_global')
  const isAdminGrupo     = computed(() => user.value?.role === 'admin_grupo')
  const isViewer         = computed(() => user.value?.role === 'viewer')
  const isAdmin          = computed(() => ['admin_global', 'admin_grupo'].includes(user.value?.role ?? ''))

  function setTokens(access: string, refresh: string) {
    accessToken.value  = access
    refreshToken.value = refresh
    localStorage.setItem('access_token',  access)
    localStorage.setItem('refresh_token', refresh)
    preAuthToken.value  = ''
    pendingGrupos.value = []
    localStorage.removeItem('pre_auth_token')
    localStorage.removeItem('pending_grupos')
  }

  function clearTokens() {
    accessToken.value   = ''
    refreshToken.value  = ''
    preAuthToken.value  = ''
    pendingGrupos.value = []
    meusGrupos.value    = []
    user.value          = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem('pre_auth_token')
    localStorage.removeItem('pending_grupos')
    localStorage.removeItem('meus_grupos')
    podePlataforma.value = false
    localStorage.removeItem('pode_plataforma')
  }

  async function login(email: string, password: string) {
    const { data } = await api.post('/auth/login', { email, password })
    const resp = data.data

    if (resp.needs_select) {
      // Seleção pendente. O admin global sempre cai aqui, mesmo com um grupo
      // só: sem Plataforma na lista ele entraria direto no grupo e o painel
      // dele ficaria sem caminho de volta que não fosse deslogar.
      preAuthToken.value   = resp.pre_auth_token
      pendingGrupos.value  = resp.grupos ?? []
      podePlataforma.value = !!resp.pode_plataforma
      localStorage.setItem('pre_auth_token',  resp.pre_auth_token)
      localStorage.setItem('pending_grupos', JSON.stringify(resp.grupos ?? []))
      localStorage.setItem('pode_plataforma', resp.pode_plataforma ? '1' : '0')
      return
    }

    setTokens(resp.access_token, resp.refresh_token)
    await fetchMe()
    await refreshMeusGrupos()
  }

  // grupoID vazio só faz sentido no contexto de plataforma, que não tem grupo.
  async function selectContexto(ctx: Contexto, grupoID = '') {
    const { data } = await api.post('/auth/select-grupo', {
      pre_auth_token: preAuthToken.value,
      contexto: ctx,
      grupo_id: grupoID
    })
    setTokens(data.data.access_token, data.data.refresh_token)
    await fetchMe()
    await refreshMeusGrupos()
  }

  async function trocaContexto(ctx: Contexto, grupoID = '') {
    const { data } = await api.post('/auth/troca-grupo', { contexto: ctx, grupo_id: grupoID })
    setTokens(data.data.access_token, data.data.refresh_token)
    await fetchMe()
    await refreshMeusGrupos()
  }

  // Mantidos para os chamadores que só trocam de grupo.
  const selectGrupo = (grupoID: string) => selectContexto('grupo', grupoID)
  const trocaGrupo  = (grupoID: string) => trocaContexto('grupo', grupoID)

  /** Destinos possíveis para a tela de troca de contexto. */
  async function fetchContextos(): Promise<{ pode_plataforma: boolean; grupos: GrupoInfo[] }> {
    const { data } = await api.get('/auth/contextos')
    const resp = data.data ?? {}
    podePlataforma.value = !!resp.pode_plataforma
    localStorage.setItem('pode_plataforma', resp.pode_plataforma ? '1' : '0')
    return { pode_plataforma: !!resp.pode_plataforma, grupos: resp.grupos ?? [] }
  }

  async function fetchGrupos(): Promise<GrupoInfo[]> {
    const { data } = await api.get('/auth/grupos')
    return data.data ?? []
  }

  async function refreshMeusGrupos() {
    try {
      const grupos = await fetchGrupos()
      meusGrupos.value = grupos
      localStorage.setItem('meus_grupos', JSON.stringify(grupos))
    } catch {
      // silencioso — não crítico
    }
  }

  async function refresh() {
    const { data } = await api.post('/auth/refresh', {
      refresh_token: refreshToken.value
    })
    setTokens(data.data.access_token, data.data.refresh_token)
  }

  // Nunca rejeita: revogar o token no servidor é best-effort. O estado local é
  // sempre limpo, para que o redirect do chamador aconteça em qualquer cenário.
  async function logout() {
    try {
      await api.post('/auth/logout', { refresh_token: refreshToken.value })
    } catch {
      // 401 (token já expirado), 422 ou falha de rede não devem impedir a saída
    } finally {
      clearTokens()
    }
  }

  async function fetchMe() {
    const { data } = await api.get('/auth/me')
    user.value = data.data
  }

  // Restaura o estado do usuário a partir do token salvo.
  // Idempotente e deduplicado: App.vue (startup) e o guard do router chamam os
  // dois, e sem o cache da promise em voo isso dispararia dois GET /auth/me
  // concorrentes — que, com token expirado, viram dois refresh concorrentes
  // disputando um refresh token de uso único (rotação obrigatória).
  let bootstrapPromise: Promise<void> | null = null

  async function ensureLoaded() {
    if (!accessToken.value || user.value) return
    if (bootstrapPromise) return bootstrapPromise

    bootstrapPromise = (async () => {
      try {
        await fetchMe()
        await refreshMeusGrupos()
      } catch {
        clearTokens()
      } finally {
        bootstrapPromise = null
      }
    })()

    return bootstrapPromise
  }

  // Mantido como alias — App.vue chama init() no startup.
  async function init() {
    return ensureLoaded()
  }

  return {
    accessToken, refreshToken, user, preAuthToken, pendingGrupos, meusGrupos, podePlataforma,
    isAuthenticated, needsGroupSelect, isAdminGlobal, isAdminGrupo, isViewer, isAdmin,
    contexto, noContextoPlataforma, nomeGrupoAtivo,
    login, selectGrupo, trocaGrupo, selectContexto, trocaContexto,
    fetchGrupos, fetchContextos, refreshMeusGrupos,
    logout, refresh, fetchMe, init, ensureLoaded, clearTokens, setTokens
  }
})
