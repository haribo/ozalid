<script setup lang="ts">
/**
 * Where judging happens.
 *
 * One capture at full size, the two verdicts within reach, and the keyboard
 * doing the work: arrows to move, space to validate, Escape to leave. On a
 * case of twelve captures the reviewer never touches the mouse.
 */
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import type { components } from '@/shared/api'
import { AppButton, MovedIcon, VerdictPair } from '@/shared/ui'

type Grid = components['schemas']['Grid']
type Comment = components['schemas']['Comment']

const props = defineProps<{
  slug: string
  grid: Grid
  comments: Comment[]
  stepId: string
  variantId: string
  /** The recording view (ADR 0023): the player instead of a capture, the
   * pair judging the video of this variant at the current edition. */
  recording?: boolean
  busy?: boolean
}>()

const emit = defineEmits<{
  close: []
  move: [stepId: string, variantId: string]
  /** Accept the capture; withdraw carries the capture's draft refusals when the
   * verdict is switched in one gesture (ADR 0020). */
  accept: [stepId: string, variantId: string, withdraw: boolean]
  unaccept: [stepId: string, variantId: string]
  /** Refuse the capture: the remark and its variants, plus the acceptance
   * take-back when one stood (one gesture, never two). */
  refuse: [input: { stepId: string; body: string; variantIds: string[]; unaccept: boolean }]
  unrefuse: [stepId: string, variantId: string]
  edit: [commentId: string, body: string, variantIds: string[]]
  /** A judgment lands on the capture on screen (ADR 0022): the variant rides
   * along so the server releases or restores exactly this coverage. */
  judge: [commentId: string, issueRefId: string, accept: boolean, remark: string, variantId: string]
  /** Empty variantId keeps the take-back ref-level (a refusal reopening). */
  unjudge: [commentId: string, issueRefId: string, variantId: string]
  moveRecording: [variantId: string]
  judgeRecording: [recordingId: string, accept: boolean, remark: string]
  unjudgeRecording: [recordingId: string]
}>()

/** Where the open step sits in the flow. The counter counts steps: the
 * horizontal walk is the flow, and the variant is a lens on it (#149). */
const stepIndex = computed(() => props.grid.steps.findIndex((s) => s.id === props.stepId))

/** The video of this variant at the current edition (ADR 0023). */
const rec = computed(() =>
  props.recording ? props.grid.recordings.find((r) => r.variantId === props.variantId) : undefined,
)

const step = computed(() => props.grid.steps.find((s) => s.id === props.stepId))
const variant = computed(() => props.grid.variants.find((v) => v.id === props.variantId))
const capture = computed(() => step.value?.captures.find((c) => c.variantId === props.variantId))

/** The comments covering this exact capture — what is already known about it. */
const onSquare = computed(() =>
  props.comments.filter((c) => c.stepId === props.stepId && c.variantIds.includes(props.variantId)),
)

/** The issue refs of the comments covering this capture, each with its owner. */
const refsOnSquare = computed(() =>
  onSquare.value.flatMap((c) => (c.issues ?? []).map((tracked) => ({ comment: c, ref: tracked }))),
)

/** A delivery waiting for a verdict. That is what gets judged here, not the
 * capture itself. One at a time: the capture's other refs live in the recap,
 * and the next delivered one takes this spot once this one is judged (#171). */
const toJudge = computed(() =>
  refsOnSquare.value.find(({ ref: tracked }) => tracked.state === 'to-review'),
)

/** Accepting released this variant from the coverage (ADR 0022), so an
 * acceptance is read off the judgment history — the last judgment naming this
 * variant — never off the coverage. History from before the rule kept its
 * coverage, so the coverage-based reading stays as the fallback. */
const acceptedHere = computed(() =>
  props.comments.filter(
    (c) =>
      c.stepId === props.stepId &&
      (c.judgments ?? []).filter((j) => j.variantId === props.variantId).at(-1)?.verdict ===
        'accepted',
  ),
)
const acceptedRef = computed(
  () =>
    acceptedHere.value
      .flatMap((c) => (c.issues ?? []).map((tracked) => ({ comment: c, ref: tracked })))
      .at(0) ?? refsOnSquare.value.find(({ ref: tracked }) => tracked.state === 'accepted'),
)
const refusedRef = computed(() =>
  refsOnSquare.value.find(({ ref: tracked }) => tracked.state === 'refused'),
)

/** The branch loop (#175): a ref-less remark delivered, judged or settled as
 * itself — no issue ever attached, the remark's own words do the talking. */
