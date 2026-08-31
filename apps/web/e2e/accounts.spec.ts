/**
 * Onboarding, walked through the API an administrator actually uses.
 *
 * Until this existed, an instance could not be handed a second person: the only
 * way in was `ozalid bootstrap`, which mints a fresh administrator every time
 * it runs.
 */
import { expect, test } from '@playwright/test'

const API = process.env.OZALID_API ?? 'http://localhost:8091'
const TOKEN = process.env.OZALID_E2E_TOKEN ?? ''

/** An address nobody has yet: this suite writes for real. */
const anAddress = () => `hired-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.test`

test('an administrator makes an account, and it is listed', async ({ request }) => {
  const email = anAddress()

  const made = await request.post(`${API}/api/accounts`, { data: { name: 'nina', email } })
  expect(made.status()).toBe(201)
  const account = await made.json()
  expect(account.email).toBe(email)
  expect(account.isAdmin).toBe(false)
  expect(account.deactivatedAt).toBeUndefined()

  const listed = await request.get(`${API}/api/accounts`)
  expect(listed.status()).toBe(200)
  expect((await listed.json()).map((a: { id: string }) => a.id)).toContain(account.id)
})

test('the same address twice is refused, and says which way to fix it', async ({ request }) => {
  const email = anAddress()

  expect(
    (await request.post(`${API}/api/accounts`, { data: { name: 'nina', email } })).status(),
  ).toBe(201)

  const again = await request.post(`${API}/api/accounts`, { data: { name: 'nina again', email } })
  expect(again.status()).toBe(409)
  expect((await again.json()).title).toContain('already has an account')
})

test('an account without a name or an address is refused', async ({ request }) => {
  const blank = await request.post(`${API}/api/accounts`, {
    data: { name: '  ', email: anAddress() },
  })
  expect(blank.status()).toBe(400)

  const noAddress = await request.post(`${API}/api/accounts`, { data: { name: 'nina', email: '' } })
  expect(noAddress.status()).toBe(400)
})

test('a deactivated account is still listed, and still says who it was', async ({ request }) => {
  const made = await request.post(`${API}/api/accounts`, {
    data: { name: 'leaving', email: anAddress() },
  })
  const account = await made.json()

  expect((await request.delete(`${API}/api/accounts/${account.id}`)).status()).toBe(204)
  // Idempotent: the day it happened does not move.
  expect((await request.delete(`${API}/api/accounts/${account.id}`)).status()).toBe(204)

  const listed = await request.get(`${API}/api/accounts`)
  const found = (await listed.json()).find((a: { id: string }) => a.id === account.id)
  expect(found, 'the deactivated account vanished from the list').toBeTruthy()
  expect(found.deactivatedAt).toBeTruthy()
  expect(found.name).toBe('leaving')
})

test('a service account administers nothing, however good its token', async () => {
  // A token belongs to one project and administration belongs to none, so this
  // is a refusal about rights rather than about the credential (product.md §8.2).
  const refused = await fetch(`${API}/api/accounts`, {
    method: 'POST',
    headers: { 'content-type': 'application/json', authorization: `Bearer ${TOKEN}` },
    body: JSON.stringify({ name: 'a program', email: anAddress() }),
  })
  expect(refused.status).toBe(403)

  const anonymous = await fetch(`${API}/api/accounts`)
  expect(anonymous.status).toBe(401)
})

test('deactivating an account nobody has is not found', async ({ request }) => {
  expect((await request.delete(`${API}/api/accounts/nobody-has-this`)).status()).toBe(404)
})

test('a role set wrongly at creation is not permanent', async ({ request }) => {
  const made = await request.post(`${API}/api/accounts`, {
    data: { name: 'meant to administer', email: anAddress() },
  })
  const account = await made.json()
  expect(account.isAdmin).toBe(false)

  expect(
    (
      await request.patch(`${API}/api/accounts/${account.id}`, { data: { isAdmin: true } })
    ).status(),
  ).toBe(204)

  const listed = await (await request.get(`${API}/api/accounts`)).json()
  expect(listed.find((a: { id: string }) => a.id === account.id).isAdmin).toBe(true)

  // And back down, since there is another administrator standing.
  expect(
    (
      await request.patch(`${API}/api/accounts/${account.id}`, { data: { isAdmin: false } })
    ).status(),
  ).toBe(204)
})

test('the last administrator cannot be removed, by either route', async ({ request }) => {
  // The suite is bootstrapped with exactly one administrator, and the tests
  // above make no others — so this is the last one, and both routes out of
  // administration must refuse.
  const me = await (await request.get(`${API}/api/me`)).json()

  const demoted = await request.patch(`${API}/api/accounts/${me.id}`, { data: { isAdmin: false } })
  expect(demoted.status()).toBe(409)
  expect((await demoted.json()).title).toContain('last administrator')

  const removed = await request.delete(`${API}/api/accounts/${me.id}`)
  expect(removed.status()).toBe(409)

  // Still standing, so the refusals were not cosmetic.
  expect((await request.get(`${API}/api/accounts`)).status()).toBe(200)
})

test('a role nobody has is not found', async ({ request }) => {
  expect(
    (
      await request.patch(`${API}/api/accounts/nobody-has-this`, { data: { isAdmin: true } })
    ).status(),
  ).toBe(404)
})
