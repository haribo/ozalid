/**
 * A reviewer's sitting, through the interface, against a real server.
 *
 * Every test seeds its own case. Sharing one across a file makes each test
 * depend on what the previous one did, and the first failure then tells you
 * about the second — which is exactly what happened when this suite was
 * written that way.
 */
import { expect, test, type Page } from '@playwright/test'
import { commentOnStep, moveTheDarkVariant, seed, acceptEverything } from './fixture'
import { emptyMailbox, linkSentTo } from './mailbox'

/** The grid's own captures. The recap is another table, and its ticks are actions
 * rather than statuses — an assertion that spans both proves nothing about
 * either. */
const gridMarks = (page: Page) => page.locator('table').first().locator('tbody [role="img"]')

test('a case arrives with everything left to judge', async ({ page }) => {
  const seeded = await seed(page)
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)

  await expect(page.getByRole('heading', { name: 'reset a forgotten password' })).toBeVisible()
  await expect(page.getByText('to-review')).toBeVisible()

  // Six captures, none of them marked: nothing has been said, and a bare capture
  // is the only reading that leaves every pixel visible (frontend ADR 0003).
  await expect(page.locator('tbody button[aria-label*="in the carousel"]')).toHaveCount(6)
  await expect(gridMarks(page)).toHaveCount(0)
})

test('clicking a capture opens the carousel on that exact capture', async ({ page }) => {
  const seeded = await seed(page)
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)
  await page.locator('tbody button[aria-label*="in the carousel"]').nth(2).click()

  const carousel = page.getByRole('dialog', { name: 'capture' })
  await expect(carousel).toContainText('opens the link received by e-mail')
  await expect(carousel).toContainText('2 / 3')
})

test('space accepts, and the server is what says so', async ({ page }) => {
  const seeded = await seed(page)
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)
  await page.locator('tbody button[aria-label*="in the carousel"]').first().click()
  await page.keyboard.press(' ')

  await expect(page.locator('table').first().locator('[aria-label="accepted"]')).toHaveCount(1)

  // Redrawn from what the server answered, never from a guess made in the
  // browser (ADR 0002) — so it survives a reload.
  await page.reload()
  await expect(page.locator('table').first().locator('[aria-label="accepted"]')).toHaveCount(1)

  // And the same key takes it back (#156): a misclick is not a life
  // sentence. The carousel route survived the reload, so the capture is still
  // under the keyboard.
  await page.keyboard.press(' ')
  await page.keyboard.press('Escape')
  await expect(page.locator('table').first().locator('[aria-label="accepted"]')).toHaveCount(0)
  await page.reload()
  await expect(page.locator('table').first().locator('[aria-label="accepted"]')).toHaveCount(0)
})

test('one defect over two variants is one comment, not two', async ({ page }) => {
  const seeded = await seed(page)
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)
  await page.locator('tbody button[aria-label*="in the carousel"]').nth(2).click()
  await page.getByRole('button', { name: 'refuse', exact: true }).click()

  await page.locator('textarea').fill('the button is clipped')
  await page.getByRole('button', { name: 'all' }).click()
  await page
    .getByRole('dialog', { name: 'capture' })
    .locator('form')
    .getByRole('button', { name: 'refuse' })
    .click()

  // One row in the recap — the text also shows in the carousel, so the count is
  // taken where the claim is about.
  const recap = page.locator('table').last()
  await expect(recap.getByText('the button is clipped')).toHaveCount(1)
  await expect(page.locator('table').first().locator('[aria-label="refused"]')).toHaveCount(2)
})

test('the recap takes you back to the capture a comment was written on', async ({ page }) => {
  const seeded = await seed(page)
  await commentOnStep(seeded, 1, 'the label misleads')
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)

  await page.getByRole('link', { name: 'opens the link received by e-mail' }).click()

  const carousel = page.getByRole('dialog', { name: 'capture' })
  await expect(carousel).toContainText('opens the link received by e-mail')
  await expect(carousel).toContainText('the label misleads')
})

