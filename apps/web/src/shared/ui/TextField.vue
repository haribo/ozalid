<script setup lang="ts">
/**
 * The one way a text field is made (#155, frontend ADR 0006).
 *
 * Two shapes, one contract. The standard shape: a mono uppercase label above
 * the box — the forms everywhere. The floating shape: the label starts inside
 * the field, icon and all, and lands on the border once something is typed,
 * so naming the field never costs a line of height (#129, the front door).
 *
 * `invalid` reddens the border — the 409 whose sentence the caller shows
 * beside (#116). The label stays a real <label> wherever it sits, so
 * getByLabel keeps resolving.
 */
import { computed, ref, useId } from 'vue'

withDefaults(
  defineProps<{
    label: string
    type?: string
    required?: boolean
    autocomplete?: string
    placeholder?: string
    floating?: boolean
    invalid?: boolean
  }>(),
  {
    type: 'text',
    required: false,
    autocomplete: undefined,
    placeholder: undefined,
    floating: false,
    invalid: false,
  },
)

const model = defineModel<string>({ default: '' })

/** One set of metrics for the landed label AND the legend that mirrors it:
 * the notch is exactly as wide as the words, or it lies. */
const LANDED_TYPE = 'font-mono text-[10px] tracking-wider uppercase'

const id = useId()
const focused = ref(false)
const landed = computed(() => focused.value || model.value.length > 0)
</script>

<template>
  <div v-if="floating" class="relative">
    <input
      :id="id"
      v-model="model"
      :type="type"
      :required="required"
      :autocomplete="autocomplete"
      class="h-[42px] w-full rounded border-[1.5px] border-transparent bg-slate-100 px-3 text-body text-slate-900 outline-none dark:bg-slate-800 dark:text-slate-100"
      @focus="focused = true"
      @blur="focused = false"
    />
    <!-- The border lives on a decorative fieldset so its legend can cut a
         real gap under the landed label: the label needs no plate, and no
         guess about the ground behind it survives to be wrong (#173). The
         browser draws a fieldset's top border at the middle of its legend,
         so the whole thing is shifted up by half the legend's 12px height —
         the visible line lands exactly on the input's top edge (the trick is
         tribnest's FieldFrame). The legend mirrors the label's content,
         transparent, only to size the notch. -->
    <fieldset
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 -top-1.5 bottom-0 m-0 min-w-0 rounded border-[1.5px] p-0"
      :class="
        invalid
          ? 'border-red-600 dark:border-red-500'
          : focused
            ? 'border-indigo-500 dark:border-indigo-400'
            : 'border-slate-300 dark:border-slate-600'
      "
    >
      <legend
        class="ml-2 block h-3 overflow-hidden whitespace-nowrap transition-all"
        :class="landed ? 'max-w-full' : 'max-w-[0.01px]'"
      >
        <span class="flex items-center gap-1.5 px-1.5 opacity-0" :class="LANDED_TYPE">
          <slot name="icon" :landed="true" />
          {{ label }}
        </span>
      </legend>
    </fieldset>
    <label
      :for="id"
      class="pointer-events-none absolute flex items-center transition-all select-none"
      :class="[
        focused ? 'text-indigo-700 dark:text-indigo-300' : 'text-slate-500 dark:text-slate-400',
        landed
          ? `top-0 left-2 -translate-y-1/2 gap-1.5 px-1.5 ${LANDED_TYPE}`
          : 'top-1/2 left-3 -translate-y-1/2 gap-2 text-body',
      ]"
    >
      <slot name="icon" :landed="landed" />
      {{ label }}
    </label>
  </div>

  <label v-else class="flex flex-col gap-1.5">
    <span class="font-mono text-label tracking-wider text-slate-500 uppercase">{{ label }}</span>
    <input
      v-model="model"
      :type="type"
      :required="required"
      :autocomplete="autocomplete"
      :placeholder="placeholder"
      class="rounded border bg-white px-2.5 py-1.5 text-body dark:bg-slate-900"
      :class="
        invalid ? 'border-red-600 dark:border-red-500' : 'border-slate-300 dark:border-slate-600'
      "
    />
  </label>
</template>
