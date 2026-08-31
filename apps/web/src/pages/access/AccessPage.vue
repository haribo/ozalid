<script setup lang="ts">
/**
 * Who reaches one project.
 *
 * People and programs in one list, because they see the same thing (ADR 0019).
 * A deactivated account is absent — the server drops it, since this page
 * answers "who reaches this project" and a deactivated account reaches nothing
 * (product.md §8.2).
 */
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { AdminIcon, RightsPill } from '@/shared/ui'
import { formatMoment } from '@/shared/lib'
import { useAccess } from '@/features/access'
import { useAccounts } from '@/features/accounts'

const route = useRoute()
const slug = computed(() => String(route.params.slug))

const { members, error, busy, load, grant, revoke, retireProgram } = useAccess(() => slug.value)
const { accounts, load: loadAccounts } = useAccounts()

const adding = ref(false)
const chosen = ref('')

onMounted(() => {
  void load()
  void loadAccounts()
})
watch(slug, () => void load())

/** Only accounts that are not already on the project, and not retired. */
const addable = computed(() =>
  accounts.value.filter(
    (a) => !a.deactivatedAt && !members.value.some((m) => m.accountId === a.id),
  ),
)

async function add() {
  if (!chosen.value) return
  await grant(chosen.value, 'member')
  chosen.value = ''
  adding.value = false
}
</script>

<template>
  <main class="mx-auto max-w-5xl px-6 py-8">
    <nav class="mb-4 font-mono text-[11.5px] text-slate-500 dark:text-slate-400">
      <RouterLink to="/" class="text-indigo-700 dark:text-indigo-300">projets</RouterLink>
      › <b class="font-medium text-slate-900 dark:text-slate-100">{{ slug }}</b>
    </nav>

    <div class="mb-4 flex items-end gap-3">
      <h1 class="text-lg font-semibold">Accès</h1>
      <button
        type="button"
        class="ml-auto inline-flex items-center gap-1.5 rounded border border-indigo-600 bg-indigo-600 px-3 py-1.5 text-[12px] font-medium text-white focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-indigo-500 dark:border-indigo-500 dark:bg-indigo-500"
        @click="adding = !adding"
      >
        <AdminIcon name="add" :size="12" />Ajouter
      </button>
    </div>

    <form
      v-if="adding"
      class="mb-5 flex flex-wrap items-end gap-3 rounded-md border border-slate-200 p-4 dark:border-slate-700"
      @submit.prevent="add"
    >
      <label class="flex flex-col gap-1.5">
        <span class="font-mono text-[10px] tracking-wider text-slate-500 uppercase">compte</span>
        <select
          v-model="chosen"
          required
          class="rounded border border-slate-300 bg-white px-2.5 py-1.5 text-[13px] dark:border-slate-600 dark:bg-slate-900"
        >
          <option value="" disabled>choisir…</option>
          <option v-for="a in addable" :key="a.id" :value="a.id">{{ a.name }}</option>
        </select>
      </label>
      <button
        type="submit"
        :disabled="busy"
        class="rounded border border-indigo-600 bg-indigo-600 px-3 py-1.5 text-[12px] font-medium text-white disabled:opacity-60 dark:border-indigo-500 dark:bg-indigo-500"
      >
        Ajouter
      </button>
    </form>

    <p v-if="error" class="mb-3 font-mono text-[12px] text-red-700 dark:text-red-400">
      {{ error }}
    </p>

    <table class="w-full border-collapse text-[13px]">
      <thead>
        <tr class="border-b border-slate-300 dark:border-slate-600">
          <th
            class="py-1.5 pr-3 text-left font-mono text-[10px] tracking-wider text-slate-500 uppercase"
          >
            nom
          </th>
          <th />
          <th
            class="py-1.5 pr-3 text-left font-mono text-[10px] tracking-wider text-slate-500 uppercase"
          >
            droits
          </th>
          <th
            class="py-1.5 pr-3 text-left font-mono text-[10px] tracking-wider text-slate-500 uppercase"
          >
            depuis
          </th>
          <th />
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="m in members"
          :key="m.accountId"
          class="border-b border-slate-200 dark:border-slate-700"
        >
          <td class="py-2 pr-3 font-medium">
            <RouterLink
              v-if="!m.isPerson"
              :to="`/projects/${slug}/access/${m.accountId}`"
              class="text-indigo-700 dark:text-indigo-300"
            >
              {{ m.name }}
            </RouterLink>
            <template v-else>{{ m.name }}</template>
          </td>
          <td class="py-2 pr-3">
            <span
              class="inline-flex items-center gap-1.5 font-mono text-[10.5px] text-slate-500 dark:text-slate-400"
            >
              <!-- The icon carries it, and it names itself to a screen reader.
                   Spelling it out beside would be the same word twice. -->
              <AdminIcon :name="m.isPerson ? 'person' : 'program'" :size="12" />
              <template v-if="m.tokens === 0">sans jeton</template>
            </span>
          </td>
          <td class="py-2 pr-3"><RightsPill :rights="m.rights" /></td>
          <td class="py-2 pr-3 font-mono text-[12px] text-slate-500 dark:text-slate-400">
            {{ formatMoment(m.addedAt) }}
          </td>
          <td class="py-2 text-right whitespace-nowrap">
            <button
              v-if="m.isPerson"
              type="button"
              :disabled="busy"
              class="mr-1.5 inline-flex h-7 w-7 items-center justify-center rounded border border-slate-300 text-indigo-700 disabled:opacity-50 dark:border-slate-600 dark:text-indigo-300"
              @click="grant(m.accountId, m.rights === 'member' ? 'reader' : 'member')"
            >
              <AdminIcon
                name="swap"
                :label="`passer en ${m.rights === 'member' ? 'reader' : 'member'}`"
              />
            </button>
            <RouterLink
              v-else
              :to="`/projects/${slug}/access/${m.accountId}`"
              class="mr-1.5 inline-flex h-7 w-7 items-center justify-center rounded border border-slate-300 text-indigo-700 dark:border-slate-600 dark:text-indigo-300"
            >
              <AdminIcon name="keys" />
            </RouterLink>
            <button
              type="button"
              :disabled="busy"
              class="inline-flex h-7 w-7 items-center justify-center rounded border border-slate-300 text-indigo-700 disabled:opacity-50 dark:border-slate-600 dark:text-indigo-300"
              @click="m.isPerson ? revoke(m.accountId) : retireProgram(m.accountId)"
            >
              <AdminIcon
                name="remove"
                :label="m.isPerson ? 'retirer du projet' : 'retirer le programme'"
              />
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </main>
</template>
