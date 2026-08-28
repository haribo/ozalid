<script setup lang="ts">
/**
 * Where a signed-out visitor lands, whatever they were reaching for.
 *
 * The link is followed in whatever tab the mail client opens; the shell hears
 * that and refreshes the session. This page only has to notice and carry on to
 * the page that was asked for.
 */
import { watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { SignInPanel, useSession } from '@/features/session'

const route = useRoute()
const router = useRouter()
const { standing } = useSession()

watch(standing, (now) => {
  if (now !== 'in') return
  const wanted = route.query.redirect
  void router.replace(typeof wanted === 'string' ? wanted : '/')
})
</script>

<template>
  <main class="px-6">
    <SignInPanel />
  </main>
</template>
