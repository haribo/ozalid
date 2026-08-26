<script setup lang="ts">
/**
 * What has been said about a case, and where each report stands.
 *
 * Read-only, deliberately: this is a state, not a control panel. Judging
 * happens in front of the capture, and linking an issue or delivering are API
 * calls made by the dev or their pipeline — ozalid knows no tracker and will
 * never open an issue on anyone's behalf (ADR 0003).
 *
 * Comments are gathered under their step, in the grid's own order, so the table
 * is read as a continuation of it rather than as a separate list. The step's
 * name is the way back to the capture the comments were written on.
 */
import { computed } from 'vue'
import type { components } from '@/shared/api'
import { ActionIcon, KindIcon, StateIcon, VariantHead } from '@/shared/ui'
import type { Tone } from '@/shared/lib'

type Grid = components['schemas']['Grid']
type Comment = components['schemas']['Comment']

const props = defineProps<{ grid: Grid; comments: Comment[] }>()
const emit = defineEmits<{ open: [stepId: string, variantId: string] }>()

const TONE: Record<string, Tone> = {
  'to-track': 'dev',
  tracked: 'dev',
  'to-review': 'reviewer',
  refused: 'dev',
  validated: 'done',
  discarded: 'idle',
}

const PILL: Record<string, string> = {
  idle: 'border-slate-300 bg-slate-100 text-slate-500 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-400',
  reviewer:
    'border-indigo-400 bg-indigo-50 text-indigo-700 dark:border-indigo-500 dark:bg-indigo-950 dark:text-indigo-300',
  dev: 'border-amber-500 bg-amber-50 text-amber-800 dark:border-amber-500 dark:bg-amber-950 dark:text-amber-300',
  done: 'border-emerald-600 bg-emerald-50 text-emerald-800 dark:border-emerald-500 dark:bg-emerald-950 dark:text-emerald-300',
}

/** Nobody is waiting on these: the issue is filed but not delivered, the fix is
 * accepted, the report was set aside. They stay visible — nothing is ever
 * deleted — but they step back so what needs a move keeps the eye. */
const SETTLED = new Set(['tracked', 'validated', 'discarded'])

const openCount = computed(
  () => props.comments.filter((c) => c.state !== 'validated' && c.state !== 'discarded').length,
)

/** One block per step, in the grid's order. A step the grid no longer knows
 * still shows its comments — losing them would be worse than an odd row. */
const groups = computed(() => {
  const rank = new Map(props.grid.steps.map((s, i) => [s.id, i]))
  const byStep = new Map<string, Comment[]>()
  for (const c of props.comments) {
    const bucket = byStep.get(c.stepId)
    if (bucket) bucket.push(c)
    else byStep.set(c.stepId, [c])
  }
  return [...byStep.entries()]
    .map(([stepId, items]) => ({
      stepId,
      name: props.grid.steps.find((s) => s.id === stepId)?.name ?? '—',
      order: rank.get(stepId) ?? Number.MAX_SAFE_INTEGER,
      items,
    }))
    .toSorted((a, b) => a.order - b.order)
})

/** Which capture the step's name opens: the one the first comment was written
 * against, so the reviewer lands on what is being talked about. */
function entryVariant(items: Comment[]) {
  const known = new Set(props.grid.variants.map((v) => v.id))
  return items.flatMap((c) => c.variantIds).find((id) => known.has(id)) ?? props.grid.variants[0]?.id
}

/** The last refusal's remark: it is what the dev has to read, not the state. */
function lastRefusal(c: Comment) {
  return c.judgments.toReversed().find((j) => j.verdict === 'refused')?.remark
}
</script>

