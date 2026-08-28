/**
 * Signing in, walked end to end: a link is asked for, a message arrives, the
 * link is spent, and a session exists.
 *
 * The message is read from a real mailbox rather than from the database, so the
 * sending is watched too — it is the half most likely to be misconfigured on a
 * real instance, and the half a database shortcut would never exercise.
 */
import { expect, test } from '@playwright/test'

const API = process.env.OZALID_API ?? 'http://localhost:8091'
const MAILPIT = process.env.OZALID_E2E_MAILPIT ?? 'http://localhost:8045'

/** The link in the newest message sent to an address. */
async function linkSentTo(address: string): Promise<string> {
  const listed = await fetch(`${MAILPIT}/api/v1/search?query=${encodeURIComponent('to:' + address)}`)
  const { messages } = (await listed.json()) as { messages: { ID: string }[] }
  expect(messages.length, `no message reached ${address}`).toBeGreaterThan(0)

  const body = await fetch(`${MAILPIT}/api/v1/message/${messages[0].ID}`)
  const { Text } = (await body.json()) as { Text: string }
  const found = Text.match(/\/sign-in\/([A-Za-z0-9_-]+)/)
  expect(found, `no sign-in link in the message:\n${Text}`).not.toBeNull()
  return found![1]
}

async function ask(email: string) {
  return fetch(`${API}/api/sign-in`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ email }),
  })
}

test('a person asks, reads their mail, and is signed in', async ({ request }) => {
  // The account the e2e instance was bootstrapped with.
  const email = 'e2e@ozalid.test'

  const asked = await ask(email)
  expect(asked.status).toBe(202)

  const link = await linkSentTo(email)

  const claimed = await request.post(`${API}/api/sign-in/claim`, { data: { link } })
  expect(claimed.status()).toBe(204)

  // The session rides in a cookie the browser holds, so the same context now
  // answers as somebody.
  const me = await request.get(`${API}/api/me`)
  expect(me.status()).toBe(200)
  expect((await me.json()).email).toBe(email)

  // Spent. The second attempt is refused the same way an invented link is.
  const again = await request.post(`${API}/api/sign-in/claim`, { data: { link } })
  expect(again.status()).toBe(401)

  await request.post(`${API}/api/sign-out`)
  expect((await request.get(`${API}/api/me`)).status()).toBe(401)
})

test('an address nobody has gets the same answer as one that exists', async () => {
  // Whether somebody has an account here is not something a stranger learns by
  // asking, so the two answers have to be identical.
  const stranger = await ask(`nobody-${Date.now()}@ozalid.test`)
  expect(stranger.status).toBe(202)
})

test('a link nobody issued is refused', async ({ request }) => {
  const claimed = await request.post(`${API}/api/sign-in/claim`, { data: { link: 'never-issued' } })
  expect(claimed.status()).toBe(401)
})
