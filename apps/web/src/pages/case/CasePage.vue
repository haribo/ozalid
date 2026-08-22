<script setup lang="ts">
/** One case: what it is, and the evidence it is judged from. */
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api, type components } from '@/shared/api'
import { StatePill } from '@/shared/ui'
import { formatMoment, type CaseState } from '@/shared/lib'
import { CaseGrid } from '@/widgets/case-grid'

type Case = components['schemas']['Case']
type Grid = components['schemas']['Grid']

const route = useRoute()
const caseId = computed(() => String(route.params.caseId))

const kase = ref<Case | null>(null)
const grid = ref<Grid | null>(null)
const error = ref('')
const loading = ref(true)

// watch rather than watchEffect: what is read after an await is not tracked,
// so navigating from one case to another would keep showing the first.
watch(
  caseId,
  async (id) => {
    loading.value = true
    error.value = ''

    const detail = await api.GET('/cases/{caseId}', { params: { path: { caseId: id } } })
    if (detail.error) {
      error.value = detail.error.title
      loading.value = false
      return
    }
    kase.value = detail.data

    const evidence = await api.GET('/cases/{caseId}/captures', {
      params: { path: { caseId: id } },
    })
    grid.value = evidence.error ? null : evidence.data
    loading.value = false
  },
  { immediate: true },
)
</script>

<template>
  <div class="mx-auto max-w-6xl px-6 py-8">
    <p v-if="error" class="font-mono text-[12px] text-red-700 dark:text-red-400">
      {{ error }}
    </p>
    <p v-else-if="loading" class="font-mono text-[12px] text-slate-500">chargement…</p>

    <template v-else-if="kase">
      <h1 class="mb-2.5 text-[21px] font-semibold">
        {{ kase.title }}
      </h1>

      <div
        class="mb-3 flex flex-wrap items-center gap-x-3 gap-y-1.5 font-mono text-[11px] text-slate-500 dark:text-slate-400"
      >
        <StatePill :state="kase.state as CaseState" />
        <span>#{{ kase.id }}</span>
        <template v-if="grid?.editionId">
          <span>·</span>
          <span>édition du {{ formatMoment(grid.takenAt) }}</span>
          <template v-if="grid.revision">
            <span>·</span>
            <span>rev {{ grid.revision }}</span>
          </template>
        </template>
      </div>

      <CaseGrid v-if="grid" :grid="grid" />
    </template>
  </div>
</template>
