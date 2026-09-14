import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
import { proximoFoco, type IdFoco } from '@/utils/foco'

export type Tema = 'dark' | 'light'

function temaSalvo(): Tema {
  return localStorage.getItem('theme') === 'light' ? 'light' : 'dark'
}

export const useUiStore = defineStore('ui', () => {
  const theme         = ref<Tema>(temaSalvo())
  /*
   * Limpeza única de 'sidebar_pinned'. A chave existiu enquanto o clique no
   * logo fixava a sidebar aberta — o que virou defeito: quem clicasse uma vez
   * ficava com a barra aberta para sempre, inclusive após recarregar. O
   * gatilho saiu, mas quem já clicou tem a chave gravada; sem remover, esses
   * navegadores carregariam estado morto indefinidamente.
   */
  localStorage.removeItem('sidebar_pinned')

  /**
   * Barra de filtros recolhida por padrão: fechada o cabeçalho fixo devolve
   * ~55px de altura ao conteúdo. A preferência persiste para quem trabalha
   * com os filtros sempre à vista.
   */
  const filtrosAbertos = ref(localStorage.getItem('filtros_abertos') === 'true')

  /**
   * Exibição de centavos. Persistida porque é preferência de leitura, não estado
   * de navegação: há empresas em que o centavo é irrelevante, e quem desliga não
   * deve precisar desligar de novo a cada acesso.
   *
   * Atua só onde existem centavos hoje — o pivô do Resultado e os cards de KPI.
   * Ver a nota em utils/formato.ts.
   */
  const mostrarCentavos = ref(localStorage.getItem('mostrar_centavos') !== 'false')

  function toggleCentavos() {
    mostrarCentavos.value = !mostrarCentavos.value
    localStorage.setItem('mostrar_centavos', String(mostrarCentavos.value))
  }

  /**
   * Modo foco: QUAL bloco está expandido, ocupando a página sozinho. Some a
   * faixa de abas e os blocos irmãos; barra superior, lateral e o botão de
   * filtros permanecem.
   *
   * Era um booleano enquanto só a tabela do Resultado tinha o modo. Virou um id
   * quando os blocos da aba Fluxo de Caixa ganharam o mesmo botão: com um
   * booleano por bloco, nada impediria dois expandidos ao mesmo tempo, e cada
   * tela teria a própria versão da regra.
   *
   * Não persiste: é modo de trabalho, não preferência. Voltar dias depois a uma
   * tela sem abas, sem lembrar como entrou, seria uma armadilha.
   */
  const foco = ref<IdFoco | null>(null)

  const haFoco = computed(() => foco.value !== null)

  /**
   * Período que a tela de dados está mostrando.
   *
   * Existe aqui, e não dentro do DashboardView, porque o assistente precisa
   * dele: sem essa referência, "e no mês passado?" não tem a partir de quê, e
   * a resposta viria sobre o mês corrente do servidor — que pode não ser o que
   * a pessoa está olhando.
   *
   * Não persiste: é onde a pessoa está agora, não preferência dela.
   */
  const periodo = ref({ ano: new Date().getFullYear(), mes: new Date().getMonth() + 1 })

  function definirPeriodo(ano: number, mes: number) {
    periodo.value = { ano, mes }
  }

  function emFoco(id: IdFoco) {
    return foco.value === id
  }

  function alternarFoco(id: IdFoco) {
    foco.value = proximoFoco(foco.value, id)
  }

  function sairDoFoco() {
    foco.value = null
  }

  function toggleFiltros() {
    filtrosAbertos.value = !filtrosAbertos.value
    localStorage.setItem('filtros_abertos', String(filtrosAbertos.value))
  }

  /**
   * O atributo no <html> e o localStorage acompanham o ref por watcher, não por
   * atribuição dentro do toggle. Assim qualquer caminho que mude `theme` — e não
   * só o botão — mantém DOM e armazenamento em sincronia.
   *
   * `immediate` cobre o caso de o valor salvo divergir do atributo que o script
   * anti-FOUC do index.html aplicou.
   */
  watch(theme, t => {
    document.documentElement.setAttribute('data-theme', t)
    localStorage.setItem('theme', t)
  }, { immediate: true })

  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
  }

  return { theme, filtrosAbertos, mostrarCentavos, foco, haFoco, periodo,
           toggleTheme, toggleFiltros, toggleCentavos, definirPeriodo,
           emFoco, alternarFoco, sairDoFoco }
})
