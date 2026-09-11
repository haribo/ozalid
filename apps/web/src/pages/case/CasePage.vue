<script setup lang="ts">
/**
 * One case: what it is, the evidence it is judged from, and — when a capture is
 * open — the carousel where judging happens.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, type components } from '@/shared/api'
import { StatePill } from '@/shared/ui'
import { formatMoment, type CaseState } from '@/shared/lib'
import { useReview } from '@/features/review'
import { useSession } from '@/features/session'
import { CaseGrid } from '@/widgets/case-grid'
import { CaptureCarousel } from '@/widgets/capture-carousel'
import { CommentRecap } from '@/widgets/comment-recap'

type Case = components['schemas']['Case']

const route = useRoute()
const router = useRouter()
const slug = computed(() => String(route.params.slug))
const caseId = computed(() => String(route.params.caseId))

const kase = ref<Case | null>(null)
type Category = components['schemas']['Category']
const categories = ref<Category[]>([])
const loading = ref(true)

/** The way back: the case's ancestors, root first — the same trail the
 * catalogue draws, ancestors only. The title right below says the current
 * page, so repeating it in the trail would be noise (#190). */
const trail = computed(() => {
  const out: Category[] = []
  let id = kase.value?.categoryId ?? null
  while (id) {
    const node = categories.value.find((c) => c.id === id)
    if (!node) break
    out.unshift(node)
    id = node.parentId ?? null
  }
  return out
})

// Which capture is open is the route's to say, not a ref's: an open capture
// has an address, so a colleague can be sent to the exact capture (#125). The
// two routes share this component, which is what keeps the instance — and a
// verdict held through an expired session (#70) — alive across open and close.
const open = computed(() =>
  route.params.stepId && route.params.variantId
    ? { stepId: String(route.params.stepId), variantId: String(route.params.variantId) }
    : null,
)

/** The recording view: same carousel, addressed by variant (ADR 0023). */
const openRecording = computed(() =>
  route.path.includes('/recordings/') && route.params.variantId
    ? String(route.params.variantId)
    : null,
)

const caseUrl = computed(() => `/projects/${slug.value}/cases/${caseId.value}`)

function openCapture(stepId: string, variantId: string) {
  void router.push(`${caseUrl.value}/steps/${stepId}/variants/${variantId}`)
}

function openRecordingView(variantId: string) {
  void router.push(`${caseUrl.value}/recordings/${variantId}`)
}

function moveToRecording(variantId: string) {
  void router.replace(`${caseUrl.value}/recordings/${variantId}`)
}

/** Arrow keys walk, they do not stack: replace, so back means the grid. */
function moveTo(stepId: string, variantId: string) {
  void router.replace(`${caseUrl.value}/steps/${stepId}/variants/${variantId}`)
}

/** Push rather than back(): a link opened straight onto a capture has no
 * page behind it to go back to. */
function closeCarousel() {
  void router.push(caseUrl.value)
}

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

    const detail = await api.GET('/projects/{slug}/cases/{caseId}', {
      params: { path: { slug: slug.value, caseId: id } },
    })
    if (detail.error) {
      review.error.value = detail.error.title
      loading.value = false
      return
    }
    kase.value = detail.data
    const tree = await api.GET('/projects/{slug}/categories', {
      params: { path: { slug: slug.value } },
    })
    categories.value = tree.error ? [] : tree.data
    await review.load()
    // Opening the case is claiming it (ADR 0005, #95); the interval is the
    // heartbeat, and leaving the page lets go.
    await review.claim()
    loading.value = false
  },
  { immediate: true },
)

const HEARTBEAT_MS = 30_000
let heartbeat: ReturnType<typeof setInterval> | undefined

/** Leaving is letting go — including by closing the tab or a hard
 * navigation, where Vue never unmounts: keepalive lets the release outlive
 * the page (ADR 0005). */
function releaseOnLeave() {
  void fetch(`/api/projects/${slug.value}/cases/${caseId.value}/lock`, {
    method: 'DELETE',
    keepalive: true,
  })
}
onMounted(() => {
  heartbeat = setInterval(() => void review.claim(), HEARTBEAT_MS)
  window.addEventListener('pagehide', releaseOnLeave)
})
onBeforeUnmount(() => {
  clearInterval(heartbeat)
  window.removeEventListener('pagehide', releaseOnLeave)
  void review.release()
})

async function onAccept(stepId: string, variantId: string, withdraw: boolean) {
  await review.accept(stepId, variantId, withdraw)
  await refreshCase()
}

async function onUnaccept(stepId: string, variantId: string) {
  await review.unaccept(stepId, variantId)
  await refreshCase()
}

async function onRefuse(input: Parameters<typeof review.refuse>[0]) {
  await review.refuse(input)
  await refreshCase()
}

