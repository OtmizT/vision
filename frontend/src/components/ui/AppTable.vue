<template>
  <div class="table-card">
    <!-- Header com busca e controles -->
    <div v-if="searchable || $slots.controls" class="table-header">
      <input
        v-if="searchable"
        v-model="searchQuery"
        class="table-search"
        :placeholder="searchPlaceholder"
        @input="$emit('search', searchQuery)"
      />
      <div class="table-controls">
        <slot name="controls" />
      </div>
    </div>

    <!-- Tabela -->
    <div class="table-scroll">
      <table>
        <thead>
          <tr>
            <th
              v-for="col in columns"
              :key="col.key"
              :class="{ sorted: sortKey === col.key }"
              :style="col.width ? { width: col.width } : {}"
              @click="col.sortable !== false ? toggleSort(col.key) : null"
            >
              {{ col.label }}
              <span v-if="col.sortable !== false" class="sort-arrow">
                <svg v-if="sortKey === col.key && sortDir === 'asc'"  viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 15l7-7 7 7"/></svg>
                <svg v-else-if="sortKey === col.key && sortDir === 'desc'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 9l-7 7-7-7"/></svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="opacity:.3"><path d="M8 9l4-4 4 4M8 15l4 4 4-4"/></svg>
              </span>
            </th>
            <th v-if="$slots.actions" style="width:100px; text-align:right">AÇÕES</th>
          </tr>
        </thead>
        <tbody>
          <template v-if="loading">
            <tr v-for="i in skeletonRows" :key="i">
              <td v-for="col in columns" :key="col.key">
                <div class="skeleton" :style="{ width: Math.random() > 0.5 ? '70%' : '50%' }" />
              </td>
              <td v-if="$slots.actions"><div class="skeleton" style="width:60px;margin-left:auto"/></td>
            </tr>
          </template>

          <template v-else-if="rows.length === 0">
            <tr>
              <td :colspan="columns.length + ($slots.actions ? 1 : 0)" style="padding:0">
                <slot name="empty">
                  <EmptyState :title="emptyTitle" />
                </slot>
              </td>
            </tr>
          </template>

          <template v-else>
            <tr v-for="(row, i) in rows" :key="rowKey ? row[rowKey] : i" class="table-row">
              <td v-for="col in columns" :key="col.key">
                <slot :name="`cell-${col.key}`" :row="row" :value="row[col.key]">
                  <span :class="{ 'td-mono': col.mono }">{{ row[col.key] ?? '—' }}</span>
                </slot>
              </td>
              <td v-if="$slots.actions" class="td-actions">
                <slot name="actions" :row="row" />
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <!-- Paginação -->
    <div v-if="total > 0" class="pagination">
      <span class="page-info">
        Mostrando {{ startItem }}–{{ endItem }} de {{ total }}
      </span>
      <div class="page-btns">
        <button class="page-btn" :disabled="page <= 1" @click="$emit('page', page - 1)">‹</button>
        <button
          v-for="p in pageNumbers"
          :key="p"
          :class="['page-btn', { active: p === page, 'page-btn--ellipsis': p === '...' }]"
          :disabled="p === '...'"
          @click="p !== '...' && $emit('page', p as number)"
        >{{ p }}</button>
        <button class="page-btn" :disabled="page >= totalPages" @click="$emit('page', page + 1)">›</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import EmptyState from './EmptyState.vue'

interface Column {
  key:      string
  label:    string
  sortable?: boolean
  mono?:    boolean
  width?:   string
}

const props = withDefaults(defineProps<{
  columns:           Column[]
  rows:              Record<string, any>[]
  loading?:          boolean
  total?:            number
  page?:             number
  perPage?:          number
  rowKey?:           string
  searchable?:       boolean
  searchPlaceholder?: string
  emptyTitle?:       string
  skeletonRows?:     number
}>(), {
  loading: false, total: 0, page: 1, perPage: 20,
  searchable: false, searchPlaceholder: 'Buscar...',
  emptyTitle: 'Nenhum registro encontrado',
  skeletonRows: 5
})

