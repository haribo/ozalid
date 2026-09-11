/**
 * ozalid, looking at its own screens.
 *
 * The suite already walks every one of them in a real browser; this keeps the
 * pictures and pushes them, so a change to the interface arrives as captures in
 * ozalid and is reviewed there.
 *
 * **The flow is chosen for stability, not importance.** Byte comparison is the
 * whole mechanism, and most of ozalid's screens carry a date, a generated name
 * or an id that moves on every run — push those and freshness marks everything
 * forever, which is a signal nobody reads. The two screens below carry none of
 * that (#107).
 */
import { readFile } from 'node:fs/promises'
import { expect, test, type Page } from '@playwright/test'
import { hashOf, push, pushes, type Recording, type Shot } from './push'
import { seed } from './fixture'
import { SIGNED_IN } from './session'

// Signed out, deliberately: these screens exist before anybody is.
test.use({ storageState: { cookies: [], origins: [] } })

test.skip(!pushes, 'set OZALID_PUSH_API and OZALID_PUSH_TOKEN to push captures')

/**
 * An address nobody has, and a different one per shot.
 *
 * Nobody has it, so the server sends no mail and the screen is the same by
 * design — the answer does not say whether an account exists. Different every
 * time because asking too often for one address is refused, and a 429 would put
 * an error on the picture instead of the message.
 */
const anAddress = () =>
  `capture-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@ozalid.invalid`

/**
 * A screenshot of a fresh paint, not of a patch.
 *
 * After an interaction, Chromium repaints only the damaged region, and the
 * anti-aliasing of rounded corners composed onto an earlier paint differs by
 * one channel unit from the same corners painted whole — 32 pixels of ±1,
 * found by diffing two captures of one settled screen. Byte comparison is the
 * whole mechanism (#107), so every shot is taken from a full paint: hide,
 * frame, show, frame.
 */
async function freshPaint(page: Page): Promise<Buffer> {
  await page.waitForTimeout(250) // transition-all is 150ms; the label has landed
  await page.evaluate(async () => {
    document.body.style.visibility = 'hidden'
    await new Promise((r) => requestAnimationFrame(r))
    document.body.style.visibility = ''
    await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
  })
  return page.screenshot({ fullPage: true })
}

async function walk(page: Page, theme: 'light' | 'dark'): Promise<Shot[]> {
  const variant = { theme }
  await page.emulateMedia({ colorScheme: theme })

  await page.goto('/sign-in')
  await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()
  const asked: Shot[] = [
    {
      step: 'arrive at the door',
      variant,
      bytes: await freshPaint(page),
    },
  ]

  // A fixed address, typed and never sent: the bytes must be the same on
  // every run, and this is the screen where the label lands on the border —
  // the state the first real review asked to see (#131). Sending it would
  // mail a real-looking address and eat the rate limiter; the unique one
  // below replaces it before the click.
  await page.getByLabel('address').fill('reviewer@ozalid.example')
  // Blurred before the picture: focus was never what the picture was about,
  // and a focused field invites the renderer to differ. The label stays
  // landed, since the value holds it there.
  await page.getByLabel('address').blur()
  await expect
    .poll(async () => {
      const label = (await page.locator('form label').boundingBox())!
      const input = (await page.getByLabel('address').boundingBox())!
      return input.y - label.y
    })
    .toBeGreaterThan(4)
  asked.push({
    step: 'enter an address',
    variant,
    bytes: await freshPaint(page),
  })

  await page.getByLabel('address').fill(anAddress())
  await page.getByRole('button', { name: 'Send the link' }).click()
  await expect(page.getByText('The link is on its way.')).toBeVisible()
  asked.push({
    step: 'ask for a link',
    variant,
    bytes: await freshPaint(page),
  })

  return asked
}

