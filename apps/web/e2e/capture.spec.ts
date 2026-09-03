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

async function walk(page: Page, theme: 'light' | 'dark'): Promise<Shot[]> {
  const variant = { theme }
  await page.emulateMedia({ colorScheme: theme })

  await page.goto('/sign-in')
  await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()
  const asked: Shot[] = [
    { step: 'arrive at the door', variant, bytes: await page.screenshot({ fullPage: true }) },
  ]

  await page.getByLabel('address').fill(anAddress())
  await page.getByRole('button', { name: 'Send the link' }).click()
  await expect(page.getByText('The link is on its way.')).toBeVisible()
  asked.push({
    step: 'ask for a link',
    variant,
    bytes: await page.screenshot({ fullPage: true }),
  })

  return asked
}

test('signing in, kept as evidence', async ({ page }) => {
  const shots = [...(await walk(page, 'light')), ...(await walk(page, 'dark'))]

  // Two screens, two themes: the axis a change most often breaks on one side
  // only.
  expect(shots).toHaveLength(4)

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
