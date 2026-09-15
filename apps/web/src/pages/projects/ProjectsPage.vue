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
import { TextField, EmptyState, AppButton, AdminIcon } from '@/shared/ui'
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
      <AppButton v-if="person?.isAdmin" @click="making = !making">
        <AdminIcon name="add" :size="12" />New project
      </AppButton>
    </div>

    <form
      v-if="making"
      class="mb-5 grid gap-3 rounded-md border border-slate-200 p-4 sm:grid-cols-2 dark:border-slate-700"
      @submit.prevent="submit"
    >
      <TextField v-model="name" label="name" required />
      <TextField v-model="slug" label="id" required />
      <AppButton type="submit"> Create </AppButton>
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

    <EmptyState v-else prose>
      <b class="mb-1 block text-slate-900 dark:text-slate-100"> You belong to no project. </b>
      Ask whoever runs the instance to add you.
    </EmptyState>
  </main>
</template>
