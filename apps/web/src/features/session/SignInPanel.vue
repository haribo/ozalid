<script setup lang="ts">
import { AppButton } from '@/shared/ui'
/**
 * The door. One field, one button, then one message.
 *
 * There is no password, so there is nothing to reset and no second form. What
 * would be a recovery procedure elsewhere is the ordinary way in here
 * (ADR 0019).
 *
 * The label starts inside the field, icon and all, and lands on the border once
 * something is typed — so naming the field never costs a line of height. Asked
 * for in the first real review of these screens (#129). The icon lives in the
 * label and follows it.
 */
import { ref } from 'vue'
import { TextField } from '@/shared/ui'
import { useSession } from './useSession'

const { linkSent, busy, error, requestLink, askAgain } = useSession()
const email = ref('')
</script>

<template>
  <div class="mx-auto w-full max-w-sm py-10">
    <!-- One green thing: the disc, the same language as the book's verdicts
         (frontend ADR 0004). The old state wore three at once (#130). -->
    <div v-if="linkSent" class="flex items-start gap-3">
      <span
        class="grid h-7 w-7 flex-none place-items-center rounded-full border-2 border-emerald-600 text-emerald-700 dark:border-emerald-500 dark:text-emerald-400"
        aria-hidden="true"
      >
        <svg
          width="13"
          height="13"
          viewBox="0 0 16 16"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="m2.8 8.6 3.4 3.4 7-7.4" />
        </svg>
      </span>
      <div class="text-body leading-relaxed">
        <b class="block font-semibold">The link is on its way.</b>
        <span class="text-slate-500 dark:text-slate-400">
          Nothing yet? Check the address, then your spam.
        </span>
        <AppButton variant="ghost" @click="askAgain"> Use another address </AppButton>
      </div>
    </div>

    <form v-else class="flex flex-col gap-3" @submit.prevent="requestLink(email)">
      <h1 class="text-title font-semibold">Sign in</h1>

      <TextField
        v-model="email"
        floating
        label="address"
        type="email"
        required
        autocomplete="email"
      >
        <template #icon="{ landed }">
          <svg
            :width="landed ? 12 : 17"
            :height="landed ? 12 : 17"
            viewBox="0 0 16 16"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            aria-hidden="true"
          >
            <rect x="1.8" y="3.4" width="12.4" height="9.2" rx="1.4" />
            <path d="m2.4 4.4 5.6 4.4 5.6-4.4" />
          </svg>
        </template>
      </TextField>

      <p v-if="error" class="text-body text-rose-700 dark:text-rose-400">{{ error }}</p>

      <AppButton :disabled="busy" type="submit"> Send the link </AppButton>
    </form>
  </div>
</template>
