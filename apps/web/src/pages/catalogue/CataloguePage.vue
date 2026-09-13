<script setup lang="ts">
/**
 * The catalogue at one depth: the same screen whether it lists sub-categories
 * or cases, so descending teaches nothing new.
 */
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, type components } from '@/shared/api'
import { TextField, EmptyState, AppButton, AdminIcon } from '@/shared/ui'
import { useSession } from '@/features/session'
import { useReview } from '@/features/review'
import { CategoryTable } from '@/widgets/category-table'
import { CaseTable } from '@/widgets/case-table'
import { ReviewCarousel, type Walk, type WalkSummary } from '@/widgets/capture-carousel'

type Category = components['schemas']['Category']
type Case = components['schemas']['Case']
type QueueEntry = components['schemas']['QueueEntry']

const route = useRoute()
const router = useRouter()
const { person } = useSession()
const slug = computed(() => String(route.params.slug))
const categoryId = computed(() =>
  route.params.categoryId ? String(route.params.categoryId) : null,
)

const categories = ref<Category[]>([])
const cases = ref<Case[]>([])
const error = ref('')
const loading = ref(true)

// Whether this person may build the tree. The server decides (`WriteProject`);
// this only decides what is offered, from what is already readable: an
// administrator writes everywhere (product.md §8.2), and a member's rights sit
// in the member list a reader may also read. No endpoint had to change (#116).
const mayWrite = ref(false)

const making = ref(false)
const newName = ref('')
const creating = ref(false)
const createError = ref('')

async function create() {
  creating.value = true
  createError.value = ''
  const result = await api.POST('/projects/{slug}/categories', {
    params: { path: { slug: slug.value } },
    // Appended after its siblings — the tree orders by position, then name.
    body: {
      name: newName.value,
      position: children.value.length,
      ...(categoryId.value ? { parentId: categoryId.value } : {}),
    },
  })
  creating.value = false
  if (result.error) {
    // The 409 arrives with its sentence — a sibling already carries that name —
    // and the form stays open with the words: there is one word to change.
    createError.value = result.error.title
    return
  }
  categories.value = [...categories.value, result.data]
  newName.value = ''
  making.value = false
}

/** The children of the node being looked at — the root when none is named. */
const children = computed(() =>
  categories.value.filter((c) => (c.parentId ?? null) === categoryId.value),
)

const trail = computed(() => {
  const out: Category[] = []
  let id = categoryId.value
  while (id) {
    const node = categories.value.find((c) => c.id === id)
    if (!node) break
    out.unshift(node)
    id = node.parentId ?? null
  }
  return out
})

/**
 * The queue under the category being read (product.md §3.6).
 *
 * Loaded once per depth and held for the length of the walk: recomputing it
 * after every verdict would move the list under the reviewer, and the walk
 * they started is the walk they finish (#205).
 */
const queue = ref<QueueEntry[]>([])

const movedCount = computed(() => queue.value.filter((e) => e.capture.status === 'moved').length)

/** Where the walk stands, said by the route: a capture worth judging is a
 * capture worth pointing at, in a walk as much as in a case (frontend
 * ADR 0005, ADR 0007). */
const walking = computed(() =>
  route.path.includes('/queue/') && route.params.caseId && route.params.stepId
    ? {
        caseId: String(route.params.caseId),
        stepId: String(route.params.stepId),
        variantId: String(route.params.variantId),
      }
    : null,
)

const queueUrl = computed(() =>
  categoryId.value
    ? `/projects/${slug.value}/categories/${categoryId.value}/queue`
    : `/projects/${slug.value}/queue`,
)

const scopeUrl = computed(() =>
  categoryId.value
    ? `/projects/${slug.value}/categories/${categoryId.value}`
    : `/projects/${slug.value}`,
)

function entryUrl(entry: QueueEntry) {
  return `${queueUrl.value}/cases/${entry.caseId}/steps/${entry.stepId}/variants/${entry.variant.id}`
}

function startWalk() {
  const first = queue.value[0]
  if (first) void router.push(entryUrl(first))
}

/** Arrow keys walk, they do not stack: replace, so back means the catalogue
 * rather than a retrace of every capture judged (frontend ADR 0005). */
function walkTo(entry: QueueEntry) {
  void router.replace(entryUrl(entry))
}

function moveInCase(stepId: string, variantId: string) {
  if (!walking.value) return
  void router.replace(
    `${queueUrl.value}/cases/${walking.value.caseId}/steps/${stepId}/variants/${variantId}`,
  )
}

/** Leaving the walk reads the queue back: what was judged during it has left,
 * and the catalogue says what is left rather than what was there on entry. */
async function leaveWalk() {
  summary.value = null
  await router.push(scopeUrl.value)
  await loadQueue()
}

/** The trail above the case being judged, trimmed of what the walk's own
 * scope already says: entered from Recovery, the banner shows what is below
 * Recovery and not the path back to it. */
