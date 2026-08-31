/**
 * Signing in, walked end to end: a link is asked for, a message arrives, the
 * link is spent, and a session exists.
 *
 * This is the one file that starts signed out. Every other test in the suite
 * inherits the session the setup project opened, which is why the state is
 * cleared here by hand.
 */
import { expect, test } from '@playwright/test'
import { emptyMailbox, linkSentTo } from './mailbox'
import { seed } from './fixture'
import { REVIEWER } from './session'

test.use({ storageState: { cookies: [], origins: [] } })

const API = process.env.OZALID_API ?? 'http://localhost:8091'

async function ask(email: string) {
  return fetch(`${API}/api/sign-in`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ email }),
  })
}

test('a person asks, reads their mail, and is signed in', async ({ request }) => {
  const email = REVIEWER
  await emptyMailbox(email)

  const asked = await ask(email)
  expect(asked.status).toBe(202)

  const link = await linkSentTo(email)

  const claimed = await request.post(`${API}/api/sign-in/claim`, { data: { link } })
  expect(claimed.status()).toBe(204)

  // The session rides in a cookie the browser holds, so the same context now
  // answers as somebody.
  const me = await request.get(`${API}/api/me`)
  expect(me.status()).toBe(200)
  expect((await me.json()).email).toBe(email)

  // Spent. The second attempt is refused the same way an invented link is.
  const again = await request.post(`${API}/api/sign-in/claim`, { data: { link } })
  expect(again.status()).toBe(401)

  await request.post(`${API}/api/sign-out`)
  expect((await request.get(`${API}/api/me`)).status()).toBe(401)
})

test('an address nobody has gets the same answer as one that exists', async () => {
  // Whether somebody has an account here is not something a stranger learns by
  // asking, so the two answers have to be identical.
  const stranger = await ask(`nobody-${Date.now()}@ozalid.test`)
  expect(stranger.status).toBe(202)
})

test('a link nobody issued is refused', async ({ request }) => {
  const claimed = await request.post(`${API}/api/sign-in/claim`, { data: { link: 'never-issued' } })
  expect(claimed.status()).toBe(401)
})

test('a signed-out visitor reaching a case is offered the way in', async ({ page }) => {
  await page.goto('/projects/e2e/cases/00000000-0000-0000-0000-000000000000')

  await expect(page.getByRole('heading', { name: 'Se connecter' })).toBeVisible()
})

test('the link opens in another tab, and the waiting one carries on', async ({ page, context }) => {
  // The mail client opens its own tab, so the tab that was refused never sees
  // the link. It is told, and resumes — without this the reviewer is left
  // looking at a sign-in form next to a session that works.
  const seeded = await seed(page)
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)
  await expect(page.getByRole('heading', { name: 'Se connecter' })).toBeVisible()

  await emptyMailbox(REVIEWER)
  await page.getByLabel('adresse').fill(REVIEWER)
  await page.getByRole('button', { name: 'Envoyer le lien' }).click()
  await expect(page.getByText('Le lien est parti.')).toBeVisible()

  const other = await context.newPage()
  await other.goto(`/sign-in/${await linkSentTo(REVIEWER)}`)

  await expect(
    page.getByRole('heading', { name: 'réinitialiser un mot de passe oublié' }),
  ).toBeVisible()
})
