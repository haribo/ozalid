<script setup lang="ts">
/**
 * The one way a button is made (#155, after tribnest's ADR-0008).
 *
 * Variants are action contracts — primary, secondary, destructive, ghost —
 * never content. `success` is the transient save-feedback state: inert, and
 * it keeps the width it had at the switch so nothing in the row moves. An
 * icon-only button squares itself and must be given a `label`, since the icon
 * alone says nothing to a screen reader.
 *
 * A raw `<button>` outside shared/ui is a lint error, which is what keeps the
 * cursor, the disabled look and the focus ring from drifting ever again.
 */
import { ref, watch } from 'vue'

type Variant = 'primary' | 'secondary' | 'destructive' | 'ghost'
type Size = 'sm' | 'md' | 'lg'

const props = withDefaults(
  defineProps<{
    variant?: Variant
    size?: Size
    type?: 'button' | 'submit'
    disabled?: boolean
    /** Inert success feedback; the caller decides when the work is done. */
    success?: boolean
    /** Square icon-only button. `label` becomes mandatory in spirit. */
    icon?: boolean
    /** What a screen reader hears when the slot is only a glyph. */
    label?: string
  }>(),
  {
    variant: 'primary',
    size: 'md',
    type: 'button',
    disabled: false,
    success: false,
    icon: false,
    label: undefined,
  },
)

const emit = defineEmits<{ click: [] }>()

const el = ref<HTMLButtonElement | null>(null)
const heldWidth = ref<string | undefined>(undefined)

// Success keeps the width the button had at the switch: the label changes,
// the row does not move.
watch(
  () => props.success,
  (now) => {
    heldWidth.value = now && el.value ? `${el.value.offsetWidth}px` : undefined
  },
)

const VARIANT: Record<Variant, string> = {
  primary:
    'border-indigo-600 bg-indigo-600 text-white dark:border-indigo-500 dark:bg-indigo-500 dark:text-slate-950',
  secondary: 'border-indigo-600 text-indigo-700 dark:border-indigo-500 dark:text-indigo-300',
  destructive: 'border-amber-600 text-amber-700 dark:border-amber-500 dark:text-amber-400',
  ghost: 'border-transparent text-slate-500 underline underline-offset-3 dark:text-slate-400',
}

const SIZE: Record<Size, string> = {
  sm: 'px-2.5 py-1 text-label',
  md: 'px-3 py-1.5 text-body',
  lg: 'px-4 py-2 text-body',
}

const ICON_SIZE: Record<Size, string> = {
  sm: 'h-6 w-6',
  md: 'h-7 w-7',
  lg: 'h-9 w-9',
}

function onClick() {
  if (!props.disabled && !props.success) emit('click')
}
</script>

<template>
  <button
    ref="el"
    :type="type"
    :disabled="disabled"
    :aria-label="label"
    :style="heldWidth ? { minWidth: heldWidth } : undefined"
    class="inline-flex items-center justify-center gap-1.5 rounded border font-medium focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-indigo-500"
    :class="[
      success
        ? 'cursor-default border-emerald-600 text-emerald-700 dark:border-emerald-500 dark:text-emerald-400'
        : VARIANT[variant],
      icon ? ICON_SIZE[size] : SIZE[size],
      disabled ? 'cursor-not-allowed opacity-50' : success ? '' : 'cursor-pointer',
    ]"
    @click="onClick"
  >
    <slot />
  </button>
</template>
