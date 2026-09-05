<script setup lang="ts">
/**
 * The catalogue at one depth: the same screen whether it lists sub-categories
 * or cases, so descending teaches nothing new.
 */
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api, type components } from '@/shared/api'
import { AdminIcon } from '@/shared/ui'
import { useSession } from '@/features/session'
import { CategoryTable } from '@/widgets/category-table'
import { CaseTable } from '@/widgets/case-table'

type Category = components['schemas']['Category']
type Case = components['schemas']['Case']

const route = useRoute()
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
    loading.value = false
  },
  { immediate: true },
)
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
      <button
        v-if="mayWrite"
        type="button"
        class="ml-auto inline-flex items-center gap-1.5 rounded border border-indigo-600 bg-indigo-600 px-3 py-1.5 text-body font-medium text-white focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-indigo-500 dark:border-indigo-500 dark:bg-indigo-500"
        @click="making = !making"
      >
        <AdminIcon name="add" :size="12" />New category
      </button>
    </div>

    <!-- The parent is the category being read — the breadcrumb answers it, so
         the form asks one thing. The tree is of unrestricted depth, which makes
         this the same gesture at every level (#116). -->
    <form
      v-if="making"
      class="mb-5 flex flex-wrap items-end gap-3 rounded-md border border-slate-200 p-4 dark:border-slate-700"
      @submit.prevent="create"
    >
      <label class="flex flex-col gap-1.5">
        <span class="font-mono text-label tracking-wider text-slate-500 uppercase">name</span>
        <input
          v-model="newName"
          required
          class="rounded border bg-white px-2.5 py-1.5 text-body dark:bg-slate-900"
          :class="
            createError
              ? 'border-red-600 dark:border-red-500'
              : 'border-slate-300 dark:border-slate-600'
          "
        />
      </label>
      <button
        type="submit"
        :disabled="creating"
        class="rounded border border-indigo-600 bg-indigo-600 px-3 py-1.5 text-body font-medium text-white disabled:opacity-60 dark:border-indigo-500 dark:bg-indigo-500"
      >
        Create
      </button>
      <button
        type="button"
        class="rounded border border-slate-300 px-3 py-1.5 text-body text-slate-600 dark:border-slate-600 dark:text-slate-400"
        @click=";((making = false), (createError = ''), (newName = ''))"
      >
        Cancel
      </button>
      <p v-if="createError" class="w-full font-mono text-mono text-red-700 dark:text-red-400">
        {{ createError }}
      </p>
    </form>

    <p v-if="error" class="font-mono text-mono text-red-700 dark:text-red-400">
      {{ error }}
    </p>
    <p v-else-if="loading" class="font-mono text-mono text-slate-500">loading…</p>

    <template v-else>
      <CategoryTable v-if="children.length" :slug="slug" :categories="children" class="mb-6" />
      <CaseTable v-if="cases.length" :slug="slug" :cases="cases" />
      <p
        v-if="!children.length && !cases.length"
        class="rounded-md border border-dashed border-slate-300 px-4 py-6 text-center font-mono text-mono text-slate-500 dark:border-slate-600 dark:text-slate-400"
      >
        nothing here yet
      </p>
    </template>
  </div>
</template>