const remarkToJudge = computed(() =>
  onSquare.value.find((c) => (c.issues ?? []).length === 0 && c.state === 'to-review'),
)
const acceptedRemark = computed(
  () =>
    acceptedHere.value.find((c) => (c.issues ?? []).length === 0) ??
    onSquare.value.find((c) => (c.issues ?? []).length === 0 && c.state === 'accepted'),
)
const refusedRemark = computed(() =>
  onSquare.value.find((c) => (c.issues ?? []).length === 0 && c.state === 'refused'),
)

/** The reviewer's own drafts on this capture: remarks no issue is attached to
 * yet. Editable and withdrawable — they never counted anywhere (ADR 0020). */
const drafts = computed(() =>
  onSquare.value.filter((c) => (c.issues ?? []).length === 0 && c.state === 'to-track'),
)

/** Tracked remarks speak through their issue titles; the drafts through their
 * own words. Settled and discarded ones say nothing here. */
const trackedTitles = computed(() =>
  onSquare.value
    .filter((c) => c.state === 'tracked')
    .flatMap((c) => (c.issues ?? []).map((r) => `#${r.issueId} ${r.title ?? ''}`.trim())),
)

/** The context line: one grammar for every state (ADR 0022) — the issue and
 * its title, or the bare remark's own words. The verdict reads on the pair. */
const contextRef = computed(() => toJudge.value ?? acceptedRef.value ?? refusedRef.value)
const contextRemark = computed(
  () => remarkToJudge.value ?? acceptedRemark.value ?? refusedRemark.value,
)

/** What the pair shows. The fix's judgment outranks the capture's own status:
 * when an issue is on this capture, the verdict is about the fix. */
const verdict = computed<'none' | 'accepted' | 'refused'>(() => {
  if (props.recording) {
    if (rec.value?.status === 'accepted') return 'accepted'
    if (rec.value?.status === 'refused') return 'refused'
    return 'none'
  }
  if (toJudge.value || remarkToJudge.value) return 'none'
  if (acceptedRef.value) return 'accepted'
  if (refusedRef.value) return 'refused'
  if (capture.value?.status === 'accepted') return 'accepted'
  if (capture.value?.status === 'refused') return 'refused'
  return 'none'
})

/** A capture that has moved is back to needing eyes, whatever its verdict
 * says. The image itself never wears a mark (ADR 0020): the badge says it
 * moved, the bar says the verdict, the grid keeps its discs. */
const moved = computed(() => capture.value?.status === 'moved')

/** Left and right walk the steps, keeping the variant; a step that lacks it
 * is skipped rather than switching the lens under the reviewer (#149). */
function go(delta: number) {
  if (props.recording) {
    // The video is the walk's first position: right enters the steps.
    if (delta > 0) {
      const first = props.grid.steps.find((s) =>
        s.captures.some((c) => c.variantId === props.variantId),
      )
      if (first) emit('move', first.id, props.variantId)
    }
    return
  }
  for (let i = stepIndex.value + delta; i >= 0 && i < props.grid.steps.length; i += delta) {
    const candidate = props.grid.steps[i]
    if (candidate.captures.some((c) => c.variantId === props.variantId)) {
      emit('move', candidate.id, props.variantId)
      return
    }
  }
  // Left past the first step lands on the video, when there is one.
  if (delta < 0 && props.grid.recordings.some((r) => r.variantId === props.variantId)) {
    emit('moveRecording', props.variantId)
  }
}

/** Up and down move to the same step's next variant. */
function goVariant(delta: number) {
  const here = props.grid.variants.findIndex((v) => v.id === props.variantId)
  const next = props.grid.variants[here + delta]
  if (!next) return
  if (props.recording) {
    if (props.grid.recordings.some((r) => r.variantId === next.id)) {
      emit('moveRecording', next.id)
    }
    return
  }
  if (step.value?.captures.some((c) => c.variantId === next.id)) {
    emit('move', props.stepId, next.id)
  }
}

// ---- the refuse / edit sheet ----------------------------------------------

/** Open, the sheet is the only editing surface: the pair goes inert and the
 * capture shrinks — it stays on screen, the remark describes pixels the
 * reviewer can still see (ADR 0020). */
const sheet = ref<null | { editing: string | null }>(null)
const remark = ref('')
const chosen = ref<string[]>([])

/** Group shortcuts, derived from the axis values actually present — the
 * project declares its own axes and ozalid ships no list of them (ADR 0001). */
const groups = computed(() => {
  const values = new Set<string>()
  for (const v of props.grid.variants) {
    for (const value of Object.values(v.values)) values.add(value)
  }
  return [...values].toSorted()
})

