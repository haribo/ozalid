/**
 * The session every other test runs inside.
 *
 * Signed in once, through the interface, exactly the way a person does — no
 * cookie written by hand. What the rest of the suite then exercises is a
 * browser carrying a session the product itself issued.
 */
import { expect, test as setup } from '@playwright/test'
import { emptyMailbox, linkSentTo } from './mailbox'
import { REVIEWER, SIGNED_IN } from './session'

setup('a reviewer signs in, once, for the whole suite', async ({ page, context }) => {
  await emptyMailbox(REVIEWER)
  await page.goto('/sign-in')
  await page.getByLabel('address').fill(REVIEWER)
  await page.getByRole('button', { name: 'Send the link' }).click()
  await expect(page.getByText('The link is on its way.')).toBeVisible()

  await page.goto(`/sign-in/${await linkSentTo(REVIEWER)}`)
  await expect(page.getByRole('button', { name: 'sign out' })).toBeVisible()

  await context.storageState({ path: SIGNED_IN })
})
