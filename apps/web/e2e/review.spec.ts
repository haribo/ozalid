/**
 * A reviewer's sitting, through the interface, against a real server.
 *
 * Every test seeds its own case. Sharing one across a file makes each test
 * depend on what the previous one did, and the first failure then tells you
 * about the second — which is exactly what happened when this suite was
 * written that way.
 */
import { expect, test, type Page } from '@playwright/test'
import { commentOnStep, moveTheDarkVariant, seed, validateEverything } from './fixture'

/** The grid's own cells. The recap is another table, and its ticks are actions
 * rather than statuses — an assertion that spans both proves nothing about
 * either. */
const gridMarks = (page: Page) => page.locator('table').first().locator('tbody [role="img"]')

test('a case arrives with everything left to judge', async ({ page }) => {
  const seeded = await seed(page)
  await page.goto(`/cases/${seeded.caseId}`)

  await expect(
    page.getByRole('heading', { name: 'réinitialiser un mot de passe oublié' }),
  ).toBeVisible()
  await expect(page.getByText('to-review')).toBeVisible()

  // Six squares, none of them marked: nothing has been said, and a bare capture
  // is the only reading that leaves every pixel visible (frontend ADR 0003).
  await expect(page.locator('tbody button[aria-label*="dans le carrousel"]')).toHaveCount(6)
  await expect(gridMarks(page)).toHaveCount(0)
})

test('clicking a capture opens the carousel on that exact square', async ({ page }) => {
  const seeded = await seed(page)
  await page.goto(`/cases/${seeded.caseId}`)
  await page.locator('tbody button[aria-label*="dans le carrousel"]').nth(2).click()

  const carousel = page.locator('.overflow-hidden.rounded-lg').first()
  await expect(carousel).toContainText('ouvre le lien reçu par e-mail')
  await expect(carousel).toContainText('3 / 6')
})

test('space validates, and the server is what says so', async ({ page }) => {
  const seeded = await seed(page)
  await page.goto(`/cases/${seeded.caseId}`)
  await page.locator('tbody button[aria-label*="dans le carrousel"]').first().click()
  await page.keyboard.press(' ')

  await expect(page.locator('table').first().locator('[aria-label="validée"]')).toHaveCount(1)

  // Redrawn from what the server answered, never from a guess made in the
  // browser (ADR 0002) — so it survives a reload.
  await page.reload()
  await expect(page.locator('table').first().locator('[aria-label="validée"]')).toHaveCount(1)
})

test('one defect over two variants is one comment, not two', async ({ page }) => {
  const seeded = await seed(page)
  await page.goto(`/cases/${seeded.caseId}`)
  await page.locator('tbody button[aria-label*="dans le carrousel"]').nth(2).click()
  await page.getByRole('button', { name: 'commenter' }).click()

  await page.getByPlaceholder('ce que vous voyez, dans vos mots').fill('le bouton est coupé')
  await page.getByRole('button', { name: 'tout' }).click()
  await page.getByRole('button', { name: 'ajouter', exact: true }).click()

  // One row in the recap — the text also shows in the carousel, so the count is
  // taken where the claim is about.
  const recap = page.locator('table').last()
  await expect(recap.getByText('le bouton est coupé')).toHaveCount(1)
  await expect(page.locator('table').first().locator('[aria-label="commentée"]')).toHaveCount(2)
})

test('the recap takes you back to the capture a comment was written on', async ({ page }) => {
  const seeded = await seed(page)
  await commentOnStep(seeded, 1, 'le libellé induit en erreur')
  await page.goto(`/cases/${seeded.caseId}`)

  await page.getByRole('link', { name: 'ouvre le lien reçu par e-mail' }).click()

  const carousel = page.locator('.overflow-hidden.rounded-lg').first()
  await expect(carousel).toContainText('ouvre le lien reçu par e-mail')
  await expect(carousel).toContainText('le libellé induit en erreur')
})

test('a capture that moved comes back asking to be looked at', async ({ page }) => {
  const seeded = await seed(page)
  await validateEverything(seeded)
  await page.goto(`/cases/${seeded.caseId}`)
  await expect(page.locator('table').first().locator('[aria-label="validée"]')).toHaveCount(6)

  await moveTheDarkVariant(page, seeded)
  await page.reload()

  // Three dark squares moved, the light ones did not. The verdict they carried
  // is gone from the grid: for the question it asks, they are to judge again.
  await expect(page.locator('table').first().locator('[aria-label="a bougé"]')).toHaveCount(3)
  await expect(page.locator('table').first().locator('[aria-label="validée"]')).toHaveCount(3)
  await expect(page.getByText('3 captures ont bougé')).toBeVisible()

  // Freshness is an overlay, never a state: the case does not move.
  await expect(page.getByText('reviewed')).toBeVisible()
})

test('every mark the grid draws wears a disc', async ({ page }) => {
  // A status is a glyph on a disc (frontend ADR 0004). Checked against a real
  // render because that is where the rule was broken: a bare glyph landed on
  // the product's own button and became one of its pixels.
  const seeded = await seed(page)
  await validateEverything(seeded)
  await commentOnStep(seeded, 0, 'à revoir')
  await page.goto(`/cases/${seeded.caseId}`)

  await expect(gridMarks(page)).toHaveCount(6)
  for (const mark of await gridMarks(page).all()) {
    await expect(mark).toHaveClass(/rounded-full/)
  }
})
