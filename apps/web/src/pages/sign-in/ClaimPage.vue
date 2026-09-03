<script setup lang="ts">
/**
 * The page the link in the mail opens.
 *
 * It spends the link and leaves. Expired, already spent and never issued are
 * one answer here, exactly as they are on the server: which of the three it was
 * is not something a visitor gets told.
 */
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSession } from '@/features/session'

const route = useRoute()
const router = useRouter()
const { claim } = useSession()

const dead = ref(false)

onMounted(async () => {
  if (await claim(String(route.params.link))) {
    void router.replace('/')
    return
  }
  dead.value = true
})
</script>

<template>
  <main class="mx-auto w-full max-w-sm px-6 py-10">
    <div
      v-if="dead"
      class="rounded border border-amber-500 bg-amber-50 p-3.5 text-[13px] leading-relaxed text-amber-800 dark:bg-amber-950 dark:text-amber-300"
    >
      <b class="block font-semibold">This link no longer works.</b>
      A link works once, and for fifteen minutes.
      <RouterLink to="/sign-in" class="mt-2 block underline underline-offset-2">
        Ask for a new link
      </RouterLink>
    </div>
    <p v-else class="text-[13px] text-slate-500 dark:text-slate-400">Signing in…</p>
  </main>
</template>
