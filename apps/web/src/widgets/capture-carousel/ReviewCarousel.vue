<script setup lang="ts">
/**
 * The carousel wired to a review session.
 *
 * The judging screen is one component (`CaptureCarousel`) and the wiring
 * between its verdicts and the session that writes them is one component too:
 * two pages open the carousel — a case, and a queue walked across cases
 * (#205) — and a second copy of these fifteen handlers is exactly the drift
 * #155 was opened for.
 *
 * It owns no state. Every verdict goes straight to the session it was handed,
 * and `changed` lets the page read back whatever the server recomputed
 * (ADR 0012).
 */
import type { components } from '@/shared/api'
import type { useReview } from '@/features/review'
import CaptureCarousel, { type Walk } from './CaptureCarousel.vue'

type QueueEntry = components['schemas']['QueueEntry']

const props = defineProps<{
  slug: string
  review: ReturnType<typeof useReview>
  stepId: string
  variantId: string
  recording?: boolean
  walk?: Walk
}>()

const emit = defineEmits<{
  close: []
  move: [stepId: string, variantId: string]
  moveRecording: [variantId: string]
  moveEntry: [entry: QueueEntry]
  /** A verdict landed: the server recomputed what it recomputes, and the page
   * decides what to read back. */
  changed: []
}>()

async function accept(stepId: string, variantId: string, withdraw: boolean) {
  await props.review.accept(stepId, variantId, withdraw)
  emit('changed')
}

async function unaccept(stepId: string, variantId: string) {
  await props.review.unaccept(stepId, variantId)
  emit('changed')
}

async function refuse(input: Parameters<ReturnType<typeof useReview>['refuse']>[0]) {
  await props.review.refuse(input)
  emit('changed')
}

async function unrefuse(stepId: string, variantId: string) {
  await props.review.unrefuse(stepId, variantId)
  emit('changed')
}

async function edit(commentId: string, body: string, variantIds: string[]) {
  await props.review.edit(commentId, body, variantIds)
  emit('changed')
}

async function judge(
  commentId: string,
  issueRefId: string,
  accepted: boolean,
  remark: string,
  variantId: string,
) {
  await props.review.judge(commentId, issueRefId, accepted, remark, variantId)
  emit('changed')
}

async function unjudge(commentId: string, issueRefId: string, variantId: string) {
  await props.review.unjudge(commentId, issueRefId, variantId)
  emit('changed')
}

async function judgeRecording(recordingId: string, accepted: boolean, remark: string) {
  await props.review.judgeRecording(recordingId, accepted, remark)
  emit('changed')
}

async function unjudgeRecording(recordingId: string) {
  await props.review.unjudgeRecording(recordingId)
  emit('changed')
}
</script>

<template>
  <CaptureCarousel
    v-if="review.grid.value"
    :slug="slug"
    :grid="review.grid.value"
    :comments="review.comments.value"
    :step-id="stepId"
    :variant-id="variantId"
    :recording="recording"
    :walk="walk"
    :held-by="review.lockedBy.value"
    :busy="review.saving.value"
    @close="emit('close')"
    @move="(stepId, variantId) => emit('move', stepId, variantId)"
    @move-recording="(variantId) => emit('moveRecording', variantId)"
    @move-entry="(entry) => emit('moveEntry', entry)"
    @accept="accept"
    @unaccept="unaccept"
    @refuse="refuse"
    @unrefuse="unrefuse"
    @edit="edit"
    @judge="judge"
    @unjudge="unjudge"
    @judge-recording="judgeRecording"
    @unjudge-recording="unjudgeRecording"
  />
</template>
