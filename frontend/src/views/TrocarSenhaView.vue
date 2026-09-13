<template>
  <div class="troca-page">
    <div class="card">
      <div class="logo">
        <div class="logo-icon"><span class="logo-marca" aria-hidden="true" /></div>
        <div class="logo-text">Visi<span>ON</span></div>
      </div>

      <h2 class="title">Defina sua senha</h2>
      <p class="subtitle">
        Sua senha atual foi definida por um administrador e vale só para este
        primeiro acesso. Escolha uma senha sua para continuar.
      </p>

      <div class="campos">
        <div class="field">
          <label for="atual">SENHA ATUAL</label>
          <input id="atual" v-model="atual" type="password" class="input-el"
                 placeholder="A senha que você recebeu" autocomplete="current-password" />
        </div>
        <div class="field">
          <label for="nova">NOVA SENHA</label>
          <input id="nova" v-model="nova" type="password" class="input-el"
                 placeholder="Mínimo 8 caracteres" autocomplete="new-password" />
        </div>
        <div class="field">
          <label for="conf">CONFIRMAR SENHA</label>
          <input id="conf" v-model="conf" type="password" class="input-el"
                 placeholder="Repita a senha nova" autocomplete="new-password"
                 @keyup.enter="salvar" />
        </div>

        <p v-if="erro" class="erro">{{ erro }}</p>

        <button class="btn-primary" :disabled="salvando" @click="salvar">
          {{ salvando ? 'Salvando...' : 'Salvar e continuar' }}
        </button>

        <!-- Sair precisa continuar possível: alguém que recebeu a senha errada
             ficaria preso nesta tela sem nenhuma saída. -->
        <button class="link-sair" @click="sair">Sair da conta</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { rotaInicial } from '@/utils/navegacao'

const auth = useAuthStore()
const router = useRouter()

const atual = ref('')
const nova = ref('')
const conf = ref('')
const erro = ref('')
const salvando = ref(false)

async function salvar() {
  erro.value = ''
  if (!atual.value) { erro.value = 'Informe a senha atual.'; return }
  if (nova.value.length < 8) { erro.value = 'A senha nova precisa ter ao menos 8 caracteres.'; return }
  if (nova.value !== conf.value) { erro.value = 'As senhas não conferem.'; return }
  if (nova.value === atual.value) { erro.value = 'A senha nova precisa ser diferente da atual.'; return }

  salvando.value = true
  try {
    await auth.trocarSenha(atual.value, nova.value)
    router.replace(rotaInicial(auth.contexto))
  } catch (e: unknown) {
    const r = e as { response?: { data?: { message?: string } } }
    erro.value = r?.response?.data?.message ?? 'Não foi possível trocar a senha.'
  } finally {
    salvando.value = false
  }
}

async function sair() {
  await auth.logout()
  router.replace('/login')
}
</script>

<style scoped>
.troca-page {
  min-height: 100vh;
  background: var(--bg);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.card {
  width: 100%;
  max-width: 440px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 40px 32px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.logo { display: flex; align-items: center; gap: 10px; }
.logo-icon { width: 36px; height: 36px; display: grid; place-items: center; }
/* A logo entra como MASCARA e nao como imagem: o PNG tem a forma no canal alfa
   e a cor vem do token, entao a marca acompanha os dois temas. */
.logo-marca {
  display: block; width: 36px; height: 36px;
  background: var(--primary);
  -webkit-mask: url('/logo-vision.png') center / contain no-repeat;
          mask: url('/logo-vision.png') center / contain no-repeat;
}
.logo-text { font-size: var(--fs-lg); font-weight: 800; color: var(--text); }
.logo-text span { color: var(--primary); }

.title { font-size: var(--fs-lg); font-weight: 700; color: var(--text); margin: 0; }
.subtitle { font-size: var(--fs-sm); color: var(--text-muted); margin: 0; line-height: 1.5; }

.campos { display: flex; flex-direction: column; gap: 14px; }
.field { display: flex; flex-direction: column; gap: 6px; }

label {
  font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-dim);
  letter-spacing: 1.5px; text-transform: uppercase;
}

.input-el {
  background: var(--surface); border: 1px solid var(--border-strong); border-radius: 8px;
  padding: 10px 12px; font-size: var(--fs-sm); color: var(--text);
  outline: none; transition: border-color 0.2s;
}
.input-el:focus { border-color: var(--primary); }

.erro {
  font-family: var(--font-display); font-size: var(--fs-xs); color: var(--danger);
  background: var(--danger-weak); border-radius: 8px; padding: 10px 12px; margin: 0;
}

.link-sair {
  background: none; border: none; color: var(--text-dim);
  font-size: var(--fs-xs); cursor: pointer; padding: 4px;
  align-self: center; text-decoration: underline;
}
.link-sair:hover { color: var(--text-muted); }
</style>
