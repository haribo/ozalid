/**
 * The session every other test runs inside.
 *
 * Signed in once, through the interface, exactly the way a person does — no
 * cookie written by hand. What the rest of the suite then exercises is a
 * browser carrying a session the product itself issued.
 */
import { expect, test as setup } from '@playwright/test'
import { linkSentTo } from './mailbox'
import { REVIEWER, SIGNED_IN } from './session'

setup('a reviewer signs in, once, for the whole suite', async ({ page, context }) => {
  await page.goto('/sign-in')
  await page.getByLabel('adresse').fill(REVIEWER)
  await page.getByRole('button', { name: 'Envoyer le lien' }).click()
  await expect(page.getByText('Le lien est parti.')).toBeVisible()

  await page.goto(`/sign-in/${await linkSentTo(REVIEWER)}`)
  await expect(page.getByRole('button', { name: 'se déconnecter' })).toBeVisible()

  await context.storageState({ path: SIGNED_IN })
})
