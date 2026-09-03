<script setup lang="ts">
/**
 * Where judging happens.
 *
 * One capture at full size, the two verdicts within reach, and the keyboard
 * doing the work: arrows to move, space to validate, Escape to leave. On a
 * case of twelve squares the reviewer never touches the mouse.
 */
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import type { components } from '@/shared/api'
import { ActionIcon, KindIcon, MovedIcon, StateIcon } from '@/shared/ui'
import { hasMoved, type Tone } from '@/shared/lib'

type Grid = components['schemas']['Grid']
type Comment = components['schemas']['Comment']

const props = defineProps<{
  slug: string
  grid: Grid
  comments: Comment[]
  stepId: string
  variantId: string
  busy?: boolean
}>()

const emit = defineEmits<{
  close: []
  move: [stepId: string, variantId: string]
  validate: [stepId: string, variantId: string]
  comment: [
    input: { stepId: string; kind: 'defect' | 'improvement'; body: string; variantIds: string[] },
  ]
  judge: [commentId: string, accept: boolean, remark: string]
}>()

/** Every square that exists, in reading order: steps down, variants across. */
const squares = computed(() =>
  props.grid.steps.flatMap((step) =>
    props.grid.variants
      .filter((v) => step.cells.some((c) => c.variantId === v.id))
      .map((v) => ({ stepId: step.id, variantId: v.id })),
  ),
)

const index = computed(() =>
  squares.value.findIndex((s) => s.stepId === props.stepId && s.variantId === props.variantId),
)

const step = computed(() => props.grid.steps.find((s) => s.id === props.stepId))
const variant = computed(() => props.grid.variants.find((v) => v.id === props.variantId))
const cell = computed(() => step.value?.cells.find((c) => c.variantId === props.variantId))

/** The comments covering this exact square — what is already known about it. */
const onSquare = computed(() =>
  props.comments.filter((c) => c.stepId === props.stepId && c.variantIds.includes(props.variantId)),
)

/** A delivery waiting for a verdict. That is what gets judged here, not the
 * capture itself. */
const toJudge = computed(() => onSquare.value.find((c) => c.state === 'to-review'))

/** The verdict already on this square, in the grid's own vocabulary — one
 * language learnt once, whatever the size the capture is shown at. */
const VERDICT_TONE: Record<string, Tone> = { validated: 'done', 'to-fix': 'dev' }
const VERDICT_MARK: Record<string, string> = {
  validated: 'text-emerald-700 dark:text-emerald-400',
  'to-fix': 'text-amber-700 dark:text-amber-400',
}
const VERDICT_LABEL: Record<string, string> = { validated: 'validated', 'to-fix': 'commented' }

/** A capture that has moved is back to needing eyes, whatever its verdict says,
 * so it comes back to full strength here as it does in the grid. */
const moved = computed(() => hasMoved(cell.value?.freshness))
const judged = computed(
  () => cell.value !== undefined && cell.value.status in VERDICT_TONE && !moved.value,
)

function go(delta: number) {
  const next = squares.value[index.value + delta]
  if (next) emit('move', next.stepId, next.variantId)
}

/** Up and down move to the same step's next variant; left and right walk the
 * whole grid in reading order. */
function goVariant(delta: number) {
  const here = props.grid.variants.findIndex((v) => v.id === props.variantId)
  const next = props.grid.variants[here + delta]
  if (next && step.value?.cells.some((c) => c.variantId === next.id)) {
    emit('move', props.stepId, next.id)
  }
}

// ---- comment composer -----------------------------------------------------

const composing = ref(false)
const kind = ref<'defect' | 'improvement'>('defect')
const body = ref('')
const chosen = ref<string[]>([])

/** The variant on screen is ticked to start with — it is the one being looked
 * at. */
watch(
  () => props.variantId,
  (id) => {
    if (!composing.value) chosen.value = [id]
  },
  { immediate: true },
)

function openComposer() {
  composing.value = true
  chosen.value = [props.variantId]
}

