<template>
  <div class="cg-card">
    <div v-if="spec?.titulo" class="cg-titulo">{{ spec.titulo }}</div>
    <div class="cg-wrap">
      <canvas ref="canvasEl" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { Chart } from '@/utils/chart'
import { useUiStore } from '@/stores/ui'
import { configDoGrafico, type VisaoSpec } from '@/utils/visaospec'

const props = defineProps<{ spec: VisaoSpec }>()

const canvasEl = ref<HTMLCanvasElement | null>(null)
// Ver a nota em GraficoDonut.vue: a classe é invariante nos genéricos, então o
// alargamento vai na atribuição.
let chart: Chart | null = null

// O tema vem da store porque as cores do gráfico são lidas do CSS no momento
// da criação — trocar de tema com o chat aberto deixaria o gráfico com a
// paleta antiga até alguém rolar a conversa.
const { theme } = storeToRefs(useUiStore())

function desenhar() {
  if (!canvasEl.value) return

  // Destruir antes de recriar: sem isso o Chart.js mantém a instância velha
  // presa ao canvas e os dois passam a responder ao mesmo mouse.
  chart?.destroy()
  chart = null

  const cfg = configDoGrafico(props.spec)
  // Spec inválida não vira gráfico. O texto da resposta continua na bolha
  // acima — desenhar errado seria pior, porque o errado parece certo.
  if (!cfg) return

  chart = new Chart(canvasEl.value, cfg) as Chart
}

watch(() => props.spec, () => nextTick(desenhar), { deep: true })
watch(theme, () => nextTick(desenhar))
onMounted(() => nextTick(desenhar))

/*
 * Destruir no unmount não é zelo, é necessidade: cada Chart.js registra
 * listeners de resize na janela. Numa conversa longa, cada gráfico que saísse
 * do DOM sem destruir deixaria os seus para trás.
 */
onBeforeUnmount(() => {
  chart?.destroy()
  chart = null
})
</script>

<style scoped>
.cg-card {
  margin-top: 10px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  padding: 12px;
}

.cg-titulo {
  font-size: var(--fs-xs);
  font-weight: 600;
  color: var(--text-muted);
  margin-bottom: 8px;
  letter-spacing: .02em;
}

/*
 * Altura explícita: o Chart.js roda com maintainAspectRatio desligado, então
 * quem define a altura é este contêiner. Sem ela o canvas realimenta o pai e a
 * bolha cresce a cada quadro.
 */
.cg-wrap {
  position: relative;
  height: 200px;
}

.cg-wrap canvas {
  position: absolute;
  inset: 0;
}
</style>