test('signing in, kept as evidence', async ({ page, browser }) => {
  const shots = [...(await walk(page, 'light')), ...(await walk(page, 'dark'))]

  // Three screens, two themes: the axis a change most often breaks on one
  // side only.
  expect(shots).toHaveLength(6)

  // Walked again, and the bytes must be the same ones. This is the check the
  // whole thing rests on: byte comparison is the mechanism, so a screen that
  // differs between two runs of the same code would mark itself moved forever,
  // and a signal that always fires is one nobody reads (#107).
  //
  // It fails the day a date, a generated name or an id reaches one of these
  // screens — which is why the flow was chosen for stability rather than for
  // importance.
  const again = [...(await walk(page, 'light')), ...(await walk(page, 'dark'))]
  expect(again.map((s) => `${s.step}·${s.variant.theme}·${hashOf(s.bytes)}`)).toEqual(
    shots.map((s) => `${s.step}·${s.variant.theme}·${hashOf(s.bytes)}`),
  )

  // The same walk once more per theme, filmed. The video is never compared
  // and never byte-stable (ADR 0013) — it is there to be watched, not judged.
  const recordings: Recording[] = []
  for (const theme of ['light', 'dark'] as const) {
    const filming = await browser.newContext({
      // Pinned to the viewport: without a size Playwright scales the frame
      // down to fit 800×800, and the pushed video arrives small (#232).
      recordVideo: { dir: test.info().outputPath('videos'), size: { width: 1280, height: 720 } },
      viewport: { width: 1280, height: 720 },
      storageState: { cookies: [], origins: [] },
      colorScheme: theme,
      baseURL: test.info().project.use.baseURL,
    })
    const filmed = await filming.newPage()
    await walk(filmed, theme)
    const video = filmed.video()
    await filming.close()
    if (video) {
      // saveAs is the finished file; path() can still be being muxed, and
      // hashing a file that is still growing pushed one set of bytes under
      // another set's address.
      const saved = test.info().outputPath(`videos/${theme}.webm`)
      await video.saveAs(saved)
      recordings.push({ variant: { theme }, bytes: await readFile(saved) })
    }
  }
  expect(recordings).toHaveLength(2)

  await push('signing in', shots, recordings)
})

/**
 * The dead sign-in link, turned away (#241): a link nobody issued lands on a
 * fixed amber panel — one step, no id on screen (the token stays in the URL),
 * chosen like every capture flow for its byte-stability (#107).
 */
async function walkDeadLink(page: Page, theme: 'light' | 'dark'): Promise<Shot[]> {
  await page.emulateMedia({ colorScheme: theme })
  await page.goto('/sign-in/a-link-nobody-issued')
  await expect(page.getByText('This link no longer works.')).toBeVisible()
  return [
    {
      step: 'is turned away',
      variant: { theme },
      bytes: await freshPaint(page),
    },
  ]
}

test('a dead link, kept as evidence', async ({ page, browser }) => {
  const shots = [...(await walkDeadLink(page, 'light')), ...(await walkDeadLink(page, 'dark'))]
  expect(shots).toHaveLength(2)

  // The same stability check the sign-in flow carries: byte comparison is
  // the mechanism, and an unstable screen would mark itself moved forever.
  const again = [...(await walkDeadLink(page, 'light')), ...(await walkDeadLink(page, 'dark'))]
  expect(again.map((s) => `${s.variant.theme}·${hashOf(s.bytes)}`)).toEqual(
    shots.map((s) => `${s.variant.theme}·${hashOf(s.bytes)}`),
  )

  const recordings: Recording[] = []
  for (const theme of ['light', 'dark'] as const) {
    const filming = await browser.newContext({
      recordVideo: { dir: test.info().outputPath('videos'), size: { width: 1280, height: 720 } },
      viewport: { width: 1280, height: 720 },
      storageState: { cookies: [], origins: [] },
      colorScheme: theme,
      baseURL: test.info().project.use.baseURL,
    })
    const filmed = await filming.newPage()
    await walkDeadLink(filmed, theme)
    const video = filmed.video()
    await filming.close()
    if (video) {
      const saved = test.info().outputPath(`videos/dead-link-${theme}.webm`)
      await video.saveAs(saved)
      recordings.push({ variant: { theme }, bytes: await readFile(saved) })
    }
  }
  expect(recordings).toHaveLength(2)

  await push('a dead link is turned away', shots, recordings)
})

