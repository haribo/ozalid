<script setup lang="ts">
/**
 * The grid a case is judged from: steps down, variants across.
 *
 * It answers one question — what have I validated, and what have I not? — and
 * carries no other detail (frontend ADR 0003). A square that has been judged and
 * has not moved steps back and wears its mark; a square that still needs eyes
 * stays bare and at full strength. Nothing is edited here: the thumbnail is the
 * way in, and clicking one opens the carousel on that exact capture.
 *
 * A capture sits on an inert ground: no shadow, no gradient, no tinted cell
 * behind the image. This interface frames someone else's product, and nothing
 * of ozalid's may be mistaken for part of it.
 */
import { computed } from 'vue'
import type { components } from '@/shared/api'
import { EmptyState, AppButton, MissingIcon, MovedIcon, StateIcon, VariantHead } from '@/shared/ui'
import { hasMoved, type Tone } from '@/shared/lib'

type Grid = components['schemas']['Grid']
type Cell = Grid['steps'][number]['cells'][number]

const props = defineProps<{
  slug: string
  grid: Grid
  openCell?: { stepId: string; variantId: string } | null
}>()
const emit = defineEmits<{ open: [stepId: string, variantId: string] }>()

const variants = computed(() => props.grid.variants)

/** A step missing a variant its siblings carry is a hole, not a blank: the run
 * failed there, and drawing it neutrally would hide that (ADR 0016). */
function cellOf(step: Grid['steps'][number], variantId: string) {
  return step.cells.find((c) => c.variantId === variantId)
}

function recordingOf(variantId: string) {
  return props.grid.recordings.find((r) => r.variantId === variantId)
}

/** Tall on a portrait variant, so the thumbnail keeps the shape it was taken in. */
function isPortrait(values: Record<string, string>) {
  return Object.values(values).includes('mobile')
}

const SIZE = { wide: 'h-[72px] w-[116px]', tall: 'h-[94px] w-[60px]' }

const NEUTRAL = 'border-slate-300 dark:border-slate-600'
const RING: Record<string, string> = {
  validated: 'border-emerald-600 dark:border-emerald-500',
  'to-fix': 'border-amber-600 dark:border-amber-500',
  'to-review': NEUTRAL,
}
const INK: Record<string, string> = {
  validated: 'text-emerald-700 dark:text-emerald-400',
  'to-fix': 'text-amber-700 dark:text-amber-400',
}
const TONE: Record<string, Tone> = { validated: 'done', 'to-fix': 'dev' }
const LABEL: Record<string, string> = { validated: 'validated', 'to-fix': 'commented' }

/**
 * The six readings a cell can have, and no others.
 *
 * A capture that has moved is one of them: for the only question the grid asks,
 * it has not been validated — not the bytes on display. It renders as a square
 * to judge, and the mark saying why it came back takes the place of the verdict
 * badge it used to wear. Validated-and-moved and commented-and-moved are one
 * cell: what separated them is exactly what the grid no longer reports
 * (frontend ADR 0003).
 */
function reading(cell: Cell): 'moved' | 'judged' | 'pending' {
  if (hasMoved(cell.freshness)) return 'moved'
  if (cell.status === 'validated' || cell.status === 'to-fix') return 'judged'
  return 'pending'
}

/** Judged and settled: it steps back, because full intensity is reserved for
 * what still needs eyes. That is what makes the answer readable at a glance
 * rather than by counting. */
function settled(cell: Cell) {
  return reading(cell) === 'judged'
}

function isOpen(step: Grid['steps'][number], cell: Cell) {
  return props.openCell?.stepId === step.id && props.openCell?.variantId === cell.variantId
}

const hasRecordings = computed(() => props.grid.recordings.length > 0)
</script>

