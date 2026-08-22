<script setup lang="ts">
/**
 * A variant's column header. On eight or twelve columns the shape reads faster
 * than the word, so the icons come first — but the label stays, because an
 * icon nobody has learnt is decoration.
 *
 * The axes are declared by the project (ADR 0001), so a value is only drawn as
 * an icon when the interface recognises it. Anything else falls back to text,
 * which is what makes an unknown axis harmless.
 */
import { computed } from 'vue'

const props = defineProps<{ label: string; values: Record<string, string> }>()

const KNOWN = ['desktop', 'mobile', 'light', 'dark'] as const
type Known = (typeof KNOWN)[number]

const icons = computed(() =>
  Object.values(props.values).filter((v): v is Known => (KNOWN as readonly string[]).includes(v)),
)
</script>

<template>
  <span class="inline-flex items-center gap-1.5" :title="label">
    <svg
      v-for="name in icons"
      :key="name"
      :width="name === 'desktop' ? 15 : 13"
      :height="name === 'desktop' ? 15 : 13"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      stroke-width="1.3"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
      class="shrink-0"
    >
      <template v-if="name === 'desktop'">
        <rect x="1.5" y="2.5" width="13" height="9" rx="1" />
        <path d="M5.5 14h5" />
      </template>
      <template v-else-if="name === 'mobile'">
        <rect x="4.5" y="1.5" width="7" height="13" rx="1.2" />
        <path d="M7 12.6h2" />
      </template>
      <template v-else-if="name === 'light'">
        <circle cx="8" cy="8" r="3.2" />
        <path
          d="M8 1v1.6M8 13.4V15M1 8h1.6M13.4 8H15M3.2 3.2l1.1 1.1M11.7 11.7l1.1 1.1M12.8 3.2l-1.1 1.1M4.3 11.7l-1.1 1.1"
        />
      </template>
      <template v-else>
        <path d="M13.5 9.6A5.8 5.8 0 0 1 6.4 2.5a5.8 5.8 0 1 0 7.1 7.1z" />
      </template>
    </svg>
    {{ label }}
  </span>
</template>