function toggle(id: string) {
  chosen.value = chosen.value.includes(id)
    ? chosen.value.filter((v) => v !== id)
    : [...chosen.value, id]
}

/**
 * Group shortcuts, so saying "everywhere" or "both dark ones" is one click and
 * not four.
 *
 * They are derived from the axis values actually present, never hard-coded:
 * the project declares its own axes and ozalid ships no list of them
 * (ADR 0001). A project with a `role` axis gets `admin` and `member`
 * shortcuts without anyone writing them.
 */
const groups = computed(() => {
  const values = new Set<string>()
  for (const v of props.grid.variants) {
    for (const value of Object.values(v.values)) values.add(value)
  }
  return [...values].toSorted()
})

function pickValue(value: string) {
  chosen.value = props.grid.variants
    .filter((v) => Object.values(v.values).includes(value))
    .map((v) => v.id)
}

function pickAll() {
  chosen.value = props.grid.variants.map((v) => v.id)
}

const canAdd = computed(() => body.value.trim() !== '' && chosen.value.length > 0)

function add() {
  if (!canAdd.value) return
  emit('comment', {
    stepId: props.stepId,
    kind: kind.value,
    body: body.value.trim(),
    variantIds: [...chosen.value],
  })
  body.value = ''
  composing.value = false
}

// ---- judging a delivery ---------------------------------------------------

const refusing = ref(false)
const remark = ref('')

function accept() {
  if (toJudge.value) emit('judge', toJudge.value.id, true, '')
}

function refuse() {
  if (!refusing.value) {
    refusing.value = true
    return
  }
  if (toJudge.value && remark.value.trim() !== '') {
    emit('judge', toJudge.value.id, false, remark.value.trim())
    remark.value = ''
    refusing.value = false
  }
}

// ---- keyboard -------------------------------------------------------------

