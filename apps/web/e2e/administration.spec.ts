/**
 * Onboarding somebody, through the screens rather than through curl.
 *
 * This is the test #87 exists for: an administrator makes an account, puts it
 * on a project, and that person signs in and reaches the book. Every step in a
 * browser, against a real server.
 */
import { expect, test, type Page } from '@playwright/test'
import { emptyMailbox, linkSentTo } from './mailbox'

const PROJECT = process.env.OZALID_E2E_PROJECT ?? 'e2e'

const unique = () => `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`

/** Signs a browser in through the interface, the way a person does. */
async function signIn(page: Page, email: string) {
  await emptyMailbox(email)
  await page.goto('/sign-in')
  await page.getByLabel('address').fill(email)
  await page.getByRole('button', { name: 'Send the link' }).click()
  await expect(page.getByText('The link is on its way.')).toBeVisible()
  await page.goto(`/sign-in/${await linkSentTo(email)}`)
}

test('the landing page shows the projects, not a hardcoded one', async ({ page }) => {
  // It redirected to `/projects/demo`, which exists on nobody's instance.
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Projects' })).toBeVisible()
  await expect(page.getByRole('link', { name: PROJECT, exact: true })).toBeVisible()

  // Who reaches each project, which is what one looks at before opening its
  // accesses. Deactivated accounts are not counted.
  const row = page.getByRole('row').filter({ hasText: PROJECT })
  await expect(row).toContainText(/\d+ (person|people)/)
  await expect(row).toContainText(/\d+ programs?/)
})

test('an administrator onboards somebody, who then reaches the project', async ({
  page,
  context,
}) => {
  const email = `hired-${unique()}@example.test`
  const name = `hired ${unique()}`

  // 1. the account
  await page.goto('/accounts')
  await page.getByRole('button', { name: 'New account' }).click()
  await page.getByLabel('name').fill(name)
  await page.getByLabel('address').fill(email)
  await page.getByRole('button', { name: 'Create the account' }).click()
  await expect(page.getByRole('cell', { name, exact: true })).toBeVisible()

  // 2. the membership
  await page.goto(`/projects/${PROJECT}/access`)
  await page.getByRole('button', { name: 'Add' }).first().click()
  await page.getByLabel('account').selectOption({ label: name })
  await page.getByRole('button', { name: 'Add', exact: true }).last().click()
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
  await page.getByRole('button', { name: 'New account' }).click()
  await page.getByLabel('name').fill(`plain ${unique()}`)
  await page.getByLabel('address').fill(email)
  await page.getByRole('button', { name: 'Create the account' }).click()

  const theirs = await context.browser()!.newContext()
  const their = await theirs.newPage()
  await signIn(their, email)

  // The link is absent, and that is a convenience: the server refuses either
  // way, which is what the next line checks.
  await expect(their.getByRole('link', { name: 'accounts' })).toHaveCount(0)
  const refused = await their.request.get('/api/accounts')
  expect(refused.status()).toBe(403)
  await theirs.close()
})

test('a program is given a token through the screens, and it is shown once', async ({ page }) => {
  await page.goto(`/projects/${PROJECT}/access`)

  // The suite's own runner is listed as a program; open its tokens.
  await page.getByRole('link', { name: 'its tokens' }).first().click()
  await expect(page.getByRole('heading', { name: 'Tokens' })).toBeVisible()

  const label = `drill ${unique()}`
  await page.getByLabel('label').fill(label)
  await page.getByRole('button', { name: 'Mint a token' }).click()

  const token = page.locator('code', { hasText: /^ozp_/ })
  await expect(token).toBeVisible()
  const value = await token.innerText()
  expect(value).toMatch(/^ozp_/)

  await page.getByRole('button', { name: 'I have copied the token' }).click()
  await expect(token).toHaveCount(0)

  // Reloading does not bring it back: nothing stored it.
  await page.reload()
  await expect(page.getByRole('cell', { name: label })).toBeVisible()
  await expect(page.locator('code', { hasText: /^ozp_/ })).toHaveCount(0)
})

