<template>
  <div class="input-wrap">
    <label v-if="label" :for="inputId" class="input-label">{{ label }}</label>

    <div :class="['input-field', { 'input-field--error': error, 'input-field--focused': focused }]">
      <slot name="prefix" />

      <input
        :id="inputId"
        :type="currentType"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        :autocomplete="autocomplete"
        class="input-el"
        @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        @focus="focused = true"
        @blur="focused = false"
      />

      <!-- Toggle para senha -->
      <button
        v-if="type === 'password'"
        type="button"
        class="input-eye"
        @click="showPassword = !showPassword"
        tabindex="-1"
      >
        <Eye    v-if="!showPassword" :size="15" />
        <EyeOff v-else               :size="15" />
      </button>
    </div>

    <p v-if="error" class="input-error">{{ error }}</p>
    <p v-else-if="hint" class="input-hint">{{ hint }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Eye, EyeOff } from '@lucide/vue'

const props = withDefaults(defineProps<{
  modelValue?: string
  label?:       string
  placeholder?: string
  type?:        string
  error?:       string
  hint?:        string
  disabled?:    boolean
  autocomplete?: string
}>(), { type: 'text', disabled: false })

defineEmits<{ 'update:modelValue': [value: string] }>()

const focused      = ref(false)
const showPassword = ref(false)
const inputId      = `input-${Math.random().toString(36).slice(2)}`
const currentType  = computed(() => props.type === 'password' && showPassword.value ? 'text' : props.type)
</script>

<style scoped>
.input-wrap { display: flex; flex-direction: column; gap: 6px; }

.input-label {
  font-family: var(--font-body);
  font-size: var(--fs-xs);
  font-weight: 500;
  color: var(--text-dim);
  letter-spacing: 1px;
  text-transform: uppercase;
}

.input-field {
  display: flex;
  align-items: center;
  background: var(--surface-2);
  border: 1px solid var(--border-strong);
  border-radius: 11px;
  transition: border-color 0.2s;
  overflow: hidden;
}

.input-field--focused { border-color: var(--primary); }
.input-field--error   { border-color: var(--danger); }

.input-el {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  padding: 11px 14px;
  font-family: var(--font-body);
  font-size: var(--fs-sm);
  color: var(--text);
  width: 100%;
}
.input-el::placeholder { color: var(--text-dim); }
.input-el:disabled { opacity: 0.5; cursor: not-allowed; }

.input-eye {
  width: 36px; height: 36px;
  display: flex; align-items: center; justify-content: center;
  background: transparent; border: none; cursor: pointer;
  color: var(--text-dim); flex-shrink: 0;
  transition: color 0.2s;
}
.input-eye:hover { color: var(--text-muted); }
.input-eye svg { width: 15px; height: 15px; }

.input-error { font-family: var(--font-body); font-size: var(--fs-xs); color: var(--danger); }
.input-hint  { font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text-dim); }
</style>
