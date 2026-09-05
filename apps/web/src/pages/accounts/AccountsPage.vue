<script setup lang="ts">
/**
 * Who exists on this instance.
 *
 * Deactivated accounts stay listed: a name in the journal has to resolve to
 * something (ADR 0018). The access screen of a project answers a different
 * question — who reaches it — and drops them.
 */
import { computed, onMounted, ref } from 'vue'
import { TextField, AppButton, AdminIcon } from '@/shared/ui'
import { formatMoment } from '@/shared/lib'
import { useAccounts } from '@/features/accounts'

const { accounts, error, busy, load, create, deactivate, setAdmin } = useAccounts()

const making = ref(false)
// Retired accounts are out of the way, not out of reach. They clutter the
// screen somebody onboards people on, and they are still what a name in the
// journal resolves to (ADR 0018) — so they are one click away rather than
// gone.
const showRetired = ref(false)
const shown = computed(() =>
  showRetired.value ? accounts.value : accounts.value.filter((a) => !a.deactivatedAt),
)
const retired = computed(() => accounts.value.filter((a) => a.deactivatedAt).length)
const name = ref('')
const email = ref('')
const admin = ref(false)

onMounted(load)

async function submit() {
  if (await create(name.value, email.value, admin.value)) {
    making.value = false
    name.value = ''
    email.value = ''
    admin.value = false
  }
}
</script>

<template>
  <main class="mx-auto max-w-5xl px-6 py-8">
    <div class="mb-4 flex items-end gap-3">
      <h1 class="text-title font-semibold">Accounts</h1>
      <AppButton @click="making = !making">
        <AdminIcon name="add" :size="12" />New account
      </AppButton>
    </div>

    <label
      v-if="retired"
      class="mb-3 inline-flex items-center gap-2 font-mono text-mono text-slate-500 dark:text-slate-400"
    >
      <!-- eslint-disable-next-line vue/no-restricted-html-elements -- a checkbox, not a text field (#155) -->
      <input v-model="showRetired" type="checkbox" class="accent-indigo-600" />
      show the {{ retired }} retired account{{ retired > 1 ? 's' : '' }}
    </label>

    <form
      v-if="making"
      class="mb-5 grid gap-3 rounded-md border border-slate-200 p-4 sm:grid-cols-3 dark:border-slate-700"
      @submit.prevent="submit"
    >
      <TextField v-model="name" label="name" required />
      <TextField v-model="email" label="address" type="email" required />
      <label class="flex items-center gap-2 self-end pb-1.5">
        <!-- eslint-disable-next-line vue/no-restricted-html-elements -- a checkbox, not a text field (#155) -->
        <input v-model="admin" type="checkbox" class="accent-indigo-600" />
        <span class="font-mono text-label tracking-wider text-slate-500 uppercase">
          administers the instance
        </span>
      </label>
      <AppButton :disabled="busy" type="submit"> Create the account </AppButton>
    </form>

    <p v-if="error" class="mb-3 font-mono text-mono text-red-700 dark:text-red-400">
      {{ error }}
    </p>

    <table class="w-full border-collapse text-body">
      <thead>
        <tr class="border-b border-slate-300 dark:border-slate-600">
          <th
            class="py-1.5 pr-3 text-left font-mono text-label tracking-wider text-slate-500 uppercase"
          >
            name
          </th>
          <th
            class="py-1.5 pr-3 text-left font-mono text-label tracking-wider text-slate-500 uppercase"
          >
            address
          </th>
          <th
            class="py-1.5 pr-3 text-left font-mono text-label tracking-wider text-slate-500 uppercase"
          >
            admin
          </th>
          <th
            class="py-1.5 pr-3 text-left font-mono text-label tracking-wider text-slate-500 uppercase"
          >
            since
          </th>
          <th />
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="a in shown"
          :key="a.id"
          class="border-b border-slate-200 dark:border-slate-700"
          :class="a.deactivatedAt ? 'text-slate-400 dark:text-slate-500' : ''"
        >
          <td class="py-2 pr-3 font-medium" :class="a.deactivatedAt ? 'line-through' : ''">
            {{ a.name }}
          </td>
          <td class="py-2 pr-3 font-mono text-mono">{{ a.email }}</td>
          <td class="py-2 pr-3">
            <span
              v-if="a.deactivatedAt"
              class="rounded border border-slate-300 bg-slate-100 px-1.5 py-0.5 font-mono text-mono whitespace-nowrap text-slate-500 dark:border-slate-600 dark:bg-slate-800"
            >
              retired on {{ formatMoment(a.deactivatedAt) }}
            </span>
            <span
              v-else-if="a.isAdmin"
              class="rounded border border-amber-500 bg-amber-50 px-1.5 py-0.5 font-mono text-label tracking-wide text-amber-800 uppercase dark:bg-amber-950 dark:text-amber-300"
            >
              admin
            </span>
          </td>
          <td class="py-2 pr-3 font-mono text-mono">{{ formatMoment(a.createdAt) }}</td>
          <td class="py-2 text-right whitespace-nowrap">
            <template v-if="!a.deactivatedAt">
              <AppButton
                :disabled="busy"
                variant="secondary"
                icon
                @click="setAdmin(a.id, !a.isAdmin)"
              >
                <AdminIcon
                  name="swap"
                  :label="a.isAdmin ? 'stop being an admin' : 'make an admin'"
                />
              </AppButton>
              <AppButton :disabled="busy" variant="secondary" icon @click="deactivate(a.id)">
                <AdminIcon name="remove" label="deactivate the account" />
              </AppButton>
            </template>
          </td>
        </tr>
      </tbody>
    </table>
  </main>
</template>
