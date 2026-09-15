<script setup lang="ts">
/**
 * What has been said about a case, and where each report stands.
 *
 * Read-only, deliberately: this is a state, not a control panel. Judging
 * happens in front of the capture, and linking an issue or delivering are API
 * calls made by the dev or their pipeline — ozalid knows no tracker and will
 * never open an issue on anyone's behalf (ADR 0003).
 *
 * The table says each verdict once, in its variant's column (#213): a block —
 * an issue or a bare remark, with its standing refusals — puts at most one
 * mark per variant column. A standing refusal is its remark (ADR 0020): it
 * renders as its own line, the text beside the ✗ in the refused variant's
 * column, and the claim line never carries a ✗. No aggregate state: it cannot
 * say "three accepted, one refused".
 *
 * Comments are gathered under their step, in the grid's own order, so the table
 * is read as a continuation of it rather than as a separate list. The step's
 * name is the way back to the capture the comments were written on.
 */
import { computed } from 'vue'
import type { components } from '@/shared/api'
import { VariantHead } from '@/shared/ui'

type Grid = components['schemas']['Grid']
type Comment = components['schemas']['Comment']
type IssueTracking = components['schemas']['IssueTracking']

/** A claim line: a ref on its own round, or the draft a comment still is.
 * The comment's text is what the issues are written from; once a ref exists,
 * the book reads the issue's title (#138). */
type Claim = { kind: 'claim'; comment: Comment; ref: IssueTracking | null; first: boolean }
/** A refusal line: one standing refusal, anchored by its variant's column.
 * Without a variant (history from before ADR 0022) it anchors on every
 * variant the claim still covers. */
type Refusal = { kind: 'refusal'; comment: Comment; variantId?: string; remark: string }
type Line = Claim | Refusal

const props = defineProps<{ grid: Grid; comments: Comment[] }>()
const emit = defineEmits<{ open: [stepId: string, variantId: string] }>()

/** Nobody is waiting on these: the issue is filed but not delivered, the fix is
 * accepted, the report was set aside. They stay visible — nothing is ever
 * deleted — but they step back so what needs a move keeps the eye. */
const SETTLED = new Set(['tracked', 'accepted', 'discarded'])

/** The standing refusals of one claim. A ref carries them from the server
 * (#212). A bare remark has no ref to carry them, so while it reads refused
 * its last per-variant refusal speaks (#175). */
function refusalsOf(c: Comment, ref: IssueTracking | null): Refusal[] {
  if (ref) {
    return (ref.refusals ?? []).map((r) => ({
      kind: 'refusal',
      comment: c,
      variantId: r.variantId,
      remark: r.remark,
    }))
  }
  if (c.state !== 'refused') return []
  const last = new Map<string | undefined, components['schemas']['Judgment']>()
  for (const j of c.judgments ?? []) last.set(j.variantId, j)
  return [...last.values()]
    .filter((j) => j.verdict === 'refused')
    .map((j) => ({ kind: 'refusal', comment: c, variantId: j.variantId, remark: j.remark ?? '' }))
}

/** One block per step, in the grid's order. A step the grid no longer knows
 * still shows its comments — losing them would be worse than an odd row. */