test('a capture that moved comes back asking to be looked at', async ({ page }) => {
  const seeded = await seed(page)
  await acceptEverything(seeded)
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)
  await expect(page.locator('table').first().locator('[aria-label="accepted"]')).toHaveCount(6)

  await moveTheDarkVariant(page, seeded)
  await page.reload()

  // Three dark captures moved, the light ones did not. The verdict they carried
  // is gone from the grid: for the question it asks, they are to judge again.
  await expect(page.locator('table').first().locator('[aria-label="moved"]')).toHaveCount(3)
  await expect(page.locator('table').first().locator('[aria-label="accepted"]')).toHaveCount(3)
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
  await acceptEverything(seeded)
  await commentOnStep(seeded, 0, 'needs another look')
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)

  await expect(gridMarks(page)).toHaveCount(6)
  for (const mark of await gridMarks(page).all()) {
    await expect(mark).toHaveClass(/rounded-full/)
  }
})

test('the captures in the grid actually decode, not just point somewhere', async ({ page }) => {
  // Counting captures passes whether or not the bytes arrive. `naturalWidth` is
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

test('a capture has an address, and the window is what sizes it', async ({ page }) => {
  const seeded = await seed(page)
  const grid = await (
    await page.request.get(`/api/projects/${seeded.slug}/cases/${seeded.caseId}/captures`)
  ).json()
  const step = grid.steps[1]
  const capture = step.captures[0]

  // Loaded directly, the way a colleague sent "look at step 2 in dark" would:
  // no grid was clicked, yet the dialog opens on that exact capture (#125).
  await page.goto(
    `/projects/${seeded.slug}/cases/${seeded.caseId}/steps/${step.id}/variants/${capture.variantId}`,
  )
  const carousel = page.getByRole('dialog', { name: 'capture' })
  await expect(carousel).toContainText(step.name)

  // The capture renders at its own size when the window has room — never
  // stretched, since blown-up pixels are falsified pixels. The old code wrote
  // the width in advance and stretched this 320 px fixture to 560 (#125).
  await page.setViewportSize({ width: 1600, height: 900 })
  const shot = carousel.locator('img')
  const natural = await shot.evaluate((img) => (img as HTMLImageElement).naturalWidth)
  const roomy = (await shot.boundingBox())!.width
  expect(Math.abs(roomy - natural)).toBeLessThanOrEqual(4)

  // And it tracks the window once the window is what constrains it.
  await page.setViewportSize({ width: 300, height: 500 })
  expect((await shot.boundingBox())!.width).toBeLessThan(natural)

  // Arrows walk by replacing, so leaving means the grid — not a retrace of
  // every capture looked at.
  await page.keyboard.press('ArrowLeft')
  await expect(page).toHaveURL(new RegExp(`/steps/${grid.steps[0].id}/`))
  await page.keyboard.press('Escape')
  await expect(page).toHaveURL(new RegExp(`/cases/${seeded.caseId}$`))
  await expect(page.locator('table').first()).toBeVisible()
})

test('a verdict held through an expired session survives closing the carousel', async ({
  page,
  context,
}) => {
  // The #70 walk, run from the carousel route. Closing navigates; the two
  // routes share one component, so the instance — and the verdict it holds —
  // must survive that navigation. A remount would drop it silently (#125).
  const seeded = await seed(page)
  const grid = await (
    await page.request.get(`/api/projects/${seeded.slug}/cases/${seeded.caseId}/captures`)
  ).json()
  const step = grid.steps[0]
  await page.goto(
    `/projects/${seeded.slug}/cases/${seeded.caseId}/steps/${step.id}/variants/${step.captures[0].variantId}`,
  )
  await expect(page.getByRole('dialog', { name: 'capture' })).toBeVisible()

  // Somebody to come back as, made while the session still works: asking for
  // another of REVIEWER's links would eat the rate limiter's per-address
  // budget and starve the sign-in suite behind this test.
  const email = `resumed-${Date.now()}@example.test`
  const account = await (
    await page.request.post(`/api/accounts`, { data: { name: 'resumed reviewer', email } })
  ).json()
  await page.request.put(`/api/projects/${seeded.slug}/members/${account.id}`, {
    data: { rights: 'member' },
  })

  // The session dies under the verdict.
  await context.clearCookies()
  await page.keyboard.press(' ')
  await expect(page.getByText('Session expired. The last verdict was not recorded.')).toBeVisible()

  // The reviewer closes the capture anyway, then signs back in from the bar's
  // own path: another tab claims a link, this tab is told and resumes.
  await page.keyboard.press('Escape')
  await expect(page).toHaveURL(new RegExp(`/cases/${seeded.caseId}$`))

  await emptyMailbox(email)
  const other = await context.newPage()
  await other.goto('/sign-in')
  await other.getByLabel('address').fill(email)
  await other.getByRole('button', { name: 'Send the link' }).click()
  await other.goto(`/sign-in/${await linkSentTo(email)}`)

  // The held verdict lands: the capture the space bar judged is accepted.
  await expect(page.getByText('Session expired')).toHaveCount(0)
  await expect(
    page.locator('table').first().locator('[aria-label="accepted"]').first(),
  ).toBeVisible()
})

test('refusing writes the remark and the capture stays on screen', async ({ page }) => {
  // The sheet opens under the image (ADR 0020): the reviewer describes pixels
  // they can still see — a modal covering them was rejected for exactly that.
  const seeded = await seed(page)
  await page.goto(`/projects/${seeded.slug}/cases/${seeded.caseId}`)
  await page.locator('tbody button[aria-label*="in the carousel"]').first().click()

  const carousel = page.getByRole('dialog', { name: 'capture' })
  await carousel.getByRole('button', { name: 'refuse', exact: true }).click()
  await expect(carousel.locator('textarea')).toBeVisible()
  await expect(carousel.locator('img')).toBeVisible()

  // Escape closes the sheet, not the carousel, and nothing was sent.
  await page.keyboard.press('Escape')
  await expect(carousel.locator('textarea')).toHaveCount(0)
  await expect(carousel).toBeVisible()
  await expect(page.locator('table').first().locator('[aria-label="refused"]')).toHaveCount(0)
})

test('a capture taller than the stage is scaled to fit, never overflowing', async ({
  page,
  request,
  browser,
}) => {
  // An oversized capture, pushed the way a client would push one (#177).
  const seeded = await seed(page)
  const shooter = await browser.newPage({ viewport: { width: 900, height: 1800 } })
  await shooter.setContent(
    `<!doctype html><style>body{margin:0;width:900px;height:1800px;background:#eef}</style><p>tall</p>`,
  )
  const bytes = await shooter.screenshot()
  await shooter.close()
  const { createHash } = await import('node:crypto')
  const hash = `sha256:${createHash('sha256').update(bytes).digest('hex')}`
  const API = process.env.OZALID_API ?? 'http://localhost:8091'
  const TOKEN = process.env.OZALID_E2E_TOKEN ?? ''
  await request.put(`${API}/api/projects/${seeded.slug}/blobs/${hash}`, {
    headers: { authorization: `Bearer ${TOKEN}` },
    data: bytes,
  })
  const categories = await (
    await request.get(`${API}/api/projects/${seeded.slug}/categories`, {
      headers: { authorization: `Bearer ${TOKEN}` },
    })
  ).json()
  const making = await request.post(`${API}/api/projects/${seeded.slug}/cases`, {
    headers: { authorization: `Bearer ${TOKEN}` },
    data: { title: `tall ${Date.now()}`, categoryId: categories[0].id },
  })
  expect(making.status(), await making.text()).toBe(201)
  const made = await making.json()
  await request.post(`${API}/api/projects/${seeded.slug}/editions`, {
    headers: { authorization: `Bearer ${TOKEN}` },
    data: {
      cases: [
        {
          id: made.id,
          steps: [
            {
              name: 'a very tall screen',
              captures: [
                {
                  variant: { theme: 'light', device: 'desktop' },
                  hash,
                  provenance: { environmentId: 'ci' },
                },
              ],
            },
          ],
        },
      ],
    },
  })

  await page.goto(`/projects/${seeded.slug}/cases/${made.id}`)
  await page.locator('tbody button[aria-label*="a very tall screen"]').first().click()
  const dialog = page.getByRole('dialog', { name: 'capture' })
  const img = dialog.locator('img')
  await expect(img).toBeVisible()

  // The pixels are judged at the largest size that fits (#125) — and fitting
  // is the whole claim: the image never leaves the viewport.
  const box = (await img.boundingBox())!
  const viewport = page.viewportSize()!
  expect(box.y).toBeGreaterThanOrEqual(0)
  expect(box.y + box.height).toBeLessThanOrEqual(viewport.height)
  expect(box.x + box.width).toBeLessThanOrEqual(viewport.width)
})