<template>
  <EmptyState v-if="grid.steps.length === 0"> no capture on this case </EmptyState>

  <template v-else>
    <div class="overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
      <table class="w-full border-separate border-spacing-0 text-body">
        <thead>
          <tr
            class="bg-slate-50 font-mono text-mono text-slate-500 dark:bg-slate-800/60 dark:text-slate-400"
          >
            <th
              class="border-r border-b border-slate-200 px-3 py-2 text-left font-medium dark:border-slate-700"
            >
              step
            </th>
            <th
              v-for="v in variants"
              :key="v.id"
              class="border-r border-b border-slate-200 px-3 py-2 text-center font-medium whitespace-nowrap last:border-r-0 dark:border-slate-700"
            >
              <VariantHead :label="v.label" :values="v.values" class="justify-center" />
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="hasRecordings" class="bg-slate-50/60 dark:bg-slate-800/30">
            <th
              class="border-r border-b border-slate-200 px-3 py-2.5 text-left align-middle text-body font-semibold dark:border-slate-700"
            >
              recording
              <i
                class="mt-0.5 block font-mono text-mono font-normal text-slate-500 not-italic dark:text-slate-400"
              >
                the whole flow
              </i>
            </th>
            <td
              v-for="v in variants"
              :key="v.id"
              class="border-r border-b border-slate-200 p-2.5 text-center align-middle last:border-r-0 dark:border-slate-700"
            >
              <!-- No verdict ring, ever: a recording is not comparable, so
                   nothing about it can be judged (ADR 0013). Giving it a colour
                   would invent a state the server does not compute. -->
              <a
                v-if="recordingOf(v.id)"
                :href="`/api/projects/${slug}/recordings/${recordingOf(v.id)!.id}`"
                class="inline-grid place-items-center border-2 border-slate-200 bg-slate-100 text-slate-500 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-400"
                :class="isPortrait(v.values) ? SIZE.tall : SIZE.wide"
              >
                <!-- A plain triangle, not the emoji: a coloured glyph next to a
                     capture is exactly the decoration this interface must not
                     put there. -->
                <svg width="16" height="16" viewBox="0 0 16 16" aria-hidden="true">
                  <path d="M5 3.5v9l7-4.5z" fill="currentColor" />
                </svg>
              </a>
            </td>
          </tr>

          <tr v-for="step in grid.steps" :key="step.id">
            <th
              class="border-r border-b border-slate-200 px-3 py-2.5 text-left align-middle text-body font-semibold last:border-b-0 dark:border-slate-700"
            >
              {{ step.name }}
              <i
                class="mt-0.5 block font-mono text-mono font-normal text-slate-500 not-italic dark:text-slate-400"
              >
                step {{ step.position + 1 }}
              </i>
            </th>
            <td
              v-for="v in variants"
              :key="v.id"
              class="border-r border-b border-slate-200 p-2.5 text-center align-middle last:border-r-0 dark:border-slate-700"
            >
              <template v-if="cellOf(step, v.id)">
                <span class="relative inline-block leading-none">
                  <AppButton
                    :class="[
                      isPortrait(v.values) ? SIZE.tall : SIZE.wide,
                      reading(cellOf(step, v.id)!) === 'judged'
                        ? RING[cellOf(step, v.id)!.status]
                        : NEUTRAL,
                      isOpen(step, cellOf(step, v.id)!)
                        ? 'ring-2 ring-indigo-500 ring-offset-2 dark:ring-offset-slate-900'
                        : '',
                    ]"
                    :aria-label="`open ${step.name} — ${v.label} in the carousel`"
                    variant="secondary"
                    @click="emit('open', step.id, v.id)"
                  >
                    <img
                      :src="`/api/projects/${slug}/captures/${cellOf(step, v.id)!.id}`"
                      :alt="`${step.name} — ${v.label}`"
                      loading="lazy"
                      class="block h-full w-full bg-slate-100 object-cover dark:bg-slate-900"
                      :class="settled(cellOf(step, v.id)!) ? 'opacity-40' : ''"
                    />
                  </AppButton>
                  <span
                    v-if="settled(cellOf(step, v.id)!)"
                    class="pointer-events-none absolute inset-0 grid place-items-center"
                    :class="INK[cellOf(step, v.id)!.status]"
                  >
                    <StateIcon
                      :tone="TONE[cellOf(step, v.id)!.status]"
                      :size="18"
                      :label="LABEL[cellOf(step, v.id)!.status]"
                      :class="
                        cellOf(step, v.id)!.status === 'validated'
                          ? 'bg-emerald-50 dark:bg-emerald-950'
                          : 'bg-amber-50 dark:bg-amber-950'
                      "
                    />
                  </span>
                  <span
                    v-else-if="reading(cellOf(step, v.id)!) === 'moved'"
                    class="pointer-events-none absolute inset-0 grid place-items-center text-indigo-600 dark:text-indigo-300"
                  >
                    <MovedIcon :size="18" label="moved" class="bg-indigo-50 dark:bg-indigo-950" />
                  </span>
                </span>
              </template>

              <!-- A hole: every other variant of this step was captured, this
                   one was not. That is a failed run, not a deliberate
                   absence. -->
              <template v-else>
                <!-- eslint-disable vue/no-restricted-class -- the missing mark's own dashes (ADR 0016), not an empty state -->
                <span
                  class="inline-grid place-items-center border-2 border-dashed border-red-600 bg-red-50 text-red-700 dark:border-red-500 dark:bg-red-950/50 dark:text-red-400"
                  :class="isPortrait(v.values) ? SIZE.tall : SIZE.wide"
                >
                  <MissingIcon :size="18" />
                </span>
                <!-- eslint-enable vue/no-restricted-class -->
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- The words are gone from under the thumbnails, so the language is learnt
         here instead — once, rather than translated under every cell. -->
    <div
      class="mt-2.5 flex flex-wrap justify-center gap-x-5 gap-y-1.5 font-mono text-mono text-slate-500 dark:text-slate-400"
    >
      <span class="flex items-center gap-1.5">
        <StateIcon tone="reviewer" :size="12" label="to review" />to review
      </span>
      <span class="flex items-center gap-1.5 text-emerald-700 dark:text-emerald-400">
        <StateIcon tone="done" :size="12" label="validated" />validated
      </span>
      <span class="flex items-center gap-1.5 text-amber-700 dark:text-amber-400">
        <StateIcon tone="dev" :size="12" label="commented" />commented
      </span>
      <span class="flex items-center gap-1.5 text-indigo-600 dark:text-indigo-300">
        <MovedIcon :size="12" label="moved" />moved
      </span>
      <span class="flex items-center gap-1.5 text-red-700 dark:text-red-400">
        <MissingIcon :size="12" />missing
      </span>
    </div>
  </template>
</template>
