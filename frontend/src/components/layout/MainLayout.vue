<template>
  <div class="layout">
    <AppSidebar :mobile-open="mobileOpen" @close="mobileOpen = false" />

    <main class="main-content">
      <!--
        Cabeçalho fixo com as três faixas empilhadas. Sticky em UM contêiner e
        não em cada faixa: a barra de filtros muda de altura ao quebrar linha, e
        um `top` fixo por faixa sairia do lugar.

        Os dois divs abaixo são alvos de Teleport preenchidos pelas views. Ficam
        vazios nas rotas sem filtros ou abas, e a regra :empty os retira do fluxo
        para não deixarem faixa em branco.
      -->
      <div class="appbar">
        <AppTopbar @toggle-sidebar="toggleSidebar" />
        <div v-show="ui.filtrosAbertos" id="topbar-filters" class="appbar-filtros"></div>
        <div id="appbar-tabs" class="appbar-abas"></div>
      </div>

      <div class="page-content">
        <RouterView v-slot="{ Component }">
          <Transition name="fade-page" mode="out-in">
            <component :is="Component" :key="$route.path" />
          </Transition>
        </RouterView>
      </div>
    </main>

    <!--
      O assistente vive aqui, e nao dentro de cada view: ele acompanha a pessoa
      em todas as telas do dashboard. O componente se teleporta para o body e
      some sozinho quando o grupo nao tem o recurso habilitado.
    -->
    <ChatFlutuante />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import AppSidebar from './AppSidebar.vue'
import AppTopbar  from './AppTopbar.vue'
import ChatFlutuante from '@/components/ia/ChatFlutuante.vue'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()
const mobileOpen = ref(false)

function toggleSidebar() {
  if (window.innerWidth <= 768) {
    mobileOpen.value = !mobileOpen.value
  }
}
</script>

<style scoped>
.layout { display: flex; min-height: 100vh; }

.main-content {
  margin-left: var(--sidebar-w);
  flex: 1; min-width: 0;
  position: relative; z-index: 1;
  transition: margin-left 0.28s cubic-bezier(0.4,0,0.2,1);
}

@media (max-width: 768px) {
  .main-content { margin-left: 0 !important; }
}

/* ── Cabeçalho fixo ── */
.appbar {
  position: sticky; top: 0; z-index: 50;
  background: var(--topbar-bg);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--border);
}
.appbar-filtros,
.appbar-abas {
  padding: 0 var(--sp-6);
  border-top: 1px solid var(--border);
}
/* Alvo de Teleport vazio não deve ocupar altura nem desenhar borda. */
.appbar-filtros:empty,
.appbar-abas:empty { display: none; }

@media (max-width: 1023px) { .appbar-filtros, .appbar-abas { padding: 0 var(--sp-5); } }
@media (max-width: 767px)  { .appbar-filtros, .appbar-abas { padding: 0 var(--sp-4); } }

.page-content { padding: var(--sp-6); }
@media (max-width: 1023px) { .page-content { padding: var(--sp-5); } }
@media (max-width: 767px)  { .page-content { padding: var(--sp-4); } }

.fade-page-enter-active, .fade-page-leave-active { transition: opacity 0.18s ease, transform 0.18s ease; }
.fade-page-enter-from { opacity: 0; transform: translateY(6px); }
.fade-page-leave-to  { opacity: 0; transform: translateY(-6px); }
</style>
