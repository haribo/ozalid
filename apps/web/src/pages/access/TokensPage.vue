<script setup lang="ts">
/**
 * One program's credentials.
 *
 * Minting adds a token and retires none: that is what makes a rotation
 * survivable rather than an outage. `lastUsedAt` is what finishes one — it says
 * which of the old tokens nothing presents any more.
 */
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { AdminIcon, MintedTokenPanel } from '@/shared/ui'
import { formatMoment } from '@/shared/lib'
import { useTokens } from '@/features/access'

const route = useRoute()
const slug = computed(() => String(route.params.slug))
const programId = computed(() => String(route.params.serviceAccountId))

const { tokens, minted, error, busy, load, mint, retire, dismiss } = useTokens(
  () => slug.value,
  () => programId.value,
)

const label = ref('')

onMounted(load)
watch([slug, programId], () => void load())

async function submit() {
  await mint(label.value)
  label.value = ''
}
</script>

<template>
  <main class="mx-auto max-w-5xl px-6 py-8">
    <nav class="mb-4 font-mono text-mono text-slate-500 dark:text-slate-400">
      <RouterLink to="/" class="text-indigo-700 dark:text-indigo-300">projects</RouterLink>
      ›
      <RouterLink :to="`/projects/${slug}/access`" class="text-indigo-700 dark:text-indigo-300">
        {{ slug }}
      </RouterLink>
      › <b class="font-medium text-slate-900 dark:text-slate-100">tokens</b>
    </nav>

    <!-- The one moment the token can be read. Only its hash is stored, so
         nothing here can give it back once this panel goes. -->
    <MintedTokenPanel v-if="minted" class="mb-6" :token="minted.token" @dismiss="dismiss" />

    <form class="mb-5 flex flex-wrap items-end gap-3" @submit.prevent="submit">
      <h1 class="mr-auto text-title font-semibold">Tokens</h1>
      <label class="flex flex-col gap-1.5">
        <span class="font-mono text-label tracking-wider text-slate-500 uppercase">label</span>
        <input
          v-model="label"
          required
          placeholder="github actions"
          class="rounded border border-slate-300 bg-white px-2.5 py-1.5 text-body dark:border-slate-600 dark:bg-slate-900"
        />
      </label>
      <button
        type="submit"
        :disabled="busy"
        class="inline-flex items-center gap-1.5 rounded border border-indigo-600 bg-indigo-600 px-3 py-1.5 text-body font-medium text-white disabled:opacity-60 dark:border-indigo-500 dark:bg-indigo-500"
      >
        <AdminIcon name="keys" :size="12" />Mint a token
      </button>
    </form>

    <p v-if="error" class="mb-3 font-mono text-mono text-red-700 dark:text-red-400">
      {{ error }}
    </p>

    <table v-if="tokens.length" class="w-full border-collapse text-body">
      <thead>
        <tr class="border-b border-slate-300 dark:border-slate-600">
          <th
            class="py-1.5 pr-3 text-left font-mono text-label tracking-wider text-slate-500 uppercase"
          >
            label
          </th>
          <th
            class="py-1.5 pr-3 text-left font-mono text-label tracking-wider text-slate-500 uppercase"
          >
            created
          </th>
          <th
            class="py-1.5 pr-3 text-left font-mono text-label tracking-wider text-slate-500 uppercase"
          >
            last used
          </th>
          <th />
        </tr>
      </thead>
      <tbody>
        <tr v-for="t in tokens" :key="t.id" class="border-b border-slate-200 dark:border-slate-700">
          <td class="py-2 pr-3 font-medium">{{ t.label }}</td>
          <td class="py-2 pr-3 font-mono text-mono text-slate-500 dark:text-slate-400">
            {{ formatMoment(t.createdAt) }}
          </td>
          <td
            class="py-2 pr-3 font-mono text-mono"
            :class="
              t.lastUsedAt
                ? 'text-slate-500 dark:text-slate-400'
                : 'text-slate-400 dark:text-slate-600'
            "
          >
            {{ t.lastUsedAt ? formatMoment(t.lastUsedAt) : 'never presented' }}
          </td>
          <td class="py-2 text-right">
            <button
              type="button"
              :disabled="busy"
              class="inline-flex h-7 w-7 items-center justify-center rounded border border-slate-300 text-indigo-700 disabled:opacity-50 dark:border-slate-600 dark:text-indigo-300"
              @click="retire(t.id)"
            >
              <AdminIcon name="remove" label="retire this token" />
            </button>
          </td>
        </tr>
      </tbody>
    </table>

    <p
      v-else
      class="rounded-md border border-dashed border-slate-300 px-4 py-6 text-center font-mono text-mono text-slate-500 dark:border-slate-600 dark:text-slate-400"
    >
      this program presents no token
    </p>
  </main>
</template>
