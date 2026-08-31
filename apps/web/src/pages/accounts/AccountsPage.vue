<script setup lang="ts">
/**
 * Who exists on this instance.
 *
 * Deactivated accounts stay listed: a name in the journal has to resolve to
 * something (ADR 0018). The access screen of a project answers a different
 * question — who reaches it — and drops them.
 */
import { onMounted, ref } from 'vue'
import { AdminIcon } from '@/shared/ui'
import { formatMoment } from '@/shared/lib'
import { useAccounts } from '@/features/accounts'

const { accounts, error, busy, load, create, deactivate } = useAccounts()

const making = ref(false)
const name = ref('')
const email = ref('')
const admin = ref('relecteur')

onMounted(load)

async function submit() {
  if (await create(name.value, email.value, admin.value === 'admin')) {
    making.value = false
    name.value = ''
    email.value = ''
    admin.value = 'relecteur'
  }
}
</script>

<template>
  <main class="mx-auto max-w-5xl px-6 py-8">
    <div class="mb-4 flex items-end gap-3">
      <h1 class="text-lg font-semibold">Comptes</h1>
      <button
        type="button"
        class="ml-auto inline-flex items-center gap-1.5 rounded border border-indigo-600 bg-indigo-600 px-3 py-1.5 text-[12px] font-medium text-white focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-indigo-500 dark:border-indigo-500 dark:bg-indigo-500"
        @click="making = !making"
      >
        <AdminIcon name="add" :size="12" />Nouveau compte
      </button>
    </div>

    <form
      v-if="making"
      class="mb-5 grid gap-3 rounded-md border border-slate-200 p-4 sm:grid-cols-3 dark:border-slate-700"
      @submit.prevent="submit"
    >
      <label class="flex flex-col gap-1.5">
        <span class="font-mono text-[10px] tracking-wider text-slate-500 uppercase">nom</span>
        <input
          v-model="name"
          required
          class="rounded border border-slate-300 bg-white px-2.5 py-1.5 text-[13px] dark:border-slate-600 dark:bg-slate-900"
        />
      </label>
      <label class="flex flex-col gap-1.5">
        <span class="font-mono text-[10px] tracking-wider text-slate-500 uppercase">adresse</span>
        <input
          v-model="email"
          type="email"
          required
          class="rounded border border-slate-300 bg-white px-2.5 py-1.5 text-[13px] dark:border-slate-600 dark:bg-slate-900"
        />
      </label>
      <label class="flex flex-col gap-1.5">
        <span class="font-mono text-[10px] tracking-wider text-slate-500 uppercase">rôle</span>
        <select
          v-model="admin"
          class="rounded border border-slate-300 bg-white px-2.5 py-1.5 text-[13px] dark:border-slate-600 dark:bg-slate-900"
        >
          <option value="relecteur">relecteur</option>
          <option value="admin">admin</option>
        </select>
      </label>
      <button
        type="submit"
        :disabled="busy"
        class="justify-self-start rounded border border-indigo-600 bg-indigo-600 px-3 py-1.5 text-[12px] font-medium text-white disabled:opacity-60 dark:border-indigo-500 dark:bg-indigo-500"
      >
        Créer le compte
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
          <th
            class="py-1.5 pr-3 text-left font-mono text-[10px] tracking-wider text-slate-500 uppercase"
          >
            adresse
          </th>
          <th
            class="py-1.5 pr-3 text-left font-mono text-[10px] tracking-wider text-slate-500 uppercase"
          >
            rôle
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
          v-for="a in accounts"
          :key="a.id"
          class="border-b border-slate-200 dark:border-slate-700"
          :class="a.deactivatedAt ? 'text-slate-400 dark:text-slate-500' : ''"
        >
          <td class="py-2 pr-3 font-medium" :class="a.deactivatedAt ? 'line-through' : ''">
            {{ a.name }}
          </td>
          <td class="py-2 pr-3 font-mono text-[12px]">{{ a.email }}</td>
          <td class="py-2 pr-3">
            <span
              v-if="a.deactivatedAt"
              class="rounded border border-slate-300 bg-slate-100 px-1.5 py-0.5 font-mono text-[10px] whitespace-nowrap text-slate-500 dark:border-slate-600 dark:bg-slate-800"
            >
              retiré le {{ formatMoment(a.deactivatedAt) }}
            </span>
            <span
              v-else-if="a.isAdmin"
              class="rounded border border-amber-500 bg-amber-50 px-1.5 py-0.5 font-mono text-[10px] tracking-wide text-amber-800 uppercase dark:bg-amber-950 dark:text-amber-300"
            >
              admin
            </span>
          </td>
          <td class="py-2 pr-3 font-mono text-[12px]">{{ formatMoment(a.createdAt) }}</td>
          <td class="py-2 text-right">
            <button
              v-if="!a.deactivatedAt"
              type="button"
              :disabled="busy"
              class="inline-flex h-7 w-7 items-center justify-center rounded border border-slate-300 text-indigo-700 disabled:opacity-50 dark:border-slate-600 dark:text-indigo-300"
              @click="deactivate(a.id)"
            >
              <AdminIcon name="remove" label="désactiver le compte" />
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </main>
</template>