async function onUnrefuse(stepId: string, variantId: string) {
  await review.unrefuse(stepId, variantId)
  await refreshCase()
}

async function onEdit(commentId: string, body: string, variantIds: string[]) {
  await review.edit(commentId, body, variantIds)
  await refreshCase()
}

async function onJudge(
  commentId: string,
  issueRefId: string,
  accept: boolean,
  remark: string,
  variantId: string,
) {
  await review.judge(commentId, issueRefId, accept, remark, variantId)
  await refreshCase()
}

async function onUnjudge(commentId: string, issueRefId: string, variantId: string) {
  await review.unjudge(commentId, issueRefId, variantId)
  await refreshCase()
}

async function onJudgeRecording(recordingId: string, accept: boolean, remark: string) {
  await review.judgeRecording(recordingId, accept, remark)
  await refreshCase()
}

async function onUnjudgeRecording(recordingId: string) {
  await review.unjudgeRecording(recordingId)
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
    <p v-if="review.error.value" class="font-mono text-mono text-red-700 dark:text-red-400">
      {{ review.error.value }}
    </p>
    <p v-else-if="loading" class="font-mono text-mono text-slate-500">loading…</p>

    <template v-else-if="kase">
      <nav
        aria-label="breadcrumb"
        class="mb-3 flex flex-wrap gap-x-1.5 font-mono text-mono text-slate-500 dark:text-slate-400"
      >
        <RouterLink :to="`/projects/${slug}`" class="text-indigo-700 dark:text-indigo-300">
          {{ slug }}
        </RouterLink>
        <template v-for="node in trail" :key="node.id">
          <span aria-hidden="true">›</span>
          <RouterLink
            :to="`/projects/${slug}/categories/${node.id}`"
            class="text-indigo-700 dark:text-indigo-300"
          >
            {{ node.name }}
          </RouterLink>
        </template>
      </nav>

      <h1 class="mb-2.5 text-display font-semibold">{{ kase.title }}</h1>

      <div
        class="mb-3 flex flex-wrap items-center gap-x-3 gap-y-1.5 font-mono text-mono text-slate-500 dark:text-slate-400"
      >
        <StatePill :state="kase.state as CaseState" />
        <span>#{{ kase.id }}</span>
        <template v-if="review.grid.value?.editionId">
          <span>·</span>
          <span>edition of {{ formatMoment(review.grid.value.takenAt) }}</span>
          <span
            v-if="review.lockedBy.value"
            class="inline-flex items-center gap-1.5 rounded-full border border-slate-300 bg-slate-50 px-2 py-0.5 dark:border-slate-600 dark:bg-slate-900"
          >
            <svg
              width="13"
              height="13"
              viewBox="0 0 16 16"
              fill="none"
              stroke="currentColor"
              stroke-width="1.3"
              aria-hidden="true"
            >
              <path d="M1.5 8s2.4-4.2 6.5-4.2S14.5 8 14.5 8s-2.4 4.2-6.5 4.2S1.5 8 1.5 8z" />
              <circle cx="8" cy="8" r="2" />
            </svg>
            held by {{ review.lockedBy.value.name }}
            <template v-if="review.lockedBy.value.since">
              · since {{ formatMoment(review.lockedBy.value.since) }}</template
            >
          </span>
          <template v-if="review.grid.value.revision">
            <span>·</span>
            <span>rev {{ review.grid.value.revision }}</span>
          </template>
        </template>
      </div>

      <!-- Over the page, not in it: the page stays mounted underneath with
           everything it holds, and the capture gets the window (#125). -->
      <CaptureCarousel
        v-if="(open || openRecording) && review.grid.value"
        :slug="slug"
        :grid="review.grid.value"
        :comments="review.comments.value"
        :step-id="open?.stepId ?? ''"
        :variant-id="open?.variantId ?? openRecording ?? ''"
        :recording="openRecording !== null"
        :held-by="review.lockedBy.value"
        :busy="review.saving.value"
        class="fixed inset-0 z-40"
        @close="closeCarousel"
        @move="moveTo"
        @move-recording="moveToRecording"
        @judge-recording="onJudgeRecording"
        @unjudge-recording="onUnjudgeRecording"
        @accept="onAccept"
        @unaccept="onUnaccept"
        @refuse="onRefuse"
        @unrefuse="onUnrefuse"
        @edit="onEdit"
        @judge="onJudge"
        @unjudge="onUnjudge"
      />

      <CaseGrid
        v-if="review.grid.value"
        :slug="slug"
        :grid="review.grid.value"
        :open-capture="open"
        @open="openCapture"
        @open-recording="openRecordingView"
      />

      <CommentRecap
        v-if="review.grid.value"
        :grid="review.grid.value"
        :comments="review.comments.value"
        @open="openCapture"
      />
    </template>
  </div>
</template>