defineEmits<{ search: [q: string]; page: [p: number]; sort: [key: string, dir: string] }>()

const searchQuery = ref('')
const sortKey     = ref('')
const sortDir     = ref<'asc' | 'desc'>('asc')

function toggleSort(key: string) {
  if (sortKey.value === key) sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  else { sortKey.value = key; sortDir.value = 'asc' }
}

const totalPages = computed(() => Math.ceil(props.total / props.perPage))
const startItem  = computed(() => (props.page - 1) * props.perPage + 1)
const endItem    = computed(() => Math.min(props.page * props.perPage, props.total))

const pageNumbers = computed(() => {
  const pages: (number | string)[] = []
  const total = totalPages.value
  const cur   = props.page
  if (total <= 7) {
    for (let i = 1; i <= total; i++) pages.push(i)
  } else {
    pages.push(1)
    if (cur > 3) pages.push('...')
    for (let i = Math.max(2, cur - 1); i <= Math.min(total - 1, cur + 1); i++) pages.push(i)
    if (cur < total - 2) pages.push('...')
    pages.push(total)
  }
  return pages
})
</script>

<style scoped>
.table-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--r);
  overflow: hidden;
}

.table-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
}
.table-controls { display: flex; gap: 6px; margin-left: auto; flex-wrap: wrap; }

.table-scroll { overflow-x: auto; }

table { width: 100%; border-collapse: collapse; }

th {
  font-family: var(--font-body);
  font-size: var(--fs-xs); letter-spacing: 1.5px;
  text-transform: uppercase; color: var(--text-dim);
  padding: 12px 20px; text-align: left;
  background: var(--surface-2);
  border-bottom: 1px solid var(--border);
  cursor: pointer; user-select: none; white-space: nowrap;
  transition: color 0.15s;
}
th:hover { color: var(--text-muted); }
th.sorted { color: var(--primary); }

.sort-arrow { display: inline-block; margin-left: 4px; vertical-align: middle; }
.sort-arrow svg { width: 10px; height: 10px; }

td {
  padding: 12px 20px;
  font-size: var(--fs-sm);
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
  color: var(--text);
}
.table-row:last-child td { border-bottom: none; }
.table-row:hover td { background: var(--surface-2); }

.td-mono { font-family: var(--font-body); font-size: var(--fs-xs); }
.td-actions { text-align: right; }

/* Skeleton */
.skeleton {
  height: 12px; border-radius: 4px;
  background: var(--surface-2);
  animation: shimmer 1.2s ease infinite;
}
@keyframes shimmer {
  0%,100%{opacity:0.5} 50%{opacity:1}
}

/* Pagination */
.pagination {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 18px; border-top: 1px solid var(--border);
  flex-wrap: wrap; gap: 8px;
}
.page-info { font-family: var(--font-body); font-size: var(--fs-xs); color: var(--text-dim); }
.page-btns { display: flex; gap: 3px; }
.page-btn {
  min-width: 30px; height: 30px; padding: 0 6px;
  border-radius: 6px; border: 1px solid var(--border-strong);
  background: var(--surface-2); color: var(--text-muted);
  font-family: var(--font-body); font-size: var(--fs-xs);
  cursor: pointer; display: flex; align-items: center; justify-content: center;
  transition: var(--transition);
}

@media (max-width: 767px) {
  .table-scroll { -webkit-overflow-scrolling: touch; }
  .pagination   { flex-direction: column; align-items: flex-start; gap: 12px; }
}
.page-btn:hover:not(:disabled):not(.page-btn--ellipsis),
.page-btn.active {
  border-color: var(--primary); color: var(--primary); background: var(--primary-weak);
}
.page-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.page-btn--ellipsis { cursor: default; }
</style>
