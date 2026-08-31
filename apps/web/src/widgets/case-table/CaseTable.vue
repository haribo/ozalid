<script setup lang="ts">
/**
 * The cases of one category. No filter in v1.
 *
 * The gauge here counts *captures*, not cases: the same bar, one level down.
 * Its counted legend is what keeps the two readings apart.
 */
import { RouterLink } from 'vue-router'
import { StateGauge, StatePill } from '@/shared/ui'
import { formatMoment, type CaseState } from '@/shared/lib'
import type { components } from '@/shared/api'

type Case = components['schemas']['Case']

const props = defineProps<{ slug: string; cases: Case[] }>()

function parts(c: Case) {
  const k = c.captures
  if (!k) return []
  return [
    { tone: 'done' as const, count: k.validated, label: 'validées' },
    { tone: 'reviewer' as const, count: k.toJudge, label: 'à juger' },
    { tone: 'dev' as const, count: k.commented, label: 'commentées' },
  ]
}
</script>

<template>
  <div class="overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
    <table class="w-full border-collapse text-[13.5px]">
      <thead>
        <tr
          class="bg-slate-50 font-mono text-[10.5px] tracking-widest text-slate-500 uppercase dark:bg-slate-800/60 dark:text-slate-400"
        >
          <th class="px-3 py-2 text-left font-medium">id</th>
          <th class="px-3 py-2 text-left font-medium">cas</th>
          <th class="px-3 py-2 text-left font-medium">état</th>
          <th class="w-1/4 px-3 py-2 text-left font-medium">captures</th>
          <th class="px-3 py-2 text-left font-medium whitespace-nowrap">dernière mise à jour</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="c in props.cases"
          :key="c.id"
          class="border-t border-slate-200 dark:border-slate-700"
        >
          <td
            class="px-3 py-2.5 font-mono text-[11px] whitespace-nowrap text-slate-500 dark:text-slate-400"
          >
            {{ c.id }}
          </td>
          <td class="px-3 py-2.5 font-semibold">
            <RouterLink
              :to="`/projects/${slug}/cases/${c.id}`"
              class="border-b border-slate-300 hover:border-current dark:border-slate-600"
            >
              {{ c.title }}
            </RouterLink>
          </td>
          <td class="px-3 py-2.5">
            <StatePill :state="c.state as CaseState" />
          </td>
          <td class="px-3 py-2.5">
            <StateGauge :parts="parts(c)" />
          </td>
          <td
            class="px-3 py-2.5 font-mono text-[11px] whitespace-nowrap text-slate-500 dark:text-slate-400"
          >
            {{ formatMoment(c.lastEdition) }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