function trailOf(entry: QueueEntry) {
  const names: string[] = []
  let id = entry.categoryId ?? null
  while (id && id !== categoryId.value) {
    const node = categories.value.find((c) => c.id === id)
    if (!node) break
    names.unshift(node.name)
    id = node.parentId ?? null
  }
  return names.join(' › ')
}

const review = useReview(
  () => slug.value,
  () => walking.value?.caseId ?? '',
)

/** What the reader calls where they are — the category on screen, or the
 * project. The end-of-walk screen speaks in those words (#255). */
const scopeName = computed(() => trail.value[trail.value.length - 1]?.name ?? slug.value)

/** The tally of the sitting, or null while it is still going. */
const summary = ref<WalkSummary | null>(null)

/**
 * What the walk came to, read back from the server (#255).
 *
 * The entries the walk started with are compared against the statuses those
 * captures now carry: what left the queue was judged, and the judgment says
 * where it went. Counting the keypresses instead would produce a number the
 * server never confirmed — and a reload mid-walk would make it lie.
 */
async function tally(): Promise<WalkSummary> {
  const touched = [...new Set(queue.value.map((e) => e.caseId))]
  const grids = await Promise.all(
    touched.map((caseId) =>
      api.GET('/projects/{slug}/cases/{caseId}/captures', {
        params: { path: { slug: slug.value, caseId } },
      }),
    ),
  )

  const now = new Map<string, string>()
  grids.forEach((grid) => {
    if (grid.error) return
    grid.data.steps.forEach((step) =>
      step.captures.forEach((capture) => now.set(capture.id, capture.status)),
    )
  })

  const out: WalkSummary = { accepted: 0, refused: 0, cases: 0, remaining: 0 }
  const cases = new Set<string>()
  queue.value.forEach((entry) => {
    // A capture the latest read no longer knows about cannot be counted
    // either way: it is gone from the edition on display, not judged.
    const status = now.get(entry.capture.id)
    if (status === 'accepted') {
      out.accepted++
      cases.add(entry.caseId)
    } else if (status === 'refused') {
      out.refused++
      cases.add(entry.caseId)
    } else if (status === 'to-review' || status === 'moved') {
      out.remaining++
    }
  })
  out.cases = cases.size
  return out
}

/** The last capture is judged: say so, with what the sitting came to. */
async function finishWalk() {
  summary.value = await tally()
}

/** From the end screen: leave this walk and read the project's whole queue. */
async function reviewProject() {
  summary.value = null
  await router.push(`/projects/${slug.value}`)
}

/** What the carousel walks: the entries, and who the reviewer is looking at
 * right now. */
const walk = computed<Walk | undefined>(() => {
  if (!walking.value) return undefined
  const here = queue.value.find(
    (e) => e.caseId === walking.value?.caseId && e.stepId === walking.value?.stepId,
  )
  return {
    entries: queue.value,
    caseId: walking.value.caseId,
    trail: here ? trailOf(here) : '',
    caseName: here?.caseTitle ?? '',
    scope: scopeName.value,
    summary: summary.value ?? undefined,
  }
})

// Entering a case is claiming it, leaving it is letting go — the walk crosses
// cases, so it does both as it goes (ADR 0005). A fresh claim stamps the hold
// onto what is current before the first read (ADR 0024), exactly as opening a
// case page does.
let held = ''
watch(
  () => walking.value?.caseId ?? '',
  async (now, before) => {
    if (before && before !== now) {
      await api.DELETE('/projects/{slug}/cases/{caseId}/lock', {
        params: { path: { slug: slug.value, caseId: before } },
      })
    }
    held = now
    if (!now) return
    await review.claim(true)
    await review.load()
  },
  { immediate: true },
)

const HEARTBEAT_MS = 30_000
let heartbeat: ReturnType<typeof setInterval> | undefined

/** Leaving is letting go, including by closing the tab where Vue never
 * unmounts: keepalive lets the release outlive the page (ADR 0005). */
function releaseOnLeave() {
  if (!held) return
  void fetch(`/api/projects/${slug.value}/cases/${held}/lock`, {
    method: 'DELETE',
    keepalive: true,
  })
}

onMounted(() => {
  heartbeat = setInterval(() => {
    if (held) void review.claim()
  }, HEARTBEAT_MS)
  window.addEventListener('pagehide', releaseOnLeave)
})

onBeforeUnmount(() => {
  clearInterval(heartbeat)
  window.removeEventListener('pagehide', releaseOnLeave)
  if (held) void review.release()
})

// watch, not watchEffect: dependencies read after an await are no longer
// tracked, so the route change would not re-run the effect and descending into
// a category would show nothing.
watch(
  [slug, categoryId],
  async ([currentSlug, currentCategory]) => {
    loading.value = true
    error.value = ''
    // The component is reused across levels, so an open form would survive
    // descending into a category — still naming the level it was opened at.
    making.value = false
    newName.value = ''
    createError.value = ''

    const tree = await api.GET('/projects/{slug}/categories', {
      params: { path: { slug: currentSlug } },
    })
    if (tree.error) {
      error.value = tree.error.title
      loading.value = false
      return
    }
    categories.value = tree.data

    if (person.value?.isAdmin) {
      mayWrite.value = true
    } else {
      const members = await api.GET('/projects/{slug}/members', {
        params: { path: { slug: currentSlug } },
      })
      mayWrite.value =
        !members.error &&
        members.data.some((m) => m.accountId === person.value?.id && m.rights === 'member')
    }

    // Cases are only asked for once a category is named: the root shows the
    // branches, not every case in the project.
    if (currentCategory) {
      const list = await api.GET('/projects/{slug}/cases', {
        params: { path: { slug: currentSlug }, query: { categoryId: currentCategory } },
      })
      cases.value = list.error ? [] : list.data
    } else {
      cases.value = []
    }

    await loadQueue()
    loading.value = false
  },
  { immediate: true },
)

