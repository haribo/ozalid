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
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)

  await expect(page.getByRole('heading', { name: 'reset a forgotten password' })).toBeVisible()
  await expect(page.getByText('to-review')).toBeVisible()

  // Six squares, none of them marked: nothing has been said, and a bare capture
  // is the only reading that leaves every pixel visible (frontend ADR 0003).
  await expect(page.locator('tbody button[aria-label*="in the carousel"]')).toHaveCount(6)
  await expect(gridMarks(page)).toHaveCount(0)
})

test('clicking a capture opens the carousel on that exact square', async ({ page }) => {
  const seeded = await seed(page)
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)
  await page.locator('tbody button[aria-label*="in the carousel"]').nth(2).click()

  const carousel = page.locator('.overflow-hidden.rounded-lg').first()
  await expect(carousel).toContainText('opens the link received by e-mail')
  await expect(carousel).toContainText('3 / 6')
})

test('space validates, and the server is what says so', async ({ page }) => {
  const seeded = await seed(page)
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)
  await page.locator('tbody button[aria-label*="in the carousel"]').first().click()
  await page.keyboard.press(' ')

  await expect(page.locator('table').first().locator('[aria-label="validated"]')).toHaveCount(1)

  // Redrawn from what the server answered, never from a guess made in the
  // browser (ADR 0002) — so it survives a reload.
  await page.reload()
  await expect(page.locator('table').first().locator('[aria-label="validated"]')).toHaveCount(1)
})

test('one defect over two variants is one comment, not two', async ({ page }) => {
  const seeded = await seed(page)
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)
  await page.locator('tbody button[aria-label*="in the carousel"]').nth(2).click()
  await page.getByRole('button', { name: 'comment' }).click()

  await page.getByPlaceholder('what you see, in your words').fill('the button is clipped')
  await page.getByRole('button', { name: 'all' }).click()
  await page.getByRole('button', { name: 'add', exact: true }).click()

  // One row in the recap — the text also shows in the carousel, so the count is
  // taken where the claim is about.
  const recap = page.locator('table').last()
  await expect(recap.getByText('the button is clipped')).toHaveCount(1)
  await expect(page.locator('table').first().locator('[aria-label="commented"]')).toHaveCount(2)
})

test('the recap takes you back to the capture a comment was written on', async ({ page }) => {
  const seeded = await seed(page)
  await commentOnStep(seeded, 1, 'the label misleads')
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)

  await page.getByRole('link', { name: 'opens the link received by e-mail' }).click()

  const carousel = page.locator('.overflow-hidden.rounded-lg').first()
  await expect(carousel).toContainText('opens the link received by e-mail')
  await expect(carousel).toContainText('the label misleads')
})

test('a capture that moved comes back asking to be looked at', async ({ page }) => {
  const seeded = await seed(page)
  await validateEverything(seeded)
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)
  await expect(page.locator('table').first().locator('[aria-label="validated"]')).toHaveCount(6)

  await moveTheDarkVariant(page, seeded)
  await page.reload()

  // Three dark squares moved, the light ones did not. The verdict they carried
  // is gone from the grid: for the question it asks, they are to judge again.
  await expect(page.locator('table').first().locator('[aria-label="moved"]')).toHaveCount(3)
  await expect(page.locator('table').first().locator('[aria-label="validated"]')).toHaveCount(3)
  await expect(page.getByText('3 captures have moved')).toBeVisible()

  // Freshness is an overlay, never a state: the case does not move.
  //
  // Matched on a visible span rather than by text alone: French had two words
  // where English has one — the tone label for a reviewed capture was `relu`,
  // the case state is `reviewed`. Translating collapsed them, and a bare text
  // match now resolves to the state icon's hidden <title> instead (#120).
  await expect(page.locator('span:visible').filter({ hasText: /^reviewed$/ })).toHaveCount(1)
})

test('every mark the grid draws wears a disc', async ({ page }) => {
  // A status is a glyph on a disc (frontend ADR 0004). Checked against a real
  // render because that is where the rule was broken: a bare glyph landed on
  // the product's own button and became one of its pixels.
  const seeded = await seed(page)
  await validateEverything(seeded)
  await commentOnStep(seeded, 0, 'needs another look')
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)

  await expect(gridMarks(page)).toHaveCount(6)
  for (const mark of await gridMarks(page).all()) {
    await expect(mark).toHaveClass(/rounded-full/)
  }
})

test('the captures in the grid actually decode, not just point somewhere', async ({ page }) => {
  // Counting squares passes whether or not the bytes arrive. `naturalWidth` is
  // zero for an image the browser could not decode, so this is what tells a
  // broken address apart from a working one (#71).
  const seeded = await seed(page)
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)

  const images = page.locator('tbody img')
  await expect(images).toHaveCount(6)

  // Polled, not sampled once: the images are still arriving when the grid is
  // already rendered, so reading naturalWidth straight away measures the
  // network rather than the address. Sampling once passed on a fast machine
  // and failed in CI with `320,320,0,320,0,0`.
  await expect
    .poll(
      async () =>
        images.evaluateAll((nodes) => nodes.map((n) => (n as HTMLImageElement).naturalWidth)),
      { message: 'every capture in the grid should decode' },
    )
    .toEqual([320, 320, 320, 320, 320, 320])
})