function toggle(id: string) {
  chosen.value = chosen.value.includes(id)
    ? chosen.value.filter((v) => v !== id)
    : [...chosen.value, id]
}

function pickValue(value: string) {
  chosen.value = props.grid.variants
    .filter((v) => Object.values(v.values).includes(value))
    .map((v) => v.id)
}

function pickAll() {
  chosen.value = props.grid.variants.map((v) => v.id)
}

const canSend = computed(() => {
  if (remark.value.trim() === '') return false
  // A fix's or a recording's refusal needs no variants: the thing being
  // judged already owns its own.
  return props.recording || toJudge.value || remarkToJudge.value ? true : chosen.value.length > 0
})

function openSheet() {
  sheet.value = { editing: null }
  remark.value = ''
  chosen.value = [props.variantId]
}

function openEdit(commentId: string) {
  const draft = drafts.value.find((c) => c.id === commentId)
  if (!draft) return
  sheet.value = { editing: commentId }
  remark.value = draft.body
  chosen.value = [...draft.variantIds]
}

function closeSheet() {
  sheet.value = null
  remark.value = ''
}

function send() {
  if (!sheet.value || !canSend.value) return
  const body = remark.value.trim()
  if (props.recording) {
    if (rec.value) emit('judgeRecording', rec.value.id, false, body)
    closeSheet()
    return
  }
  if (sheet.value.editing) {
    emit('edit', sheet.value.editing, body, [...chosen.value])
  } else if (toJudge.value) {
    emit('judge', toJudge.value.comment.id, toJudge.value.ref.id, false, body, props.variantId)
  } else if (remarkToJudge.value) {
    emit('judge', remarkToJudge.value.id, '', false, body, props.variantId)
  } else {
    emit('refuse', {
      stepId: props.stepId,
      body,
      variantIds: [...chosen.value],
      unaccept: capture.value?.status === 'accepted',
    })
  }
  closeSheet()
}

// ---- the pair's reading of a click ----------------------------------------

function onAccept() {
  if (props.busy || sheet.value) return
  if (props.recording) {
    if (!rec.value) return
    // Accepted un-presses; refused or unjudged accepts — one gesture, the
    // judgment lands on these bytes (ADR 0023).
    if (rec.value.status === 'accepted') emit('unjudgeRecording', rec.value.id)
    else emit('judgeRecording', rec.value.id, true, '')
    return
  }
  if (toJudge.value) {
    emit('judge', toJudge.value.comment.id, toJudge.value.ref.id, true, '', props.variantId)
    return
  }
  if (remarkToJudge.value) {
    emit('judge', remarkToJudge.value.id, '', true, '', props.variantId)
    return
  }
  if (acceptedRemark.value) {
    emit('unjudge', acceptedRemark.value.id, '', props.variantId)
    return
  }
  if (acceptedRef.value) {
    // The filled half un-presses: the judgment is taken back (#167).
    emit('unjudge', acceptedRef.value.comment.id, acceptedRef.value.ref.id, props.variantId)
    return
  }
  if (verdict.value === 'accepted') {
    emit('unaccept', props.stepId, props.variantId)
    return
  }
  // Refused → accepted is one gesture: the draft goes with its refusal.
  emit('accept', props.stepId, props.variantId, verdict.value === 'refused')
}

