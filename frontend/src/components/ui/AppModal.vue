<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="modelValue" class="modal-overlay" @mousedown.self="onBackdrop">
        <div :class="['modal-box', `modal-box--${size}`]" role="dialog" aria-modal="true">
          <!-- Header -->
          <div class="modal-header">
            <div>
              <p class="modal-title">{{ title }}</p>
              <p v-if="subtitle" class="modal-subtitle">{{ subtitle }}</p>
            </div>
            <button class="modal-close" @click="$emit('update:modelValue', false)" type="button">
              <X :size="13" />
            </button>
          </div>

          <!-- Body -->
          <div class="modal-body">
            <slot />
          </div>

          <!-- Footer -->
          <div v-if="$slots.footer" class="modal-footer">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { X } from '@lucide/vue'

withDefaults(defineProps<{
  modelValue: boolean
  title?:     string
  subtitle?:  string
  size?:      'sm' | 'md' | 'lg'
  persistent?: boolean
}>(), { size: 'md', persistent: false })

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

function onBackdrop() {
  emit('update:modelValue', false)
}
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0; z-index: 1000;
  background: var(--overlay);
  backdrop-filter: blur(4px);
  display: flex; align-items: center; justify-content: center;
  padding: 24px;
}

.modal-box {
  background: var(--surface);
  border: 1px solid var(--border-strong);
  border-radius: var(--r-lg);
  box-shadow: var(--shadow-md);
  display: flex; flex-direction: column;
  max-height: calc(100vh - 48px);
  width: 100%;
  overflow: hidden;
}

.modal-box--sm { max-width: 400px; }
.modal-box--md { max-width: 540px; }
.modal-box--lg { max-width: 720px; }

.modal-header {
  display: flex; align-items: flex-start; justify-content: space-between;
  padding: 24px 28px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.modal-title    { font-size: var(--fs-md); font-weight: 700; color: var(--text); }
.modal-subtitle { font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text-dim); margin-top: 3px; }

.modal-close {
  width: 28px; height: 28px;
  border-radius: 7px; border: 1px solid var(--border-strong);
  background: var(--surface-2); color: var(--text-dim); cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0; transition: var(--transition);
}
.modal-close:hover { border-color: var(--danger); color: var(--danger); }
.modal-close svg  { width: 13px; height: 13px; }

.modal-body {
  padding: 24px 28px;
  overflow-y: auto;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.modal-footer {
  padding: 16px 28px;
  border-top: 1px solid var(--border);
  display: flex; justify-content: flex-end; gap: 8px;
  flex-shrink: 0;
}

@media (max-width: 600px) {
  .modal-overlay { padding: 12px; }
  .modal-box--lg,
  .modal-box--md { max-width: 100%; }
}

/* Transitions */
.modal-enter-active, .modal-leave-active { transition: opacity 0.2s ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
.modal-enter-active .modal-box { transition: transform 0.22s cubic-bezier(0.34,1.56,0.64,1); }
.modal-enter-from .modal-box   { transform: scale(0.95) translateY(8px); }
</style>