function onKey(event: KeyboardEvent) {
  // Never steal a key from someone writing.
  const target = event.target as HTMLElement | null
  if (target && (target.tagName === 'TEXTAREA' || target.tagName === 'INPUT')) {
    if (event.key === 'Escape') (target as HTMLElement).blur()
    return
  }

  switch (event.key) {
    case 'Escape':
      emit('close')
      break
    case 'ArrowRight':
      go(1)
      break
    case 'ArrowLeft':
      go(-1)
      break
    case 'ArrowDown':
      goVariant(1)
      break
    case 'ArrowUp':
      goVariant(-1)
      break
    case ' ':
      event.preventDefault()
      if (!toJudge.value && cell.value?.status !== 'validated') {
        emit('validate', props.stepId, props.variantId)
      }
      break
    default:
      return
  }
}

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <!-- A column filling whatever box the caller gives it — the window, since
       #125. The stage takes what the bars leave. -->
  <div
    role="dialog"
    aria-label="capture"
    class="flex flex-col overflow-hidden bg-white dark:bg-slate-950"
  >
    <div
      class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 bg-slate-50 px-3 py-2 font-mono text-[11px] text-slate-500 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-400"
    >
      <span>
        <b class="font-medium text-slate-900 dark:text-slate-100">{{ step?.name }}</b>
        · {{ variant?.label }}
      </span>
      <span>
        {{ index + 1 }} / {{ squares.length }} ·
        <!-- Arrow glyphs are missing from most monospace faces and render as
             empty boxes; the system font has them. -->
        <kbd class="rounded border border-current px-1 font-sans">←</kbd>
        <kbd class="rounded border border-current px-1 font-sans">→</kbd> capture ·
        <kbd class="rounded border border-current px-1 font-sans">↑</kbd>
        <kbd class="rounded border border-current px-1 font-sans">↓</kbd> variant ·
        <kbd class="rounded border border-current px-1">Esc</kbd> close
      </span>
    </div>

    <div class="grid min-h-0 flex-1 place-items-center bg-slate-100 p-6 dark:bg-slate-950">
      <!-- The same reading as the grid, at full size: a square already judged
           steps back and wears its verdict, so what the eye lands on is what
           still needs looking at. -->
      <span v-if="cell" class="relative inline-block max-h-full max-w-full leading-none">
        <!-- Said on the image itself: the reviewer landed here from a keyboard
             walk and never saw the grid's mark. -->
        <span
          v-if="moved"
          class="absolute -top-3 -right-3 z-10 flex items-center gap-1.5 rounded border border-indigo-500 bg-white px-2 py-1 font-mono text-[10.5px] text-indigo-700 dark:border-indigo-400 dark:bg-slate-900 dark:text-indigo-300"
        >
          <MovedIcon :size="12" />moved<template v-if="cell.movedPixels !== undefined">
            · {{ cell.movedPixels }} px</template
          >
        </span>
        <!-- The capture takes the space the window offers. Its width was once
             written in advance (240/560 px), which showed a 1280 px capture at
             44% on the one screen where pixels are judged (#125). -->
        <img
          :src="`/api/projects/${slug}/captures/${cell.id}`"
          :alt="`${step?.name} — ${variant?.label}`"
          class="max-h-full max-w-full border border-slate-300 bg-white object-contain dark:border-slate-600 dark:bg-slate-900"
          :class="judged ? 'opacity-40' : ''"
        />
        <span
          v-if="judged"
          class="pointer-events-none absolute inset-0 grid place-items-center"
          :class="VERDICT_MARK[cell.status]"
        >
          <StateIcon
            :tone="VERDICT_TONE[cell.status]"
            :size="72"
            :label="VERDICT_LABEL[cell.status]"
          />
        </span>
      </span>
    </div>

    <!-- Judging a delivery replaces validating: it is no longer the capture
         being judged, it is the fix. -->
    <div
      v-if="toJudge"
      class="border-t border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-900"
    >
      <p class="mb-2.5 font-mono text-[10.5px] tracking-widest text-slate-500 uppercase">
        fix delivered<span v-if="toJudge.issue"> · issue {{ toJudge.issue.id }}</span>
      </p>
      <p class="mb-3 text-[13px] text-slate-600 dark:text-slate-300">{{ toJudge.body }}</p>

      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded border border-emerald-600 px-3 py-1.5 text-[12.5px] text-emerald-700 disabled:opacity-50 dark:border-emerald-500 dark:text-emerald-400"
          :disabled="busy"
          @click="accept"
        >
          <ActionIcon name="accept" :size="13" />accept
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded border border-amber-600 px-3 py-1.5 text-[12.5px] text-amber-700 disabled:opacity-50 dark:border-amber-500 dark:text-amber-400"
          :disabled="busy || (refusing && remark.trim() === '')"
          @click="refuse"
        >
          <ActionIcon name="refuse" :size="13" />refuse
        </button>
        <span v-if="!refusing" class="font-mono text-[11px] text-slate-500"
          >refusing will ask for your remark</span
        >
      </div>

      <textarea
        v-if="refusing"
        v-model="remark"
        class="mt-2.5 w-full rounded border border-slate-300 bg-white p-2 text-[13px] dark:border-slate-600 dark:bg-slate-950"
        rows="2"
        placeholder="what the developer must take back"
      ></textarea>
    </div>

    <!-- Otherwise: the two things a reviewer does in front of a capture. -->
    <div
      v-else-if="!composing"
      class="flex flex-wrap items-center gap-2 border-t border-slate-200 p-3 dark:border-slate-700"
    >
      <button
        type="button"
        class="inline-flex items-center gap-1.5 rounded border border-emerald-600 bg-emerald-600 px-3 py-1.5 text-[12.5px] text-white disabled:opacity-50"
        :disabled="busy || cell?.status === 'validated'"
        @click="emit('validate', stepId, variantId)"
      >
        <ActionIcon name="check" :size="13" />validate
        <kbd class="rounded border border-current px-1 text-[10px]">space</kbd>
      </button>
      <button
        type="button"
        class="inline-flex items-center gap-1.5 rounded border border-amber-600 px-3 py-1.5 text-[12.5px] text-amber-700 dark:border-amber-500 dark:text-amber-400"
        :disabled="busy"
        @click="openComposer"
      >
        <ActionIcon name="comment" :size="13" />comment
      </button>
      <span
        v-if="cell?.status === 'validated'"
        class="inline-flex items-center gap-1.5 font-mono text-[11px] text-emerald-700 dark:text-emerald-400"
      >
        <StateIcon tone="done" :size="12" />validated
      </span>
      <span
        v-for="c in onSquare"
        :key="c.id"
        class="font-mono text-[11px] text-amber-700 dark:text-amber-400"
      >
        {{ c.body }}
      </span>
    </div>

    <div
      v-else
      class="border-t border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-900"
    >
      <p class="mb-2.5 font-mono text-[10.5px] tracking-widest text-slate-500 uppercase">
        comment on this step
      </p>

      <div class="mb-2.5 flex gap-1.5">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded border px-2.5 py-1 font-mono text-[11px]"
          :class="
            kind === 'defect'
              ? 'border-amber-600 bg-amber-50 text-amber-800 dark:border-amber-500 dark:bg-amber-950 dark:text-amber-300'
              : 'border-slate-300 text-slate-600 dark:border-slate-600 dark:text-slate-300'
          "
          @click="kind = 'defect'"
        >
          <KindIcon kind="defect" :size="12" />defect
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded border px-2.5 py-1 font-mono text-[11px]"
          :class="
            kind === 'improvement'
              ? 'border-indigo-500 bg-indigo-50 text-indigo-700 dark:border-indigo-400 dark:bg-indigo-950 dark:text-indigo-300'
              : 'border-slate-300 text-slate-600 dark:border-slate-600 dark:text-slate-300'
          "
          @click="kind = 'improvement'"
        >
          <KindIcon kind="improvement" :size="12" />improvement
        </button>
      </div>

      <textarea
        v-model="body"
        class="w-full rounded border border-slate-300 bg-white p-2 text-[13px] dark:border-slate-600 dark:bg-slate-950"
        rows="3"
        placeholder="what you see, in your words"
      ></textarea>

      <div class="mt-2.5 flex flex-wrap items-center gap-x-3 gap-y-2 font-mono text-[11px]">
        <label
          v-for="v in grid.variants"
          :key="v.id"
          class="inline-flex cursor-pointer items-center gap-1.5"
        >
          <input
            type="checkbox"
            :checked="chosen.includes(v.id)"
            class="accent-indigo-600"
            @change="toggle(v.id)"
          />
          {{ v.label }}
        </label>
        <span class="ml-auto flex flex-wrap gap-1.5">
          <button
            v-for="value in groups"
            :key="value"
            type="button"
            class="rounded-full border border-slate-300 px-2 py-0.5 text-[10px] text-slate-500 hover:border-indigo-500 hover:text-indigo-600 dark:border-slate-600 dark:text-slate-400"
            @click="pickValue(value)"
          >
            {{ value }}
          </button>
          <button
            type="button"
            class="rounded-full border border-slate-300 px-2 py-0.5 text-[10px] text-slate-500 hover:border-indigo-500 hover:text-indigo-600 dark:border-slate-600 dark:text-slate-400"
            @click="pickAll"
          >
            all
          </button>
        </span>
      </div>

      <div class="mt-3 flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="rounded border border-indigo-600 bg-indigo-600 px-3 py-1.5 text-[12.5px] text-white disabled:border-slate-300 disabled:bg-slate-100 disabled:text-slate-400 dark:disabled:border-slate-600 dark:disabled:bg-slate-800"
          :disabled="!canAdd || busy"
          @click="add"
        >
          add
        </button>
        <button
          type="button"
          class="rounded border border-slate-300 px-3 py-1.5 text-[12.5px] text-slate-600 dark:border-slate-600 dark:text-slate-300"
          @click="composing = false"
        >
          cancel
        </button>
        <span class="font-mono text-[11px] text-slate-500">
          {{ chosen.length }} variant{{ chosen.length > 1 ? 's' : '' }} ticked
        </span>
      </div>
    </div>
  </div>
</template>
