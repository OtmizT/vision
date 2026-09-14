/**
 * Registro do Chart.js, um lugar só.
 *
 * Isto morava no nível de módulo do DashboardView. Funcionava enquanto todo
 * gráfico vivia dentro daquela tela — mas o chat do assistente desenha gráficos
 * em qualquer rota, e numa página onde o DashboardView nunca foi importado o
 * Chart.js estaria sem escalas, sem controladores e sem o plugin de rótulos: o
 * `new Chart()` falharia com "category is not a registered scale".
 *
 * Importado por main.ts, antes de qualquer componente montar.
 *
 * `registerables` traz tudo (barra, linha, rosca, escalas, tooltip, legenda).
 * Registrar só o que se usa economizaria alguns KB, mas cada gráfico novo
 * exigiria lembrar de registrar a peça dele — e o sintoma de esquecer é um
 * erro em tempo de execução, na tela do usuário.
 */
import { Chart, registerables } from 'chart.js'
import ChartDataLabels from 'chartjs-plugin-datalabels'

Chart.register(...registerables, ChartDataLabels)

export { Chart }
