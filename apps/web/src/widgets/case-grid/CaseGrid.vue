<script setup lang="ts">
/**
 * The grid a case is judged from: steps down, variants across.
 *
 * Nothing here is edited. Each capture wears its own verdict — a coloured ring
 * and a mark — and the thumbnail is the way in: clicking one opens the carousel
 * on that exact capture.
 *
 * A capture sits on an inert ground: no shadow, no gradient, no tinted cell
 * behind the image. This interface frames someone else's product, and nothing
 * of ozalid's may be mistaken for part of it. The verdict rings the capture
 * rather than colouring what surrounds it.
 */
import { computed } from 'vue'
import type { components } from '@/shared/api'
import { StateIcon, VariantHead } from '@/shared/ui'
import type { Tone } from '@/shared/lib'

type Grid = components['schemas']['Grid']
type Cell = Grid['steps'][number]['cells'][number]

const props = defineProps<{ grid: Grid; openCell?: { stepId: string; variantId: string } | null }>()
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

/** The ring around a capture, and the mark under it. */
const RING: Record<string, string> = {
  validated: 'border-emerald-600 dark:border-emerald-500',
  'to-fix': 'border-amber-600 dark:border-amber-500',
  'to-review': 'border-slate-300 dark:border-slate-600',
}
const MARK: Record<string, string> = {
  validated: 'text-emerald-700 dark:text-emerald-400',
  'to-fix': 'text-amber-700 dark:text-amber-400',
  'to-review': 'text-slate-500 dark:text-slate-400',
}
const LABEL: Record<string, string> = {
  validated: 'validée',
  'to-fix': 'commentée',
  'to-review': 'à juger',
}
const TONE: Record<string, Tone> = {
  validated: 'done',
  'to-fix': 'dev',
  'to-review': 'reviewer',
}

/** A verdict, once given, is stamped straight onto the capture — no disc, no
 * plate behind it — and the capture steps back. Full intensity is what the eye
 * should be left with: the squares still to look at. */
function judged(cell: Cell) {
  return cell.status === 'validated' || cell.status === 'to-fix'
}

function isOpen(step: Grid['steps'][number], cell: Cell) {
  return props.openCell?.stepId === step.id && props.openCell?.variantId === cell.variantId
}

const hasRecordings = computed(() => props.grid.recordings.length > 0)
</script>

<template>
  <p
    v-if="grid.steps.length === 0"
    class="rounded-md border border-dashed border-slate-300 px-4 py-6 text-center font-mono text-[12px] text-slate-500 dark:border-slate-600 dark:text-slate-400"
  >
    aucune capture pour ce cas
  </p>

  <div v-else class="overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
    <table class="w-full border-separate border-spacing-0 text-[13.5px]">
      <thead>
        <tr
          class="bg-slate-50 font-mono text-[10.5px] text-slate-500 dark:bg-slate-800/60 dark:text-slate-400"
        >
          <th
            class="border-r border-b border-slate-200 px-3 py-2 text-left font-medium dark:border-slate-700"
          >
            étape
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
            class="border-r border-b border-slate-200 px-3 py-2.5 text-left align-middle text-[12.5px] font-semibold dark:border-slate-700"
          >
            enregistrement
            <i
              class="mt-0.5 block font-mono text-[10px] font-normal text-slate-500 not-italic dark:text-slate-400"
            >
              le flux complet
            </i>
          </th>
          <td
            v-for="v in variants"
            :key="v.id"
            class="border-r border-b border-slate-200 p-2.5 text-center align-middle last:border-r-0 dark:border-slate-700"
          >
            <!-- No verdict ring here, ever: a recording is not comparable, so
                 nothing about it can be judged (ADR 0013). Giving it a colour
                 would invent a state the server does not compute. -->
            <a
              v-if="recordingOf(v.id)"
              :href="`/api/blobs/${recordingOf(v.id)!.hash}`"
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
            class="border-r border-b border-slate-200 px-3 py-2.5 text-left align-middle text-[12.5px] font-semibold last:border-b-0 dark:border-slate-700"
          >
            {{ step.name }}
            <i
              class="mt-0.5 block font-mono text-[10px] font-normal text-slate-500 not-italic dark:text-slate-400"
            >
              étape {{ step.position + 1 }}
            </i>
          </th>
          <td
            v-for="v in variants"
            :key="v.id"
            class="border-r border-b border-slate-200 p-2.5 text-center align-middle last:border-r-0 dark:border-slate-700"
          >
            <template v-if="cellOf(step, v.id)">
              <span class="relative inline-block leading-none">
                <button
                  type="button"
                  class="block cursor-zoom-in border-2 p-0 hover:border-indigo-500 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500"
                  :class="[
                    isPortrait(v.values) ? SIZE.tall : SIZE.wide,
                    RING[cellOf(step, v.id)!.status],
                    isOpen(step, cellOf(step, v.id)!)
                      ? 'ring-2 ring-indigo-500 ring-offset-2 dark:ring-offset-slate-900'
                      : '',
                  ]"
                  :aria-label="`ouvrir ${step.name} — ${v.label} dans le carrousel`"
                  @click="emit('open', step.id, v.id)"
                >
                  <img
                    :src="`/api/blobs/${cellOf(step, v.id)!.hash}`"
                    :alt="`${step.name} — ${v.label}`"
                    loading="lazy"
                    class="block h-full w-full bg-slate-100 object-cover dark:bg-slate-900"
                    :class="judged(cellOf(step, v.id)!) ? 'opacity-40' : ''"
                  />
                </button>
                <span
                  v-if="judged(cellOf(step, v.id)!)"
                  class="pointer-events-none absolute inset-0 grid place-items-center"
                  :class="MARK[cellOf(step, v.id)!.status]"
                >
                  <StateIcon
                    :tone="TONE[cellOf(step, v.id)!.status]"
                    :size="26"
                    :label="LABEL[cellOf(step, v.id)!.status]"
                  />
                </span>
              </span>
              <span
                class="mt-1.5 flex items-center justify-center gap-1.5 font-mono text-[10px]"
                :class="MARK[cellOf(step, v.id)!.status]"
              >
                <StateIcon
                  :tone="TONE[cellOf(step, v.id)!.status]"
                  :size="12"
                  :label="LABEL[cellOf(step, v.id)!.status]"
                />
                {{ LABEL[cellOf(step, v.id)!.status] }}
              </span>
            </template>

            <!-- A hole: every other variant of this step was captured, this one
                 was not. That is a failed run, not a deliberate absence. -->
            <template v-else>
              <span
                class="inline-grid place-items-center border-2 border-dashed border-red-600 bg-red-50 text-red-700 dark:border-red-500 dark:bg-red-950/50 dark:text-red-400"
                :class="isPortrait(v.values) ? SIZE.tall : SIZE.wide"
              >
                <svg
                  width="18"
                  height="18"
                  viewBox="0 0 16 16"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.4"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  role="img"
                  aria-label="capture manquante"
                >
                  <title>capture manquante</title>
                  <path d="M8 2.2L14.5 13.4H1.5z" />
                  <path d="M8 6.4v3M8 11.4v.1" />
                </svg>
              </span>
              <span
                class="mt-1.5 block font-mono text-[10px] text-red-700 dark:text-red-400"
                >manquante</span
              >
            </template>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
