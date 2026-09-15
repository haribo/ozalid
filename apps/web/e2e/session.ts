/** The account the e2e instance is bootstrapped with, and where its session is kept. */
export const REVIEWER = 'e2e@ozalid.test'

/** Written by the setup project, read by every project that depends on it. */
export const SIGNED_IN = 'e2e/.auth/reviewer.json'

import { expect, type BrowserContext, type Page } from '@playwright/test'
import { emptyMailbox, linkSentTo } from './mailbox'

/** The session cookie a context carries, or '' when it carries none. */
async function sessionOf(context: BrowserContext) {
  const cookies = await context.cookies()
  return cookies.find((c) => c.name === 'ozalid_session')?.value ?? ''
}

/**
 * A browser for somebody else, starting from nothing.
 *
 * `browser.newContext()` applies the config's `use`, and that carries
 * `storageState: SIGNED_IN` — so a "fresh" context opens as the suite's
 * administrator. A second person must be a second person (#123).
 */
export async function freshContext(context: BrowserContext) {
  const fresh = await context.browser()!.newContext({
    storageState: { cookies: [], origins: [] },
  })
  // Asserted, not assumed: this is the leak itself, and it is only visible
  // here — once the second person signs in, their own cookie overwrites the
  // inherited one and nothing downstream can tell what it was. A bare
  // newContext() fails this line on every machine, which is what the CI-only
  // failure of #123 lacked.
  expect(await sessionOf(fresh), 'a fresh context inherited a session').toBe('')
  return fresh
}

/**
 * Signs a browser in through the interface, the way a person does.
 *
 * It returns when the session it just claimed is the one this browser carries,
 * not when the claim page has loaded: claiming happens in `onMounted`, after
 * `goto` resolves, and a request fired in between rides whatever cookie the
 * context already had — the administrator's, on a runner that lost the race
 * (#123). "Signed in" means landed, or this helper lies to every test that
 * calls it.
 */
export async function signIn(page: Page, email: string) {
  await emptyMailbox(email)
  await page.goto('/sign-in')
  await page.getByLabel('address').fill(email)
  await page.getByRole('button', { name: 'Send the link' }).click()
  await expect(page.getByText('The link is on its way.')).toBeVisible()
  await page.goto(`/sign-in/${await linkSentTo(email)}`)
  await expect(page.getByText('sign out')).toBeVisible()
}

/**
 * What #123 is about, made checkable on any machine: the second person asks as
 * themselves. Without it the leak is only visible when a runner happens to
 * lose the race, which is how it went unnoticed for months.
 */
export async function asksAsThemselves(mine: BrowserContext, theirs: BrowserContext) {
  const [admin, person] = [await sessionOf(mine), await sessionOf(theirs)]
  expect(person, 'the second person carries no session').not.toBe('')
  expect(person, "the second person is carrying the administrator's session").not.toBe(admin)
}
