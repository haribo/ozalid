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
      class="h-[42px] w-full rounded border-[1.5px] bg-slate-100 px-3 text-body text-slate-900 outline-none dark:bg-slate-800 dark:text-slate-100"
      :class="
        invalid
          ? 'border-red-600 dark:border-red-500'
          : focused
            ? 'border-indigo-500 dark:border-indigo-400'
            : 'border-slate-300 dark:border-slate-600'
      "
      @focus="focused = true"
      @blur="focused = false"
    />
    <label
      :for="id"
      class="pointer-events-none absolute flex items-center transition-all select-none"
      :class="[
        focused ? 'text-indigo-700 dark:text-indigo-300' : 'text-slate-500 dark:text-slate-400',
        landed
          ? '-top-2 left-2 gap-1.5 rounded bg-white px-1.5 font-mono text-[10px] tracking-wider uppercase dark:bg-slate-950'
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
