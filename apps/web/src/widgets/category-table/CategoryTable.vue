<script setup lang="ts">
/**
 * The catalogue at one level: each row a category, with the gauge of its
 * whole branch. Aggregating on the descendance is what makes a branch in
 * trouble visible without descending into it.
 */
import { RouterLink } from 'vue-router'
import { StateGauge } from '@/shared/ui'
import { formatMoment } from '@/shared/lib'
import type { components } from '@/shared/api'

type Category = components['schemas']['Category']

const props = defineProps<{ slug: string; categories: Category[] }>()

function parts(c: Category) {
  return [
    { tone: 'done' as const, count: c.cases.reviewed, label: 'relus' },
    { tone: 'reviewer' as const, count: c.cases.toReview, label: 'to review' },
    { tone: 'dev' as const, count: c.cases.toFix, label: 'to fix' },
    { tone: 'idle' as const, count: c.cases.notInstrumented, label: 'not instrumented' },
  ]
}

function total(c: Category) {
  return c.cases.reviewed + c.cases.toReview + c.cases.toFix + c.cases.notInstrumented
}
</script>

<template>
  <div class="overflow-hidden rounded-md border border-slate-200 dark:border-slate-700">
    <table class="w-full border-collapse text-body">
      <thead>
        <tr
          class="bg-slate-50 font-mono text-label tracking-widest text-slate-500 uppercase dark:bg-slate-800/60 dark:text-slate-400"
        >
          <th class="px-3 py-2 text-left font-medium">category</th>
          <th class="w-1/3 px-3 py-2 text-left font-medium">progress</th>
          <th class="px-3 py-2 text-right font-medium">cases</th>
          <th class="px-3 py-2 text-left font-medium whitespace-nowrap">last updated</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="c in props.categories"
          :key="c.id"
          class="border-t border-slate-200 dark:border-slate-700"
        >
          <td class="px-3 py-2.5 font-semibold">
            <RouterLink
              :to="`/projects/${props.slug}/categories/${c.id}`"
              class="border-b border-slate-300 hover:border-current dark:border-slate-600"
            >
              {{ c.name }}
            </RouterLink>
          </td>
          <td class="px-3 py-2.5">
            <StateGauge :parts="parts(c)" />
          </td>
          <td class="px-3 py-2.5 text-right font-mono text-mono text-slate-600 dark:text-slate-300">
            {{ total(c) }}
          </td>
          <td
            class="px-3 py-2.5 font-mono text-mono whitespace-nowrap text-slate-500 dark:text-slate-400"
          >
            {{ formatMoment(c.lastActivity) }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
