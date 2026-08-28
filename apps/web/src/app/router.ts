import { createRouter, createWebHistory } from 'vue-router'
import { CataloguePage } from '@/pages/catalogue'
import { CasePage } from '@/pages/case'
import { SignInPage, ClaimPage } from '@/pages/sign-in'
import { useSession } from '@/features/session'

/**
 * Five routes, and two of them are the way in. The catalogue serves two of the
 * rest: listing the root and listing a category are the same screen at two
 * depths.
 */
export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/projects/demo' },
    { path: '/projects/:slug', component: CataloguePage },
    { path: '/projects/:slug/categories/:categoryId', component: CataloguePage },
    { path: '/cases/:caseId', component: CasePage },
    { path: '/sign-in', component: SignInPage, meta: { anonymous: true } },
    { path: '/sign-in/:link', component: ClaimPage, meta: { anonymous: true } },
  ],
})

/**
 * Nothing here is public, so the first navigation asks the server who is
 * calling before deciding what to show.
 *
 * An expired session is not sent back here: the page it happened on stays put
 * and offers to reconnect, because throwing somebody out of a review to a login
 * screen loses the verdict they were making.
 */
router.beforeEach(async (to) => {
  const { standing, refresh } = useSession()
  if (standing.value === 'unknown') await refresh()
  if (to.meta.anonymous) return true
  if (standing.value === 'out') return { path: '/sign-in', query: { redirect: to.fullPath } }
  return true
})
