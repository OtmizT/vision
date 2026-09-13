<template>
  <div class="login-page">
    <div class="login-card fade-up">
      <!-- Logo -->
      <div class="login-logo">
        <div class="login-logo-icon">
          <span class="logo-marca" aria-hidden="true" />
          <span class="logo-letra" aria-hidden="true">V</span>
        </div>
        <div>
          <div class="login-logo-name">Visi<span>ON</span></div>
          <div class="login-logo-sub">PAINEL ADMINISTRATIVO</div>
        </div>
      </div>

      <form class="login-form" @submit.prevent="submit">
        <AppInput
          v-model="email"
          label="E-mail"
          type="email"
          placeholder="seu@email.com"
          autocomplete="email"
          :error="errors.email"
        />

        <AppInput
          v-model="password"
          label="Senha"
          type="password"
          placeholder="••••••••"
          autocomplete="current-password"
          :error="errors.password"
        />

        <!-- Erro geral -->
        <p v-if="errorMsg" class="login-error">{{ errorMsg }}</p>

        <AppButton type="submit" :loading="loading" style="width:100%; margin-top:4px">
          Entrar
        </AppButton>
      </form>

      <p class="login-footer">VisiON · Painel Interno</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppInput  from '@/components/ui/AppInput.vue'
import AppButton from '@/components/ui/AppButton.vue'
import { rotaInicial } from '@/utils/navegacao'

const router   = useRouter()
const auth     = useAuthStore()
const email    = ref('')
const password = ref('')
const loading  = ref(false)
const errorMsg = ref('')
const errors   = ref({ email: '', password: '' })

async function submit() {
  errors.value  = { email: '', password: '' }
  errorMsg.value = ''

  if (!email.value)    { errors.value.email    = 'E-mail obrigatório'; return }
  if (!password.value) { errors.value.password = 'Senha obrigatória';  return }

  loading.value = true
  try {
    await auth.login(email.value, password.value)
    // Nem todo papel tem Dashboard; '/' mandaria o admin global para um 403.
    // Senha definida por administrador: a troca vem antes de qualquer tela.
    router.push(auth.senhaProvisoria ? '/trocar-senha' : rotaInicial(auth.contexto))
  } catch (e: any) {
    const msg = e?.response?.data?.message
    errorMsg.value = msg === 'credenciais inválidas'
      ? 'E-mail ou senha incorretos.'
      : (msg ?? 'Erro ao conectar. Tente novamente.')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex; align-items: center; justify-content: center;
  padding: 24px;
}

.login-card {
  width: 100%; max-width: 420px;
  background: var(--surface);
  border: 1px solid var(--border-strong);
  border-radius: var(--r);
  padding: 40px 36px;
  box-shadow: var(--shadow-md);
  position: relative; z-index: 1;
}

.login-logo {
  display: flex; align-items: center; gap: 14px;
  margin-bottom: 36px;
}

.login-logo-icon {
  width: 44px; height: 44px; flex-shrink: 0;
  display: grid; place-items: center;
}
.login-logo-icon .logo-marca { width: 44px; height: 44px; }
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
  .login-logo-icon {
    border-radius: 12px;
    background: linear-gradient(135deg, var(--primary), var(--primary-line));
    font-size: var(--fs-lg); font-weight: 800; color: var(--text-oncolor);
    box-shadow: 0 0 24px var(--primary-weak);
  }
}

.login-logo-name { font-size: var(--fs-lg); font-weight: 800; color: var(--text); }
.login-logo-name span { color: var(--primary); }
.login-logo-sub { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-dim); letter-spacing: 2px; margin-top: 2px; }

.login-form { display: flex; flex-direction: column; gap: 18px; }

.login-error {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--danger);
  background: var(--danger-weak);
  border: 1px solid var(--danger-weak);
  border-radius: 7px;
  padding: 9px 12px;
}

.login-footer {
  text-align: center;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-dim);
  margin-top: 28px;
}
</style>
