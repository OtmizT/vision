import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'

/*
 * Fontes auto-hospedadas, antes dos tokens que as referenciam.
 *
 * Vinham do Google Fonts por <link> no index.html. Servi-las do próprio nginx
 * tira uma dependência externa do caminho de carregamento e, sobretudo, para de
 * enviar o IP de cada usuário a um terceiro a cada visita — trabalhamos com
 * dados de dezenas de clientes e não há motivo para essa requisição sair daqui.
 *
 * O import é aqui e não por @import no CSS porque o Vite resolve, versiona e
 * copia os .woff2 para o dist; um @import com caminho de node_modules ficaria
 * preso ao layout de pastas do pacote.
 *
 * Cada arquivo traz todos os subsets (latin, latin-ext, cirílico, vietnamita),
 * mas o unicode-range faz o navegador baixar só o que a página usa: em pt-BR,
 * latin e latin-ext. Os demais ocupam espaço na imagem e nunca são servidos.
 */
import '@fontsource-variable/geist'
import '@fontsource-variable/geist-mono'

import './assets/tokens.css'
import './assets/main.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
