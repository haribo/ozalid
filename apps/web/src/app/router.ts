import { createRouter, createWebHistory } from 'vue-router'
import { AccountsPage } from '@/pages/accounts'
import { AccessPage, TokensPage } from '@/pages/access'
import { CataloguePage } from '@/pages/catalogue'
import { ProjectsPage } from '@/pages/projects'
import { CasePage } from '@/pages/case'
import { SignInPage, ClaimPage } from '@/pages/sign-in'
import { useSession } from '@/features/session'

/**
 * Two of these are the way in, and the rest are the book and its administration.
 * The catalogue serves two: listing the root and listing a category are the same
 * screen at two depths.
 */
export const router = createRouter({
  history: createWebHistory(),
  routes: [
    // What a person lands on. It used to redirect to a hardcoded `demo`
    // project, which exists on nobody's instance.
    { path: '/', component: ProjectsPage },
    { path: '/accounts', component: AccountsPage },
    { path: '/projects/:slug/access', component: AccessPage },
    { path: '/projects/:slug/access/:serviceAccountId', component: TokensPage },
    { path: '/projects/:slug', component: CataloguePage },
    { path: '/projects/:slug/categories/:categoryId', component: CataloguePage },
    { path: '/projects/:slug/cases/:caseId', component: CasePage },
    // The carousel: the same component, so the instance — and the verdict it
    // may be holding through an expired session (#70) — survives opening and
    // closing a capture. A capture worth judging is a capture worth pointing
    // at, which is why this is an address and not a component state (#125).
    {
      path: '/projects/:slug/cases/:caseId/steps/:stepId/variants/:variantId',
      component: CasePage,
    },
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