/** What awaits the reviewer under the category on screen — the whole project
 * at the root (product.md §3.6). */
async function loadQueue() {
  const awaiting = await api.GET('/projects/{slug}/queue', {
    params: {
      path: { slug: slug.value },
      query: categoryId.value ? { categoryId: categoryId.value } : {},
    },
  })
  queue.value = awaiting.error ? [] : awaiting.data.entries
}
</script>

<template>
  <div class="mx-auto max-w-5xl px-6 py-8">
    <nav class="mb-4 font-mono text-mono text-slate-500 dark:text-slate-400">
      <RouterLink :to="`/projects/${slug}`" class="text-indigo-700 dark:text-indigo-300">
        {{ slug }}
      </RouterLink>
      <template v-for="node in trail" :key="node.id">
        ›
        <RouterLink
          :to="`/projects/${slug}/categories/${node.id}`"
          class="text-indigo-700 dark:text-indigo-300"
        >
          {{ node.name }}
        </RouterLink>
      </template>
    </nav>

    <div class="mb-4 flex items-end gap-3">
      <h1 class="text-title font-semibold">Catalogue</h1>
      <AppButton v-if="mayWrite" @click="making = !making">
        <AdminIcon name="add" :size="12" />New category
      </AppButton>
    </div>

    <!-- The parent is the category being read — the breadcrumb answers it, so
         the form asks one thing. The tree is of unrestricted depth, which makes
         this the same gesture at every level (#116). -->
    <form
      v-if="making"
      class="mb-5 flex flex-wrap items-end gap-3 rounded-md border border-slate-200 p-4 dark:border-slate-700"
      @submit.prevent="create"
    >
      <TextField v-model="newName" label="name" required :invalid="!!createError" />
      <AppButton :disabled="creating" type="submit"> Create </AppButton>
      <AppButton
        variant="secondary"
        @click=";((making = false), (createError = ''), (newName = ''))"
      >
        Cancel
      </AppButton>
      <p v-if="createError" class="w-full font-mono text-mono text-red-700 dark:text-red-400">
        {{ createError }}
      </p>
    </form>

    <p v-if="error" class="font-mono text-mono text-red-700 dark:text-red-400">
      {{ error }}
    </p>
    <p v-else-if="loading" class="font-mono text-mono text-slate-500">loading…</p>

    <template v-else>
      <!-- The count and the reach are one sentence: a separate scope label
           leaves the reader to reattach it, and the catalogue is one screen at
           every depth, so the number must say where it counts (#205,
           product.md §3.6). -->
      <div
        v-if="queue.length"
        class="mb-6 flex flex-wrap items-center justify-between gap-4 rounded-md border border-indigo-300 border-l-4 border-l-indigo-600 bg-indigo-50 px-4 py-3 dark:border-indigo-500 dark:border-l-indigo-400 dark:bg-indigo-950"
      >
        <div>
          <p class="font-semibold">
            <span class="text-indigo-700 dark:text-indigo-300">
              {{ queue.length }} capture{{ queue.length === 1 ? '' : 's' }}
            </span>
            {{ categoryId ? `in ${trail[trail.length - 1]?.name}` : `across ${slug}` }} need{{
              queue.length === 1 ? 's' : ''
            }}
            your verdict
          </p>
          <p class="font-mono text-mono text-slate-500 dark:text-slate-400">
            {{ queue.length - movedCount }} to-review · {{ movedCount }} moved
          </p>
        </div>
        <AppButton @click="startWalk">Review them</AppButton>
      </div>

      <CategoryTable v-if="children.length" :slug="slug" :categories="children" class="mb-6" />
      <CaseTable v-if="cases.length" :slug="slug" :cases="cases" />
      <EmptyState v-if="!children.length && !cases.length"> nothing here yet </EmptyState>
    </template>

    <!-- Over the catalogue, not instead of it: the page stays mounted with
         everything it holds, including a verdict a dead session refused
         (frontend ADR 0005, ADR 0007). -->
    <ReviewCarousel
      v-if="walking"
      :slug="slug"
      :review="review"
      :step-id="walking.stepId"
      :variant-id="walking.variantId"
      :walk="walk"
      class="fixed inset-0 z-40"
      @close="leaveWalk"
      @move="moveInCase"
      @move-entry="walkTo"
      @finish="finishWalk"
      @review-project="reviewProject"
    />
  </div>
</template>
