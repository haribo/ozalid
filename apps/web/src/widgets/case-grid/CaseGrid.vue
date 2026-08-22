<script setup lang="ts">
/**
 * The grid a case is judged from: steps down, variants across.
 *
 * A capture sits on an inert ground — no shadow, no gradient, no rounded
 * corner biting into the image. This interface frames someone else's product,
 * and nothing of ozalid's may be mistaken for part of it.
 */
import { computed } from 'vue'
import type { components } from '@/shared/api'
import VariantHead from './VariantHead.vue'

type Grid = components['schemas']['Grid']

const props = defineProps<{ grid: Grid }>()

const variants = computed(() => props.grid.variants)

/** A cell with no capture is absent, not blank: not every variant exists at
 * every step, and drawing a placeholder would suggest something is missing. */
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
            class="border-b border-r border-slate-200 px-3 py-2 text-left font-medium dark:border-slate-700"
          >
            étape
          </th>
          <th
            v-for="v in variants"
            :key="v.id"
            class="border-b border-r border-slate-200 px-3 py-2 text-left font-medium whitespace-nowrap last:border-r-0 dark:border-slate-700"
          >
            <VariantHead :label="v.label" :values="v.values" />
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="hasRecordings" class="bg-slate-50/60 dark:bg-slate-800/30">
          <th
            class="border-b border-r border-slate-200 px-3 py-2.5 text-left align-top text-[12.5px] font-semibold dark:border-slate-700"
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
            class="border-b border-r border-slate-200 p-2 align-top last:border-r-0 dark:border-slate-700"
          >
            <a
              v-if="recordingOf(v.id)"
              :href="`/api/blobs/${recordingOf(v.id)!.hash}`"
              class="grid place-items-center border border-slate-300 bg-slate-100 text-slate-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-400"
              :class="isPortrait(v.values) ? 'h-[94px] w-[60px]' : 'h-[72px] w-[116px]'"
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
            class="border-b border-r border-slate-200 px-3 py-2.5 text-left align-top text-[12.5px] font-semibold last:border-b-0 dark:border-slate-700"
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
            class="border-b border-r border-slate-200 p-2 align-top last:border-r-0 dark:border-slate-700"
            :class="!cellOf(step, v.id) ? 'bg-slate-50 dark:bg-slate-800/40' : ''"
          >
            <img
              v-if="cellOf(step, v.id)"
              :src="`/api/blobs/${cellOf(step, v.id)!.hash}`"
              :alt="`${step.name} — ${v.label}`"
              loading="lazy"
              class="block border border-slate-300 bg-slate-100 object-cover dark:border-slate-600 dark:bg-slate-900"
              :class="isPortrait(v.values) ? 'h-[94px] w-[60px]' : 'h-[72px] w-[116px]'"
            />
            <span
              v-else
              class="block py-6 text-center font-mono text-[10px] text-slate-400 dark:text-slate-500"
              >pas de capture</span
            >
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