<template>
  <div v-if="comments.length" class="mt-5">
    <p class="mb-2 font-mono text-[10.5px] tracking-widest text-slate-500 uppercase">
      commentaires · {{ openCount }} ouvert{{ openCount > 1 ? 's' : '' }} sur {{ comments.length }}
    </p>

    <div class="overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
      <table class="w-full border-collapse text-[13.5px]">
        <thead>
          <tr
            class="bg-slate-50 font-mono text-[10.5px] tracking-widest text-slate-500 uppercase dark:bg-slate-800/60 dark:text-slate-400"
          >
            <th class="px-3 py-2 text-left font-medium">étape</th>
            <th class="px-2 py-2"></th>
            <th class="px-3 py-2 text-left font-medium">ce qui a été dit</th>
            <th
              v-for="v in grid.variants"
              :key="v.id"
              class="w-9 px-1 py-2 text-center font-medium"
            >
              <VariantHead :label="v.label" :values="v.values" compact class="justify-center" />
            </th>
            <th class="px-3 py-2 text-left font-medium">état</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="g in groups" :key="g.stepId">
            <tr
              v-for="(c, i) in g.items"
              :key="c.id"
              class="border-t border-slate-200 dark:border-slate-700"
              :class="SETTLED.has(c.state) ? 'opacity-50' : ''"
            >
              <td
                v-if="i === 0"
                :rowspan="g.items.length"
                class="border-r border-slate-200 px-3 py-2.5 align-middle font-semibold dark:border-slate-700"
              >
                <a
                  v-if="entryVariant(g.items)"
                  href="#"
                  class="border-b border-slate-300 hover:border-indigo-500 hover:text-indigo-700 dark:border-slate-600 dark:hover:text-indigo-300"
                  @click.prevent="emit('open', g.stepId, entryVariant(g.items)!)"
                  >{{ g.name }}</a
                >
                <template v-else>{{ g.name }}</template>
              </td>
              <td class="px-2 py-2.5 text-center align-middle">
                <a
                  v-if="c.issue?.url"
                  :href="c.issue.url"
                  target="_blank"
                  rel="noopener"
                  :title="`issue ${c.issue.id}`"
                  :class="
                    c.kind === 'defect'
                      ? 'text-amber-700 dark:text-amber-400'
                      : 'text-indigo-700 dark:text-indigo-300'
                  "
                >
                  <KindIcon :kind="c.kind" :size="14" class="mx-auto" />
                </a>
                <span
                  v-else
                  :class="
                    c.kind === 'defect'
                      ? 'text-amber-700 dark:text-amber-400'
                      : 'text-indigo-700 dark:text-indigo-300'
                  "
                >
                  <KindIcon :kind="c.kind" :size="14" class="mx-auto" />
                </span>
              </td>
              <td class="max-w-[44ch] px-3 py-2.5 align-top">
                {{ c.body }}
                <a
                  v-if="c.issue"
                  :href="c.issue.url ?? '#'"
                  target="_blank"
                  rel="noopener"
                  class="font-mono text-[11px] text-indigo-700 dark:text-indigo-300"
                  >#{{ c.issue.id }}</a
                >
                <span
                  v-if="c.discardReason"
                  class="mt-1 block font-mono text-[10.5px] text-slate-500 dark:text-slate-400"
                  >écarté : {{ c.discardReason }}</span
                >
                <span
                  v-if="lastRefusal(c)"
                  class="mt-1 flex gap-1.5 font-mono text-[10.5px] text-amber-700 dark:text-amber-400"
                >
                  <ActionIcon name="refuse" :size="12" label="refusé" />{{ lastRefusal(c) }}
                </span>
              </td>
              <td v-for="v in grid.variants" :key="v.id" class="px-1 py-2.5 text-center align-middle">
                <ActionIcon
                  v-if="c.variantIds.includes(v.id)"
                  name="check"
                  :size="13"
                  :label="`s'applique à ${v.label}`"
                  class="mx-auto text-emerald-700 dark:text-emerald-400"
                />
              </td>
              <td class="px-3 py-2.5 align-middle whitespace-nowrap">
                <span
                  class="inline-flex items-center gap-1.5 rounded border px-1.5 py-0.5 font-mono text-[10.5px]"
                  :class="PILL[TONE[c.state]]"
                >
                  <StateIcon :tone="TONE[c.state]" :size="11" :label="c.state" />{{ c.state }}
                </span>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>
  </div>
</template>
