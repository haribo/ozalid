<script setup lang="ts">
/**
 * The catalogue at one depth: the same screen whether it lists sub-categories
 * or cases, so descending teaches nothing new.
 */
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api, type components } from '@/shared/api'
import { CategoryTable } from '@/widgets/category-table'
import { CaseTable } from '@/widgets/case-table'

type Category = components['schemas']['Category']
type Case = components['schemas']['Case']

const route = useRoute()
const slug = computed(() => String(route.params.slug))
const categoryId = computed(() =>
  route.params.categoryId ? String(route.params.categoryId) : null,
)

const categories = ref<Category[]>([])
const cases = ref<Case[]>([])
const error = ref('')
const loading = ref(true)

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

    const tree = await api.GET('/projects/{slug}/categories', {
      params: { path: { slug: currentSlug } },
    })
    if (tree.error) {
      error.value = tree.error.title
      loading.value = false
      return
    }
    categories.value = tree.data

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
    <nav class="mb-4 font-mono text-[11.5px] text-slate-500 dark:text-slate-400">
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

    <p v-if="error" class="font-mono text-[12px] text-red-700 dark:text-red-400">
      {{ error }}
    </p>
    <p v-else-if="loading" class="font-mono text-[12px] text-slate-500">chargement…</p>

    <template v-else>
      <CategoryTable v-if="children.length" :slug="slug" :categories="children" class="mb-6" />
      <CaseTable v-if="cases.length" :slug="slug" :cases="cases" />
      <p
        v-if="!children.length && !cases.length"
        class="rounded-md border border-dashed border-slate-300 px-4 py-6 text-center font-mono text-[12px] text-slate-500 dark:border-slate-600 dark:text-slate-400"
      >
        rien ici pour le moment
      </p>
    </template>
  </div>
</template>
