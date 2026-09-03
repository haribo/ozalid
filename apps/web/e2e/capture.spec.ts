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
import { expect, test, type Page } from '@playwright/test'
import { hashOf, push, pushes, type Shot } from './push'

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

test('signing in, kept as evidence', async ({ page }) => {
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

  await push('signing in', shots)
})
