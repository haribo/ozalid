import { createRouter, createWebHistory } from 'vue-router'
import { CataloguePage } from '@/pages/catalogue'
import { CasePage } from '@/pages/case'

/**
 * Three routes, and the catalogue serves two of them: listing the root and
 * listing a category are the same screen at two depths.
 */
export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/projects/demo' },
    { path: '/projects/:slug', component: CataloguePage },
    { path: '/projects/:slug/categories/:categoryId', component: CataloguePage },
    { path: '/cases/:caseId', component: CasePage },
  ],
})
