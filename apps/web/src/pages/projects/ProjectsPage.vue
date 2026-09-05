<script setup lang="ts">
/**
 * Where signing in lands.
 *
 * The name opens the project's catalogue; the icon opens who reaches it. An
 * administrator sees every project here and can open none of their books —
 * administering is not reading (product.md §8.2), and the server says so
 * whatever this screen offers.
 */
import { onMounted, ref } from 'vue'
import { AdminIcon } from '@/shared/ui'
import { useProjects } from '@/features/projects'
import { useSession } from '@/features/session'

const { projects, error, loading, load, create } = useProjects()
const { person } = useSession()

const making = ref(false)
const slug = ref('')
const name = ref('')

onMounted(load)

async function submit() {
  if (await create(slug.value, name.value)) {
    making.value = false
    slug.value = ''
    name.value = ''
  }
}
</script>

<template>
  <main class="mx-auto max-w-5xl px-6 py-8">
    <div class="mb-4 flex items-end gap-3">
      <h1 class="text-title font-semibold">Projects</h1>
      <button
        v-if="person?.isAdmin"
        type="button"
        class="ml-auto inline-flex items-center gap-1.5 rounded border border-indigo-600 bg-indigo-600 px-3 py-1.5 text-body font-medium text-white focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-indigo-500 dark:border-indigo-500 dark:bg-indigo-500"
        @click="making = !making"
      >
        <AdminIcon name="add" :size="12" />New project
      </button>
    </div>

    <form
      v-if="making"
      class="mb-5 grid gap-3 rounded-md border border-slate-200 p-4 sm:grid-cols-2 dark:border-slate-700"
      @submit.prevent="submit"
    >
      <label class="flex flex-col gap-1.5">
        <span class="font-mono text-label tracking-wider text-slate-500 uppercase">name</span>
        <input
          v-model="name"
          required
          class="rounded border border-slate-300 bg-white px-2.5 py-1.5 text-body dark:border-slate-600 dark:bg-slate-900"
        />
      </label>
      <label class="flex flex-col gap-1.5">
        <span class="font-mono text-label tracking-wider text-slate-500 uppercase">id</span>
        <input
          v-model="slug"
          required
          pattern="[a-z0-9][a-z0-9-]{1,62}"
          class="rounded border border-slate-300 bg-white px-2.5 py-1.5 font-mono text-mono dark:border-slate-600 dark:bg-slate-900"
        />
      </label>
      <button
        type="submit"
        class="justify-self-start rounded border border-indigo-600 bg-indigo-600 px-3 py-1.5 text-body font-medium text-white dark:border-indigo-500 dark:bg-indigo-500"
      >
        Create
      </button>
    </form>

    <p v-if="error" class="font-mono text-mono text-red-700 dark:text-red-400">{{ error }}</p>
    <p v-else-if="loading" class="font-mono text-mono text-slate-500">loading…</p>

    <table v-else-if="projects.length" class="w-full border-collapse text-body">
      <thead>
        <tr class="border-b border-slate-300 dark:border-slate-600">
          <th
            class="py-1.5 pr-3 text-left font-mono text-label tracking-wider text-slate-500 uppercase"
          >
            project
          </th>
          <th
            class="py-1.5 pr-3 text-left font-mono text-label tracking-wider text-slate-500 uppercase"
          >
            id
          </th>
          <th
            class="py-1.5 pr-3 text-left font-mono text-label tracking-wider text-slate-500 uppercase"
          >
            members
          </th>
          <th />
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="p in projects"
          :key="p.id"
          class="border-b border-slate-200 dark:border-slate-700"
        >
          <td class="py-2 pr-3">
            <RouterLink
              :to="`/projects/${p.slug}`"
              class="font-medium text-indigo-700 dark:text-indigo-300"
            >
              {{ p.name }}
            </RouterLink>
          </td>
          <td class="py-2 pr-3 font-mono text-mono text-slate-500 dark:text-slate-400">
            {{ p.slug }}
          </td>
          <td class="py-2 pr-3 font-mono text-mono text-slate-500 dark:text-slate-400">
            {{ p.people }} {{ p.people === 1 ? 'person' : 'people' }}
            <template v-if="p.programs">
              · {{ p.programs }} {{ p.programs === 1 ? 'program' : 'programs' }}
            </template>
          </td>
          <td class="py-2 text-right">
            <RouterLink
              v-if="person?.isAdmin"
              :to="`/projects/${p.slug}/access`"
              class="inline-flex h-7 w-7 items-center justify-center rounded border border-slate-300 text-indigo-700 dark:border-slate-600 dark:text-indigo-300"
            >
              <AdminIcon name="people" />
            </RouterLink>
          </td>
        </tr>
      </tbody>
    </table>

    <p
      v-else
      class="rounded-md border border-dashed border-slate-300 px-4 py-8 text-center text-slate-600 dark:border-slate-600 dark:text-slate-400"
    >
      <b class="mb-1 block text-slate-900 dark:text-slate-100"> You belong to no project. </b>
      Ask whoever runs the instance to add you.
    </p>
  </main>
</template>
