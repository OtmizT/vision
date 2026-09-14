import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'

/*
 * Fonte auto-hospedada, antes dos tokens que a referenciam.
 *
 * Vinha do Google Fonts por <link> no index.html. Servi-la do próprio nginx
 * tira uma dependência externa do caminho de carregamento e, sobretudo, para de
 * enviar o IP de cada usuário a um terceiro a cada visita — trabalhamos com
 * dados de dezenas de clientes e não há motivo para essa requisição sair daqui.
 *
 * O import é aqui e não por @import no CSS porque o Vite resolve, versiona e
 * copia os .woff2 para o dist; um @import com caminho de node_modules ficaria
 * preso ao layout de pastas do pacote.
 *
 * O arquivo traz todos os subsets (latin, latin-ext, cirílico, vietnamita), mas
 * o unicode-range faz o navegador baixar só o que a página usa: em pt-BR, latin
 * e latin-ext. Os demais ocupam espaço na imagem e nunca são servidos.
 */
import '@fontsource-variable/geist'

/*
 * Registro do Chart.js antes de qualquer componente montar.
 *
 * Estava no DashboardView, o que amarrava todo gráfico àquela tela ter sido
 * importada. O chat do assistente desenha em qualquer rota.
 */
import './utils/chart'

import './assets/tokens.css'
import './assets/main.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
