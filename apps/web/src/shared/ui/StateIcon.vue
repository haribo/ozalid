<script setup lang="ts">
/**
 * A status: a glyph on a disc.
 *
 * The disc is the rule, and it holds everywhere a state is shown — the grid, its
 * legend, the state pills, the category gauges, the recap. It is what separates
 * ozalid's marks from the product it frames: a capture's colours belong to
 * somebody else, and a bare glyph laid over them can be taken for one of theirs.
 *
 * An action is not a status and wears no disc. `ActionIcon` carries those.
 *
 * An empty disc is a status too — nothing has been said about this yet — which
 * is why `reviewer` has no glyph and `idle` only dashes its edge.
 */
import { computed } from 'vue'
import { TONE_LABELS, type Tone } from '@/shared/lib'

const props = withDefaults(defineProps<{ tone: Tone; size?: number; label?: string }>(), {
  size: 13,
  label: '',
})

const title = computed(() => props.label || TONE_LABELS[props.tone])

/** The disc is drawn around the glyph, so its box is larger than the glyph's. */
const box = computed(() => Math.round(props.size * 1.55))
</script>

<template>
  <span
    class="grid shrink-0 place-items-center rounded-full border-[1.5px] border-current"
    :class="tone === 'idle' ? 'border-dashed' : ''"
    :style="{ width: `${box}px`, height: `${box}px` }"
    role="img"
    :aria-label="title"
  >
    <svg
      :width="size"
      :height="size"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      stroke-width="1.6"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <title>{{ title }}</title>
      <path v-if="tone === 'done'" d="M4.6 8.3l2.3 2.3 4.5-5" />
      <path v-else-if="tone === 'dev'" d="M3 4.4h10v6.2H7.6L4.9 12.6v-2H3z" />
    </svg>
  </span>
</template>