const groups = computed(() => {
  const rank = new Map(props.grid.steps.map((s, i) => [s.id, i]))
  const byStep = new Map<string, Line[]>()
  for (const c of props.comments) {
    const claims: Claim[] =
      c.issues && c.issues.length > 0
        ? c.issues.map((ref, i) => ({ kind: 'claim', comment: c, ref, first: i === 0 }))
        : [{ kind: 'claim', comment: c, ref: null, first: true }]
    const lines = claims.flatMap((claim) => [claim, ...refusalsOf(c, claim.ref)])
    const bucket = byStep.get(c.stepId)
    if (bucket) bucket.push(...lines)
    else byStep.set(c.stepId, lines)
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
function entryVariant(items: Line[]) {
  const known = new Set(props.grid.variants.map((v) => v.id))
  return (
    items.flatMap((l) => l.comment.variantIds).find((id) => known.has(id)) ??
    props.grid.variants[0]?.id
  )
}

/** Did an acceptance land on this capture? Read off the judgment history —
 * accepting released the variant from the coverage (ADR 0022). Rounds judged
 * before that rule kept their coverage, so a settled claim still covering the
 * variant reads accepted too. */
function acceptedOn(claim: Claim, variantId: string) {
  const last = (claim.comment.judgments ?? []).filter((j) => j.variantId === variantId).at(-1)
  if (last?.verdict === 'accepted') return true
  const state = claim.ref ? claim.ref.state : claim.comment.state
  return state === 'accepted' && claim.comment.variantIds.includes(variantId)
}

/** The one mark a line puts in a variant's column. */
function markOf(line: Line, variantId: string): '✓' | '✗' | '·' | '' {
  if (line.kind === 'refusal') {
    if (line.variantId) return line.variantId === variantId ? '✗' : ''
    return line.comment.variantIds.includes(variantId) ? '✗' : ''
  }
  if (acceptedOn(line, variantId)) return '✓'
  if (!line.comment.variantIds.includes(variantId)) return ''
  const standing = refusalsOf(line.comment, line.ref)
  if (standing.some((r) => !r.variantId || r.variantId === variantId)) return ''
  return '·'
}

/** A line steps back when nobody has to move on it. Refusal lines never do:
 * they are exactly what the dev has to read. */
function dimmed(line: Line) {
  if (line.kind === 'refusal') return false
  if (line.comment.state === 'discarded') return true
  return SETTLED.has(line.ref ? line.ref.state : line.comment.state)
}
</script>

<template>
  <div v-if="comments.length" class="mt-5">
    <div class="overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
      <table class="w-full border-collapse text-body">
        <thead>
          <tr
            class="bg-slate-50 font-mono text-label tracking-widest text-slate-500 uppercase dark:bg-slate-800/60 dark:text-slate-400"
          >
            <th class="px-3 py-2 text-left font-medium">step</th>
            <th class="px-3 py-2 text-left font-medium">issue / comment</th>
            <th
              v-for="v in grid.variants"
              :key="v.id"
              class="w-9 border-l border-slate-100 px-1 py-2 text-center font-medium dark:border-slate-800"
            >
              <VariantHead :label="v.label" :values="v.values" compact class="justify-center" />
            </th>
          </tr>
        </thead>
        <tbody>
          <template v-for="g in groups" :key="g.stepId">
            <tr
              v-for="(line, i) in g.items"
              :key="g.stepId + i"
              :class="[
                line.kind === 'refusal'
                  ? 'refusal-line'
                  : 'border-t border-slate-100 dark:border-slate-800',
                { 'opacity-70': dimmed(line) },
              ]"
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
              <template v-if="line.kind === 'claim'">
                <td class="max-w-[44ch] px-3 py-2.5 align-top">
                  <!-- The issue's title once tracked; the free text was the
                       draft it was written from (#138). -->
                  <template v-if="line.ref">
                    <a
                      :href="line.ref.url ?? '#'"
                      target="_blank"
                      rel="noopener"
                      class="text-indigo-700 dark:text-indigo-300"
                      ><span class="font-mono text-mono">#{{ line.ref.issueId }}</span>
                      {{ line.ref.title }}</a
                    >
                  </template>
                  <template v-else>
                    <span class="italic">{{ line.comment.body }}</span>
                  </template>
                  <span
                    v-if="line.first && line.comment.discardReason"
                    class="mt-1 block font-mono text-mono text-slate-500 dark:text-slate-400"
                    >discarded: {{ line.comment.discardReason }}</span
                  >
                </td>
                <td
                  v-for="v in grid.variants"
                  :key="v.id"
                  class="border-l border-slate-100 px-1 py-2.5 text-center align-middle font-mono dark:border-slate-800"
                >
                  <span
                    v-if="markOf(line, v.id) === '✓'"
                    role="img"
                    :aria-label="`accepted on ${v.label}`"
                    class="inline-flex h-[19px] w-[19px] items-center justify-center rounded-full border-[1.5px] text-label font-bold leading-none border-emerald-700 bg-emerald-50 text-emerald-700 dark:border-emerald-400 dark:bg-emerald-950 dark:text-emerald-400"
                    >✓</span
                  >
                  <span
                    v-else-if="markOf(line, v.id) === '·'"
                    role="img"
                    :aria-label="`waiting on ${v.label}`"
                    class="inline-flex h-[19px] w-[19px] rounded-full border-[1.5px] border-slate-300 dark:border-slate-600"
                  ></span>
                </td>
              </template>
              <template v-else>
                <td class="max-w-[44ch] px-3 pb-2.5 align-top">
                  <span class="font-mono text-mono text-amber-700 dark:text-amber-400">{{
                    line.remark
                  }}</span>
                </td>
                <td
                  v-for="v in grid.variants"
                  :key="v.id"
                  class="border-l border-slate-100 px-1 pb-2.5 text-center align-middle font-mono dark:border-slate-800"
                >
                  <span
                    v-if="markOf(line, v.id) === '✗'"
                    role="img"
                    :aria-label="`refused on ${v.label}`"
                    class="inline-flex h-[19px] w-[19px] items-center justify-center rounded-full border-[1.5px] text-label font-bold leading-none border-amber-700 bg-amber-50 text-amber-700 dark:border-amber-400 dark:bg-amber-950 dark:text-amber-400"
                    >✗</span
                  >
                </td>
              </template>
            </tr>
          </template>
        </tbody>
      </table>
    </div>
  </div>
</template>