/**
 * Judging a capture in the carousel (#243): ozalid's core surface, stable by
 * construction — step names and variant labels are fixed fixture strings, the
 * stage shows a deterministic fixture PNG, and the carousel covers the page
 * underneath, so no case id or date reaches the pixels.
 *
 * Signed in, unlike the file's other flows: judging is a reviewer's act.
 * Every walk seeds its own case, so the acceptance taken in step three never
 * leaks into the next walk's pixels.
 */
test.describe('judging a capture', () => {
  test.use({ storageState: SIGNED_IN })

  async function walkCarousel(page: Page, theme: 'light' | 'dark'): Promise<Shot[]> {
    const variant = { theme }
    await page.emulateMedia({ colorScheme: theme })
    const seeded = await seed(page)
    await page.goto(`/projects/e2e/cases/${seeded.caseId}`)

    // Into the carousel on the variant matching the UI theme, first step.
    await page
      .locator(`tbody button[aria-label$="${theme}·desktop in the carousel"]`)
      .first()
      .click()
    const carousel = page.getByRole('dialog', { name: 'capture' })
    await expect(carousel.getByRole('button', { name: 'accept', exact: true })).toBeVisible()
    const shots: Shot[] = [{ step: 'faces the capture', variant, bytes: await freshPaint(page) }]

    // The refusal sheet: the remark empty, the variant ticks visible. Blurred
    // so no caret ever blinks into the bytes.
    await carousel.getByRole('button', { name: 'refuse', exact: true }).click()
    await expect(carousel.getByText('Refuse the capture')).toBeVisible()
    await carousel.locator('textarea').blur()
    shots.push({ step: 'weighs a refusal', variant, bytes: await freshPaint(page) })

    // Cancel is a true no-op; accepting fills the half and tints the edge.
    await page.keyboard.press('Escape')
    await carousel.getByRole('button', { name: 'accept', exact: true }).click()
    await expect(carousel.getByRole('button', { name: '\u2713 accepted' })).toBeVisible()
    shots.push({ step: 'has accepted', variant, bytes: await freshPaint(page) })

    return shots
  }

  test('judging a capture, kept as evidence', async ({ page, browser }) => {
    const shots = [...(await walkCarousel(page, 'light')), ...(await walkCarousel(page, 'dark'))]
    expect(shots).toHaveLength(6)

    const again = [...(await walkCarousel(page, 'light')), ...(await walkCarousel(page, 'dark'))]
    expect(again.map((s) => `${s.step}\u00b7${s.variant.theme}\u00b7${hashOf(s.bytes)}`)).toEqual(
      shots.map((s) => `${s.step}\u00b7${s.variant.theme}\u00b7${hashOf(s.bytes)}`),
    )

    const recordings: Recording[] = []
    for (const theme of ['light', 'dark'] as const) {
      const filming = await browser.newContext({
        recordVideo: { dir: test.info().outputPath('videos'), size: { width: 1280, height: 720 } },
        viewport: { width: 1280, height: 720 },
        storageState: SIGNED_IN,
        colorScheme: theme,
        baseURL: test.info().project.use.baseURL,
      })
      const filmed = await filming.newPage()
      await walkCarousel(filmed, theme)
      const video = filmed.video()
      await filming.close()
      if (video) {
        const saved = test.info().outputPath(`videos/carousel-${theme}.webm`)
        await video.saveAs(saved)
        recordings.push({ variant: { theme }, bytes: await readFile(saved) })
      }
    }
    expect(recordings).toHaveLength(2)

    await push('judging a capture', shots, recordings, 'review')
  })
})
