import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { iaApi, type MensagemChat, type ContextoTela } from '@/api/ia'

/**
 * Estado do assistente.
 *
 * O histórico NÃO é persistido em localStorage, e é de propósito: ele vive no
 * servidor, por usuário e por grupo. Guardar cópia aqui criaria duas verdades —
 * e faria dados financeiros de cliente ficarem no disco da máquina de quem usa,
 * fora do alcance do expurgo de 90 dias.
 *
 * O que fica em localStorage é só a preferência de o painel estar aberto, que é
 * conveniência e não dado.
 */
export const useIaStore = defineStore('ia', () => {
  const aberto = ref(localStorage.getItem('ia_aberto') === 'true')
  const disponivel = ref(false)
  const mensagens = ref<MensagemChat[]>([])
  const carregando = ref(false)
  const enviando = ref(false)
  const erro = ref('')

  const vazio = computed(() => mensagens.value.length === 0)

  function alternar() {
    aberto.value = !aberto.value
    localStorage.setItem('ia_aberto', String(aberto.value))
    if (aberto.value && vazio.value) void carregarConversa()
  }

  function fechar() {
    aberto.value = false
    localStorage.setItem('ia_aberto', 'false')
  }

  /**
   * Pergunta ao servidor se o recurso está ligado para este grupo.
   *
   * A resposta decide se o botão aparece. Falha de rede assume indisponível:
   * melhor o botão não aparecer que aparecer e não funcionar.
   */
  async function verificarDisponibilidade() {
    try {
      const { data } = await iaApi.disponivel()
      disponivel.value = !!data.data?.disponivel
    } catch {
      disponivel.value = false
    }
  }

  async function carregarConversa() {
    carregando.value = true
    try {
      const { data } = await iaApi.conversa()
      mensagens.value = data.data ?? []
    } catch {
      // Sem histórico a conversa começa vazia — não vale interromper a tela
      // com um erro por causa disso.
    } finally {
      carregando.value = false
    }
  }

  async function perguntar(pergunta: string, contexto: ContextoTela) {
    const texto = pergunta.trim()
    if (!texto || enviando.value) return

    erro.value = ''
    // A pergunta aparece na hora, antes da ida ao servidor: a pessoa precisa
    // ver o que escreveu enquanto espera.
    mensagens.value.push({ papel: 'usuario', conteudo: texto })
    enviando.value = true

    try {
      const { data } = await iaApi.perguntar(texto, contexto)
      const r = data.data
      mensagens.value.push({
        papel: 'assistente',
        conteudo: r.texto,
        grafico: r.grafico,
        fonte: r.fonte,
      })
    } catch (e: unknown) {
      const resp = e as { response?: { data?: { message?: string } } }
      const msg = resp?.response?.data?.message ?? 'Não consegui responder agora.'
      erro.value = msg
      // A falha entra na conversa como bolha, e não só numa faixa de erro: a
      // pergunta continua visível acima dela, e fica claro o que falhou.
      mensagens.value.push({ papel: 'assistente', conteudo: msg, erro: true })
    } finally {
      enviando.value = false
    }
  }

  async function limpar() {
    await iaApi.limpar()
    mensagens.value = []
    erro.value = ''
  }

  return {
    aberto, disponivel, mensagens, carregando, enviando, erro, vazio,
    alternar, fechar, verificarDisponibilidade, carregarConversa, perguntar, limpar,
  }
})