function onRefuse() {
  if (props.busy || sheet.value) return
  if (props.recording) {
    if (!rec.value) return
    if (rec.value.status === 'refused') emit('unjudgeRecording', rec.value.id)
    else openSheet()
    return
  }
  if (refusedRef.value) {
    emit('unjudge', refusedRef.value.comment.id, refusedRef.value.ref.id, '')
    return
  }
  if (refusedRemark.value) {
    emit('unjudge', refusedRemark.value.id, '', '')
    return
  }
  if (verdict.value === 'refused') {
    // Withdraw the draft refusal: the reviewer's own remark goes with it.
    emit('unrefuse', props.stepId, props.variantId)
    return
  }
  openSheet()
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
      // The sheet first, the carousel second.
      if (sheet.value) closeSheet()
      else emit('close')
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
      // Space plays accept — give or take back (ADR 0020). On the video it
      // belongs to the player: play and pause, the pair is clicked.
      if (props.recording) return
      event.preventDefault()
      if (sheet.value) break
      onAccept()
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
      class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 bg-slate-50 px-3 py-2 font-mono text-mono text-slate-500 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-400"
    >
      <span>
        <b class="font-medium text-slate-900 dark:text-slate-100">{{
          recording ? 'recording' : step?.name
        }}</b>
        · {{ variant?.label }}
      </span>
      <span>
        <template v-if="!recording">{{ stepIndex + 1 }} / {{ grid.steps.length }} · </template>
        <!-- Arrow glyphs are missing from most monospace faces and render as
             empty boxes; the system font has them. -->
        <kbd class="rounded border border-current px-1 font-sans">←</kbd>
        <kbd class="rounded border border-current px-1 font-sans">→</kbd> step ·
        <kbd class="rounded border border-current px-1 font-sans">↑</kbd>
        <kbd class="rounded border border-current px-1 font-sans">↓</kbd> variant ·
        <kbd class="rounded border border-current px-1">space</kbd>
        {{ recording ? 'play' : 'accept' }} ·
        <kbd class="rounded border border-current px-1">Esc</kbd> close
      </span>
    </div>

    <!-- Flex, not grid: a grid's auto row grows with its content, and the
         max-h chain below then constrains nothing — a tall capture overflowed
         the stage instead of scaling (#177). Every link of the chain carries
         min-h-0 so the percentages stay bound to the stage. -->
    <div
      class="relative flex min-h-0 flex-1 items-center justify-center overflow-hidden bg-slate-100 p-6 dark:bg-slate-950"
    >
      <!-- The capture at full strength, always: no veil, no disc — the
           verdict lives in the bar, and the grid keeps its marks (ADR 0020).
           When the sheet is open the stage shrinks with the flex column: the
           pixels stay on screen while the remark is written. -->
      <!-- Said over the stage: the reviewer landed here from a keyboard walk
           and never saw the grid's mark. Anchored to the stage corner, not the
           image — the image's own box is what scales (#177). -->
      <span
        v-if="capture && moved"
        class="absolute top-3 right-3 z-10 flex items-center gap-1.5 rounded border border-indigo-500 bg-white px-2 py-1 font-mono text-mono text-indigo-700 dark:border-indigo-400 dark:bg-slate-900 dark:text-indigo-300"
      >
        <MovedIcon :size="12" />moved<template v-if="capture.movedPixels !== undefined">
          · {{ capture.movedPixels }} px</template
        >
      </span>
      <!-- The capture takes the space the window offers and never leaves it:
           the wrapper fills the stage, so the max constraints bind against a
           definite box and a capture of any size scales to fit (#177, #125). -->
      <span v-if="capture" class="flex h-full w-full min-h-0 items-center justify-center">
        <img
          :src="`/api/projects/${slug}/captures/${capture.id}`"
          :alt="`${step?.name} — ${variant?.label}`"
          class="max-h-full max-w-full border border-slate-300 bg-white dark:border-slate-600 dark:bg-slate-900"
        />
      </span>
      <!-- The player: the browser's own controls, streaming the sniffed
           content type. The pair below judges these exact bytes (ADR 0023). -->
      <span v-if="rec" class="flex h-full w-full min-h-0 items-center justify-center">
        <!-- Unlike a capture, the video scales up: nothing pixel-accurate is
             being judged here (ADR 0013), and a small source in a big stage
             is just hard to watch (#232). object-contain letterboxes. -->
        <video
          :src="`/api/projects/${slug}/recordings/${rec.id}`"
          controls
          :aria-label="`recording — ${variant?.label}`"
          class="h-full w-full border border-slate-300 bg-white object-contain dark:border-slate-600 dark:bg-slate-900"
        ></video>
      </span>
    </div>

    <!-- The judgment zone: one centred grammar for every state (#171). The
         verdict's echo tints the bar's top edge; everything else is the pair,
         its context line, the draft cards, and the sheet while it is open. -->
    <div
      class="flex flex-col items-center gap-2.5 border-t border-slate-200 p-3 text-center dark:border-slate-700"
      :class="{
        'shadow-[inset_0_2px_0_theme(colors.emerald.600)] dark:shadow-[inset_0_2px_0_theme(colors.emerald.500)]':
          verdict === 'accepted' && !sheet,
        'shadow-[inset_0_2px_0_theme(colors.amber.600)] dark:shadow-[inset_0_2px_0_theme(colors.amber.500)]':
          verdict === 'refused' && !sheet,
      }"
    >
      <!-- The video's standing refusal speaks; nothing else has a line
           here — a recording carries no issues (ADR 0023). -->
      <p
        v-if="recording && rec?.refusal"
        class="max-w-[56ch] font-mono text-mono text-amber-700 dark:text-amber-400"
      >
        {{ rec.refusal }}
      </p>
      <!-- The context line, identical whatever the round's state (ADR 0022):
           the issue and its title. The verdict reads on the pair alone. -->
      <p
        v-if="!recording && contextRef"
        class="max-w-[56ch] text-body text-slate-600 dark:text-slate-300"
      >
        <a
          v-if="contextRef.ref.url"
          :href="contextRef.ref.url"
          target="_blank"
          rel="noopener"
          class="text-indigo-700 hover:underline dark:text-indigo-300"
          >issue #{{ contextRef.ref.issueId }}</a
        ><span v-else>issue #{{ contextRef.ref.issueId }}</span
        >: {{ contextRef.ref.title || contextRef.comment.body }}
      </p>
      <!-- The branch loop (#175): no number — the remark's own words talk. -->
      <p
        v-else-if="contextRemark"
        class="max-w-[56ch] text-body text-slate-600 dark:text-slate-300"
      >
        {{ contextRemark.body }}
      </p>

      <VerdictPair
        :verdict="verdict"
        :disabled="busy || sheet !== null"
        @accept="onAccept"
        @refuse="onRefuse"
      />

      <!-- Tracked remarks speak through their issue titles; the drafts are
           the reviewer's own — the whole card is the way into editing. -->
      <template v-if="!sheet && !recording">
        <p
          v-for="title in trackedTitles"
          :key="title"
          class="font-mono text-mono text-slate-500 dark:text-slate-400"
        >
          {{ title }}
        </p>
        <AppButton
          v-for="draft in drafts"
          :key="draft.id"
          variant="secondary"
          class="w-full max-w-md justify-between border-slate-300 font-normal normal-case dark:border-slate-600"
          :aria-label="`edit the remark: ${draft.body}`"
          @click="openEdit(draft.id)"
        >
          <span class="flex-1 text-left font-mono text-mono text-amber-700 dark:text-amber-400">{{
            draft.body
          }}</span>
          <span aria-hidden="true" class="text-slate-400">✎</span>
        </AppButton>
      </template>

      <!-- The sheet: under the image, which stays on screen. Cancel is a true
           no-op, whatever verdict already stood. -->
      <form
        v-if="sheet"
        class="flex w-full max-w-md flex-col gap-2.5 rounded-md border border-slate-200 bg-slate-50 p-3 text-left dark:border-slate-700 dark:bg-slate-900"
        @submit.prevent="send"
      >
        <h3 class="text-body font-semibold">
          {{
            sheet.editing
              ? 'Edit the remark'
              : recording
                ? 'Refuse the recording'
                : toJudge
                  ? `Refuse the fix — issue #${toJudge.ref.issueId}`
                  : remarkToJudge
                    ? 'Refuse the fix'
                    : 'Refuse the capture'
          }}
        </h3>
        <label class="flex flex-col gap-1">
          <span class="font-mono text-label tracking-wider text-slate-500 uppercase">
            remark <b class="text-amber-700 dark:text-amber-400">*</b>
          </span>
          <textarea
            v-model="remark"
            class="w-full rounded border border-slate-300 bg-white p-2 text-body dark:border-slate-600 dark:bg-slate-950"
            rows="3"
          ></textarea>
        </label>
        <div
          v-if="!recording && !toJudge && !remarkToJudge"
          class="flex flex-wrap items-center gap-x-3 gap-y-2 font-mono text-mono"
        >
          <label
            v-for="v in grid.variants"
            :key="v.id"
            class="inline-flex cursor-pointer items-center gap-1.5"
          >
            <!-- eslint-disable-next-line vue/no-restricted-html-elements -- a checkbox, not a text field (#155) -->
            <input
              type="checkbox"
              :checked="chosen.includes(v.id)"
              class="accent-indigo-600"
              @change="toggle(v.id)"
            />
            {{ v.label }}
          </label>
          <span class="ml-auto flex flex-wrap gap-1.5">
            <AppButton
              v-for="value in groups"
              :key="value"
              variant="secondary"
              @click="pickValue(value)"
            >
              {{ value }}
            </AppButton>
            <AppButton variant="secondary" @click="pickAll"> all </AppButton>
          </span>
        </div>
        <div class="flex items-center justify-end gap-2">
          <AppButton variant="ghost" @click="closeSheet"> cancel </AppButton>
          <AppButton variant="destructive" :disabled="!canSend || busy" @click="send">
            {{ sheet.editing ? 'save' : 'refuse' }}
          </AppButton>
        </div>
      </form>
    </div>
  </div>
</template>
