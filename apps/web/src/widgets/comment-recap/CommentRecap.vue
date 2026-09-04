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
type IssueTracking = components['schemas']['IssueTracking']

/** One table row: a ref on its own round, or the draft a comment still is.
 * The comment's text is what the issues are written from; once a ref exists,
 * the book reads the issue's title (#138). */
type Row = { comment: Comment; ref: IssueTracking | null; first: boolean }

const props = defineProps<{ grid: Grid; comments: Comment[] }>()
const emit = defineEmits<{ open: [stepId: string, variantId: string] }>()

const TONE: Record<string, Tone> = {
  'to-track': 'dev',
  tracked: 'idle',
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
  const byStep = new Map<string, Row[]>()
  for (const c of props.comments) {
    const rows: Row[] =
      c.issues && c.issues.length > 0
        ? c.issues.map((ref, i) => ({ comment: c, ref, first: i === 0 }))
        : [{ comment: c, ref: null, first: true }]
    const bucket = byStep.get(c.stepId)
    if (bucket) bucket.push(...rows)
    else byStep.set(c.stepId, rows)
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
function entryVariant(items: Row[]) {
  const known = new Set(props.grid.variants.map((v) => v.id))
  return (
    items.flatMap((r) => r.comment.variantIds).find((id) => known.has(id)) ??
    props.grid.variants[0]?.id
  )
}

/** What one row is doing: the ref's own state, or the comment's while it is
 * still a draft or was discarded. */
function rowState(r: Row) {
  if (r.comment.state === 'discarded') return 'discarded'
  return r.ref ? r.ref.state : r.comment.state
}
</script>

<template>
  <div v-if="comments.length" class="mt-5">
    <p class="mb-2 font-mono text-[10.5px] tracking-widest text-slate-500 uppercase">
      comments · {{ openCount }} open of {{ comments.length }}
    </p>

    <div class="overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
      <table class="w-full border-collapse text-[13.5px]">
        <thead>
          <tr
            class="bg-slate-50 font-mono text-[10.5px] tracking-widest text-slate-500 uppercase dark:bg-slate-800/60 dark:text-slate-400"
          >
            <th class="px-3 py-2 text-left font-medium">step</th>
            <th class="px-2 py-2"></th>
            <th class="px-3 py-2 text-left font-medium">what was said</th>
            <th
              v-for="v in grid.variants"
              :key="v.id"
              class="w-9 px-1 py-2 text-center font-medium"
            >
              <VariantHead :label="v.label" :values="v.values" compact class="justify-center" />
            </th>
            <th class="px-3 py-2 text-left font-medium">state</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="g in groups" :key="g.stepId">
            <tr
              v-for="(r, i) in g.items"
              :key="(r.ref?.id ?? r.comment.id) + i"
              class="border-t border-slate-200 dark:border-slate-700"
              :class="{
                'opacity-50': SETTLED.has(rowState(r)),
                'opacity-45': rowState(r) === 'tracked',
              }"
            >
              <td
                v-if="i === 0"
                :rowspan="g.items.length"
                class="border-r border-slate-200 px-3 py-2.5 align-middle font-semibold opacity-100 dark:border-slate-700"
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
                <span
                  :class="
                    r.comment.kind === 'defect'
                      ? 'text-amber-700 dark:text-amber-400'
                      : 'text-indigo-700 dark:text-indigo-300'
                  "
                >
                  <KindIcon :kind="r.comment.kind" :size="14" class="mx-auto" />
                </span>
              </td>
              <td class="max-w-[44ch] px-3 py-2.5 align-top">
                <!-- The issue's title once tracked; the free text was the
                     draft it was written from (#138). -->
                <template v-if="r.ref">
                  <a
                    :href="r.ref.url ?? '#'"
                    target="_blank"
                    rel="noopener"
                    class="text-indigo-700 dark:text-indigo-300"
                    ><span class="font-mono text-[11px]">#{{ r.ref.issueId }}</span>
                    {{ r.ref.title }}</a
                  >
                  <span
                    v-if="r.ref.lastRefusal"
                    class="mt-1 flex gap-1.5 font-mono text-[10.5px] text-amber-700 dark:text-amber-400"
                  >
                    <ActionIcon name="refuse" :size="12" label="refused" />{{ r.ref.lastRefusal }}
                  </span>
                </template>
                <template v-else>
                  <span class="italic">{{ r.comment.body }}</span>
                </template>
                <span
                  v-if="r.first && r.comment.discardReason"
                  class="mt-1 block font-mono text-[10.5px] text-slate-500 dark:text-slate-400"
                  >discarded: {{ r.comment.discardReason }}</span
                >
              </td>
              <td
                v-for="v in grid.variants"
                :key="v.id"
                class="px-1 py-2.5 text-center align-middle"
              >
                <ActionIcon
                  v-if="r.comment.variantIds.includes(v.id)"
                  name="check"
                  :size="13"
                  :label="`applies to ${v.label}`"
                  class="mx-auto text-emerald-700 dark:text-emerald-400"
                />
              </td>
              <td class="px-3 py-2.5 align-middle whitespace-nowrap">
                <span
                  class="inline-flex items-center gap-1.5 rounded border px-1.5 py-0.5 font-mono text-[10.5px]"
                  :class="PILL[TONE[rowState(r)]]"
                >
                  <StateIcon :tone="TONE[rowState(r)]" :size="11" :label="rowState(r)" />{{
                    rowState(r)
                  }}
                </span>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>
  </div>
</template>
