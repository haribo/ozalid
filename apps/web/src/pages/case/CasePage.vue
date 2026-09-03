<script setup lang="ts">
/**
 * One case: what it is, the evidence it is judged from, and — when a capture is
 * open — the carousel where judging happens.
 */
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api, type components } from '@/shared/api'
import { MissingIcon, MovedIcon, StatePill } from '@/shared/ui'
import { formatMoment, hasMoved, type CaseState } from '@/shared/lib'
import { useReview } from '@/features/review'
import { useSession } from '@/features/session'
import { CaseGrid } from '@/widgets/case-grid'
import { CaptureCarousel } from '@/widgets/capture-carousel'
import { CommentRecap } from '@/widgets/comment-recap'

type Case = components['schemas']['Case']

const route = useRoute()
const slug = computed(() => String(route.params.slug))
const caseId = computed(() => String(route.params.caseId))

const kase = ref<Case | null>(null)
const loading = ref(true)
const open = ref<{ stepId: string; variantId: string } | null>(null)

const review = useReview(
  () => slug.value,
  () => caseId.value,
)

// The page is where the two meet: the session knows it came back, the review
// knows what it was holding, and neither may reach into the other
// (frontend ADR 0002).
const { standing } = useSession()
watch(standing, (now, before) => {
  if (before === 'expired' && now === 'in') void review.resume()
})

watch(
  caseId,
  async (id) => {
    loading.value = true
    open.value = null

    const detail = await api.GET('/projects/{slug}/cases/{caseId}', {
      params: { path: { slug: slug.value, caseId: id } },
    })
    if (detail.error) {
      review.error.value = detail.error.title
      loading.value = false
      return
    }
    kase.value = detail.data
    await review.load()
    loading.value = false
  },
  { immediate: true },
)

/** How the review stands, in one line — an information, not a gate.
 *
 * `missing` counts the holes: a case is meant to carry a capture for every
 * variant its own run declared, and a gap is a failed run rather than a
 * deliberate absence (ADR 0016). Counting it here is what keeps it from being
 * discovered months later by whoever trusted the gauge. */
const tally = computed(() => {
  const grid = review.grid.value
  const cells = grid?.steps.flatMap((s) => s.cells) ?? []
  const count = (status: string) => cells.filter((c) => c.status === status).length
  const expected = (grid?.steps.length ?? 0) * (grid?.variants.length ?? 0)
  return {
    validated: count('validated'),
    commented: count('to-fix'),
    toJudge: count('to-review'),
    missing: Math.max(0, expected - cells.length),
    // Counted like the holes, and for the same reason: a reviewer should not
    // have to scan the grid to learn there is work waiting.
    moved: cells.filter((c) => hasMoved(c.freshness)).length,
  }
})

async function onValidate(stepId: string, variantId: string) {
  await review.validate(stepId, variantId)
  await refreshCase()
}

async function onComment(input: Parameters<typeof review.comment>[0]) {
  await review.comment(input)
  await refreshCase()
}

async function onJudge(commentId: string, accept: boolean, remark: string) {
  await review.judge(commentId, accept, remark)
  await refreshCase()
}

/** The case's own state is recomputed by the server on every move, so it is
 * read back rather than guessed here (ADR 0012). */
async function refreshCase() {
  const detail = await api.GET('/projects/{slug}/cases/{caseId}', {
    params: { path: { slug: slug.value, caseId: caseId.value } },
  })
  if (!detail.error) kase.value = detail.data
}
</script>

<template>
  <div class="mx-auto max-w-6xl px-6 py-8">
    <p v-if="review.error.value" class="font-mono text-[12px] text-red-700 dark:text-red-400">
      {{ review.error.value }}
    </p>
    <p v-else-if="loading" class="font-mono text-[12px] text-slate-500">loading…</p>

    <template v-else-if="kase">
      <h1 class="mb-2.5 text-[21px] font-semibold">{{ kase.title }}</h1>

      <div
        class="mb-3 flex flex-wrap items-center gap-x-3 gap-y-1.5 font-mono text-[11px] text-slate-500 dark:text-slate-400"
      >
        <StatePill :state="kase.state as CaseState" />
        <span>#{{ kase.id }}</span>
        <template v-if="review.grid.value?.editionId">
          <span>·</span>
          <span>edition of {{ formatMoment(review.grid.value.takenAt) }}</span>
          <template v-if="review.grid.value.revision">
            <span>·</span>
            <span>rev {{ review.grid.value.revision }}</span>
          </template>
        </template>
        <template v-if="tally.validated + tally.commented + tally.toJudge > 0">
          <span>·</span>
          <span>
            {{ tally.validated }} validated
            <template v-if="tally.commented"> · {{ tally.commented }} commented</template>
            <template v-if="tally.toJudge"> · {{ tally.toJudge }} to judge</template>
          </span>
        </template>
        <span
          v-if="tally.moved > 0"
          class="inline-flex items-center gap-1.5 rounded border border-indigo-500 bg-indigo-50 px-1.5 py-0.5 text-indigo-700 dark:border-indigo-400 dark:bg-indigo-950/60 dark:text-indigo-300"
        >
          <MovedIcon :size="10" />
          {{ tally.moved }} capture{{ tally.moved > 1 ? 's' : '' }}
          {{ tally.moved > 1 ? 'have' : 'has' }}
          moved
        </span>
        <span
          v-if="tally.missing > 0"
          class="inline-flex items-center gap-1.5 rounded border border-red-600 bg-red-50 px-1.5 py-0.5 text-red-700 dark:border-red-500 dark:bg-red-950/60 dark:text-red-400"
        >
          <MissingIcon :size="10" />
          {{ tally.missing }} capture{{ tally.missing > 1 ? 's' : '' }} manquante{{
            tally.missing > 1 ? 's' : ''
          }}
        </span>
      </div>

      <CaptureCarousel
        v-if="open && review.grid.value"
        :slug="slug"
        :grid="review.grid.value"
        :comments="review.comments.value"
        :step-id="open.stepId"
        :variant-id="open.variantId"
        :busy="review.saving.value"
        class="mb-5"
        @close="open = null"
        @move="(stepId, variantId) => (open = { stepId, variantId })"
        @validate="onValidate"
        @comment="onComment"
        @judge="onJudge"
      />

      <CaseGrid
        v-if="review.grid.value"
        :slug="slug"
        :grid="review.grid.value"
        :open-cell="open"
        @open="(stepId, variantId) => (open = { stepId, variantId })"
      />

      <CommentRecap
        v-if="review.grid.value"
        :grid="review.grid.value"
        :comments="review.comments.value"
        @open="(stepId, variantId) => (open = { stepId, variantId })"
      />
    </template>
  </div>
</template>