test('a program is created from the access screen, and its token is shown there', async ({
  page,
}) => {
  // Before this, no screen called POST /projects/{slug}/service-accounts: a
  // program was added from a browser console or not at all (#110).
  const slug = `programs-${unique()}`
  await page.request.post(`/api/projects`, { data: { slug, name: 'a project needing a runner' } })

  await page.goto(`/projects/${slug}/access`)
  await page.getByRole('button', { name: 'Add' }).first().click()
  await page.getByRole('button', { name: 'A program' }).click()

  const name = `runner-${unique()}`
  await page.getByLabel('name').fill(name)
  await page.getByLabel('rights').selectOption('reader')
  await page.getByRole('button', { name: 'Create' }).click()

  // The token, in place. Sending somebody to another screen for it would send
  // them somewhere the token no longer exists.
  const token = page.locator('code', { hasText: /^ozp_/ })
  await expect(token).toBeVisible()
  const value = await token.innerText()

  const row = page.getByRole('row').filter({ hasText: name })
  await expect(row).toContainText('reader')

  // And it opens the project it was made for, which is the only claim that
  // matters about a credential.
  const opened = await fetch(
    `${process.env.OZALID_API ?? 'http://localhost:8091'}/api/projects/${slug}/cases`,
    { headers: { authorization: `Bearer ${value}` } },
  )
  expect(opened.status).toBe(200)

  await page.getByRole('button', { name: 'I have copied the token' }).click()
  await page.reload()
  await expect(page.locator('code', { hasText: /^ozp_/ })).toHaveCount(0)
  await expect(page.getByRole('row').filter({ hasText: name })).toBeVisible()

  // The form no longer asks what the first token is for: at that moment there
  // is one token and its purpose is the program just named (#113). The server
  // still requires a label, so the client sends the name — and this is where
  // that label surfaces. Checked last, because leaving the page loses the
  // token, which is the behaviour the lines above assert.
  await page
    .getByRole('row')
    .filter({ hasText: name })
    .getByRole('link', { name: 'its tokens' })
    .click()
  await expect(page.getByRole('cell', { name, exact: true })).toBeVisible()
})

test('a person is added as a reader without passing through member', async ({ page }) => {
  // The form granted `member` in hard code, so somebody meant to read was
  // added then demoted — and could write in between (#110).
  const slug = `readers-${unique()}`
  await page.request.post(`/api/projects`, { data: { slug, name: 'a project with a reader' } })
  const made = await page.request.post(`/api/accounts`, {
    data: { name: `reader ${unique()}`, email: `reader-${unique()}@example.test` },
  })
  const account = await made.json()

  await page.goto(`/projects/${slug}/access`)
  await page.getByRole('button', { name: 'Add' }).first().click()
  await page.getByLabel('account').selectOption(account.id)
  await page.getByLabel('rights').selectOption('reader')
  await page.getByRole('button', { name: 'Add' }).last().click()

  await expect(page.getByRole('row').filter({ hasText: account.name })).toContainText('reader')

  // Nothing recorded a passage through `member`: the membership was written
  // once, with the rights that were asked for.
  const members = await (await page.request.get(`/api/projects/${slug}/members`)).json()
  expect(members).toHaveLength(1)
  expect(members[0].rights).toBe('reader')
})

test('an access list with nobody on it says so, rather than heading nothing', async ({ page }) => {
  // A fresh project rendered three column heads over an empty body, which reads
  // as a screen still loading — and loading and empty call for opposite
  // gestures (#112).
  const slug = `alone-${unique()}`
  await page.request.post(`/api/projects`, { data: { slug, name: 'a project nobody is on' } })

  await page.goto(`/projects/${slug}/access`)
  await expect(page.getByText('nobody reaches this project')).toBeVisible()
  await expect(page.locator('table')).toHaveCount(0)

  // One member is enough to bring the table back, heads and all.
  const accounts = await (await page.request.get(`/api/accounts`)).json()
  const someone = accounts.find((a: { deactivatedAt?: string }) => !a.deactivatedAt)
  await page.request.put(`/api/projects/${slug}/members/${someone.id}`, {
    data: { rights: 'reader' },
  })
  await page.reload()
  await expect(page.getByText('nobody reaches this project')).toHaveCount(0)
  await expect(page.getByRole('row').filter({ hasText: someone.name })).toBeVisible()
})
