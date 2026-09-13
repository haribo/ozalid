/**
 * Walking a project's queue: many cases, one sitting (#205).
 *
 * The point of the queue is that the reviewer never returns to the catalogue
 * between two cases, so the test never does either — it opens the walk once
 * and judges to the end.
 */
import { expect, test } from '@playwright/test'
import { seedWalk } from './fixture'

const API = process.env.OZALID_API ?? 'http://localhost:8091'

test('the catalogue counts what awaits under the category on screen', async ({ page }) => {
  const seeded = await seedWalk(page)
  await page.goto(`/projects/${seeded.slug}/categories/${seeded.categoryId}`)

  // Two cases, one step each, two variants: four captures awaiting a verdict.
  // The count and the reach are one sentence (product.md §3.6).
  await expect(page.getByText(/4 captures in walk \d+ need your verdict/)).toBeVisible()
  await expect(page.getByText('4 to-review · 0 moved')).toBeVisible()
})

test('a reviewer walks a project queue across cases and judges it in one sitting', async ({
  page,
}) => {
  const seeded = await seedWalk(page)
  await page.goto(`/projects/${seeded.slug}/categories/${seeded.categoryId}`)
  await page.getByRole('button', { name: 'Review them' }).click()

  const carousel = page.getByRole('dialog', { name: 'capture' })
  const banner = page.locator('[data-test="walk-banner"]')
  const accepted = carousel.getByRole('button', { name: 'accept' })
  await expect(carousel).toContainText('1 / 4 to judge')
  await expect(banner).toContainText(seeded.cases[0].title)

  // Accept, walk, accept… The case changes under an unchanged screen, and the
  // banner is what says so — no interstitial, no transition marker.
  for (let i = 0; i < 4; i++) {
    await page.keyboard.press(' ')
    // The verdict is the server's answer, not the keypress: waiting on the
    // pair is waiting on the write that the next step depends on.
    await expect(accepted).toHaveAttribute('aria-pressed', 'true')
    if (i < 3) {
      await page.keyboard.press('ArrowRight')
      await expect(carousel).toContainText(`${i + 2} / 4 to judge`)
      // Crossing into the next case, the evidence arrives a moment after the
      // address does — the reviewer waits for pixels, and so does the test.
      await expect(page.locator('[data-test="walk-loading"]')).toBeHidden()
    }
  }
  await expect(banner).toContainText(seeded.cases[1].title)
  await expect(carousel).toContainText('4 / 4 to judge')

  await page.keyboard.press('Escape')
  await expect(carousel).toBeHidden()

  // The queue is read back on the way out: nothing is waiting any more.
  await expect(page.getByText(/need your verdict/)).toBeHidden()

  // And the server is what says the cases are judged, not the screen.
  for (const kase of seeded.cases) {
    const read = await page.request.get(`${API}/api/projects/${seeded.slug}/cases/${kase.id}`)
    expect((await read.json()).state).toBe('accepted')
  }
})

test('the walk keeps its address, so a capture in it can be pointed at', async ({ page }) => {
  const seeded = await seedWalk(page)
  await page.goto(`/projects/${seeded.slug}/categories/${seeded.categoryId}`)
  await page.getByRole('button', { name: 'Review them' }).click()
  await expect(page.getByRole('dialog', { name: 'capture' })).toContainText('1 / 4 to judge')
  await page.keyboard.press('ArrowRight')
  await expect(page.getByRole('dialog', { name: 'capture' })).toContainText('2 / 4 to judge')

  // The address names the capture being judged, inside the scope the walk was
  // started from (frontend ADR 0007).
  await expect(page).toHaveURL(
    new RegExp(
      `/projects/${seeded.slug}/categories/${seeded.categoryId}/queue/cases/[^/]+/steps/[^/]+/variants/`,
    ),
  )

  // Opened cold, that address lands on the same capture.
  const url = page.url()
  await page.goto(url)
  await expect(page.getByRole('dialog', { name: 'capture' })).toContainText('2 / 4 to judge')
})

test('the walk says it is over, and what the sitting came to', async ({ page }) => {
  const seeded = await seedWalk(page)
  await page.goto(`/projects/${seeded.slug}/categories/${seeded.categoryId}`)
  await page.getByRole('button', { name: 'Review them' }).click()

  const carousel = page.getByRole('dialog', { name: 'capture' })
  const accepted = carousel.getByRole('button', { name: 'accept' })
  // The first capture has to be on screen before a key means anything.
  await expect(carousel).toContainText('1 / 4 to judge')

  for (let i = 0; i < 4; i++) {
    await page.keyboard.press(' ')
    await expect(accepted).toHaveAttribute('aria-pressed', 'true')
    await page.keyboard.press('ArrowRight')
    if (i < 3) {
      await expect(carousel).toContainText(`${i + 2} / 4 to judge`)
      await expect(page.locator('[data-test="walk-loading"]')).toBeHidden()
    }
  }

  // Past the last capture: the tally, read back from the server rather than
  // counted in the browser (#255).
  const over = page.locator('[data-test="walk-over"]')
  await expect(over).toContainText('Nothing is waiting on you')
  await expect(over).toContainText('4 captures judged across 2 cases')
  await expect(over).toContainText('4 accepted')
  await expect(over).toContainText('0 refused')

  await over.getByRole('button', { name: /^Back to/ }).click()
  await expect(carousel).toBeHidden()
  await expect(page.getByText(/need your verdict/)).toBeHidden()
})
