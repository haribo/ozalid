<script setup lang="ts">
/**
 * The book's shell. Chrome stays quiet: this interface frames someone else's
 * screenshots, and nothing here may be mistaken for part of them.
 *
 * It listens for a sign-in that happens in another tab, because the link in the
 * mail is followed there and this tab may be the one holding an unsent verdict.
 */
import { onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { SessionExpiredBar, useSession } from '@/features/session'

const router = useRouter()
const { person, standing, signOut, listen } = useSession()

onUnmounted(listen())

async function leave() {
  await signOut()
  void router.push('/sign-in')
}
</script>

<template>
  <div class="min-h-dvh bg-white text-slate-900 dark:bg-slate-950 dark:text-slate-100">
    <header
      class="flex items-center gap-3 border-b border-slate-200 bg-slate-50 px-6 py-2.5 dark:border-slate-700 dark:bg-slate-900"
    >
      <RouterLink to="/" class="text-[15px] font-semibold"> ozalid </RouterLink>
      <nav v-if="person" class="flex gap-1 font-mono text-[11px]">
        <RouterLink
          to="/"
          class="rounded px-2 py-1 text-slate-500 dark:text-slate-400"
          active-class="bg-indigo-50 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300"
        >
          projets
        </RouterLink>
        <!-- Shown from `isAdmin`, which decides what is offered and never what
             is allowed: the server refuses regardless, and a hidden button has
             never protected anything. -->
        <RouterLink
          v-if="person.isAdmin"
          to="/accounts"
          class="rounded px-2 py-1 text-slate-500 dark:text-slate-400"
          active-class="bg-indigo-50 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300"
        >
          comptes
        </RouterLink>
      </nav>
      <span
        v-if="person"
        class="ml-auto flex items-center gap-2 font-mono text-[11px] text-slate-500 dark:text-slate-400"
      >
        <b class="font-medium text-slate-900 dark:text-slate-100">{{ person.name }}</b>
        ·
        <button type="button" class="underline underline-offset-2" @click="leave">
          se déconnecter
        </button>
      </span>
    </header>
    <SessionExpiredBar v-if="standing === 'expired'" />
    <RouterView />
  </div>
</template>
