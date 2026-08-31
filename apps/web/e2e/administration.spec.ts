/**
 * Onboarding somebody, through the screens rather than through curl.
 *
 * This is the test #87 exists for: an administrator makes an account, puts it
 * on a project, and that person signs in and reaches the book. Every step in a
 * browser, against a real server.
 */
import { expect, test, type Page } from '@playwright/test'
import { linkSentTo } from './mailbox'

const PROJECT = process.env.OZALID_E2E_PROJECT ?? 'e2e'

const unique = () => `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`

/** Signs a browser in through the interface, the way a person does. */
async function signIn(page: Page, email: string) {
  await page.goto('/sign-in')
  await page.getByLabel('adresse').fill(email)
  await page.getByRole('button', { name: 'Envoyer le lien' }).click()
  await expect(page.getByText('Le lien est parti.')).toBeVisible()
  await page.goto(`/sign-in/${await linkSentTo(email)}`)
}

test('the landing page shows the projects, not a hardcoded one', async ({ page }) => {
  // It redirected to `/projects/demo`, which exists on nobody's instance.
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Projets' })).toBeVisible()
  await expect(page.getByRole('link', { name: PROJECT, exact: true })).toBeVisible()

  // Who reaches each project, which is what one looks at before opening its
  // accesses. Deactivated accounts are not counted.
  const row = page.getByRole('row').filter({ hasText: PROJECT })
  await expect(row).toContainText(/\d+ personnes?/)
  await expect(row).toContainText(/\d+ programmes?/)
})

test('an administrator onboards somebody, who then reaches the project', async ({
  page,
  context,
}) => {
  const email = `hired-${unique()}@example.test`
  const name = `hired ${unique()}`

  // 1. the account
  await page.goto('/accounts')
  await page.getByRole('button', { name: 'Nouveau compte' }).click()
  await page.getByLabel('nom').fill(name)
  await page.getByLabel('adresse').fill(email)
  await page.getByRole('button', { name: 'Créer le compte' }).click()
  await expect(page.getByRole('cell', { name, exact: true })).toBeVisible()

  // 2. the membership
  await page.goto(`/projects/${PROJECT}/access`)
  await page.getByRole('button', { name: 'Ajouter' }).first().click()
  await page.getByLabel('compte').selectOption({ label: name })
  await page.getByRole('button', { name: 'Ajouter', exact: true }).last().click()
  await expect(page.getByRole('cell', { name, exact: true })).toBeVisible()

  // 3. the person signs in, in a browser of their own, and the project is there
  const theirs = await context.browser()!.newContext()
  const their = await theirs.newPage()
  await signIn(their, email)
  await expect(their.getByRole('link', { name: PROJECT, exact: true })).toBeVisible()
  await their.getByRole('link', { name: PROJECT, exact: true }).click()
  await expect(their).toHaveURL(new RegExp(`/projects/${PROJECT}$`))
  await theirs.close()
})

test('a person who administers nothing is not offered the accounts screen', async ({
  page,
  context,
}) => {
  const email = `plain-${unique()}@example.test`
  await page.goto('/accounts')
  await page.getByRole('button', { name: 'Nouveau compte' }).click()
  await page.getByLabel('nom').fill(`plain ${unique()}`)
  await page.getByLabel('adresse').fill(email)
  await page.getByRole('button', { name: 'Créer le compte' }).click()

  const theirs = await context.browser()!.newContext()
  const their = await theirs.newPage()
  await signIn(their, email)

  // The link is absent, and that is a convenience: the server refuses either
  // way, which is what the next line checks.
  await expect(their.getByRole('link', { name: 'comptes' })).toHaveCount(0)
  const refused = await their.request.get('/api/accounts')
  expect(refused.status()).toBe(403)
  await theirs.close()
})

test('a program is given a token through the screens, and it is shown once', async ({ page }) => {
  await page.goto(`/projects/${PROJECT}/access`)

  // The suite's own runner is listed as a program; open its tokens.
  await page.getByRole('link', { name: 'ses jetons' }).first().click()
  await expect(page.getByRole('heading', { name: 'Jetons' })).toBeVisible()

  const label = `drill ${unique()}`
  await page.getByLabel('étiquette').fill(label)
  await page.getByRole('button', { name: 'Frapper un jeton' }).click()

  const token = page.locator('code', { hasText: /^ozp_/ })
  await expect(token).toBeVisible()
  const value = await token.innerText()
  expect(value).toMatch(/^ozp_/)

  await page.getByRole('button', { name: "J'ai copié le jeton" }).click()
  await expect(token).toHaveCount(0)

  // Reloading does not bring it back: nothing stored it.
  await page.reload()
  await expect(page.getByRole('cell', { name: label })).toBeVisible()
  await expect(page.locator('code', { hasText: /^ozp_/ })).toHaveCount(0)
})
