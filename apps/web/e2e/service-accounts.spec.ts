/**
 * The programs that push to a project, and the credentials they present.
 *
 * The story worth testing is the rotation: minting a token must not break the
 * one already deployed, or nobody will ever rotate. Everything else here is
 * about a token stopping when somebody decides it should.
 */
import { expect, test } from '@playwright/test'

const API = process.env.OZALID_API ?? 'http://localhost:8091'
const PROJECT = process.env.OZALID_E2E_PROJECT ?? 'e2e'
const TOKEN = process.env.OZALID_E2E_TOKEN ?? ''

const unique = () => `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`

/** What a token can reach: the project's catalogue, and nothing outside it. */
const reads = async (token: string) =>
  (await fetch(`${API}/api/projects/${PROJECT}/cases`, {
    headers: { authorization: `Bearer ${token}` },
  })).status

test('a program is given an account, a token, and the token works', async ({ request }) => {
  const made = await request.post(`${API}/api/projects/${PROJECT}/service-accounts`, {
    data: { name: `runner ${unique()}`, tokenLabel: 'the pipeline' },
  })
  expect(made.status()).toBe(201)
  const { account, token } = await made.json()

  expect(token.token).toMatch(/^ozp_/)
  expect(token.label).toBe('the pipeline')
  expect(await reads(token.token)).toBe(200)

  // Shown once: what comes back afterwards is the label, never the token.
  const listed = await request.get(`${API}/api/projects/${PROJECT}/service-accounts/${account.id}/tokens`)
  expect(listed.status()).toBe(200)
  const tokens = await listed.json()
  expect(tokens).toHaveLength(1)
  expect(tokens[0].label).toBe('the pipeline')
  expect(tokens[0].token).toBeUndefined()
  // It has been presented once, which is what makes a rotation finishable.
  expect(tokens[0].lastUsedAt).toBeTruthy()
})

test('minting a second token leaves the first one working', async ({ request }) => {
  const made = await request.post(`${API}/api/projects/${PROJECT}/service-accounts`, {
    data: { name: `rotating ${unique()}`, tokenLabel: 'the old one' },
  })
  const { account, token: first } = await made.json()

  const minted = await request.post(`${API}/api/projects/${PROJECT}/service-accounts/${account.id}/tokens`, {
    data: { label: 'the new one' },
  })
  expect(minted.status()).toBe(201)
  const second = await minted.json()
  expect(second.token).not.toBe(first.token)

  // Both work. Retiring on mint would take the pipeline down at the moment
  // somebody was trying to make it safer.
  expect(await reads(first.token)).toBe(200)
  expect(await reads(second.token)).toBe(200)

  // Then the old one goes, and only the old one.
  const tokens = await (await request.get(`${API}/api/projects/${PROJECT}/service-accounts/${account.id}/tokens`)).json()
  const old = tokens.find((t: { label: string }) => t.label === 'the old one')
  expect((await request.delete(`${API}/api/projects/${PROJECT}/service-accounts/${account.id}/tokens/${old.id}`)).status()).toBe(204)

  expect(await reads(first.token)).toBe(401)
  expect(await reads(second.token)).toBe(200)
})

test('retiring the account stops every token it held', async ({ request }) => {
  const made = await request.post(`${API}/api/projects/${PROJECT}/service-accounts`, {
    data: { name: `leaving ${unique()}`, tokenLabel: 'first' },
  })
  const { account, token } = await made.json()
  const other = await (await request.post(`${API}/api/projects/${PROJECT}/service-accounts/${account.id}/tokens`, {
    data: { label: 'second' },
  })).json()

  expect((await request.delete(`${API}/api/projects/${PROJECT}/service-accounts/${account.id}`)).status()).toBe(204)
  // Idempotent, like retiring a person: the day it happened does not move, and
  // asking twice is not an error.
  expect((await request.delete(`${API}/api/projects/${PROJECT}/service-accounts/${account.id}`)).status()).toBe(204)

  expect(await reads(token.token)).toBe(401)
  expect(await reads(other.token)).toBe(401)
})

test('a token without a label is refused, and says why', async ({ request }) => {
  const noLabel = await request.post(`${API}/api/projects/${PROJECT}/service-accounts`, {
    data: { name: 'unnamed credential', tokenLabel: '  ' },
  })
  expect(noLabel.status()).toBe(400)
  expect((await noLabel.json()).detail).toContain('dares retire')
})

test('a program cannot make itself another program', async () => {
  // The escalation this guards: a service account minting a service account
  // leaves behind a credential that outlives whoever made it, holding rights
  // nobody reviews (product.md §8.2).
  const refused = await fetch(`${API}/api/projects/${PROJECT}/service-accounts`, {
    method: 'POST',
    headers: { 'content-type': 'application/json', authorization: `Bearer ${TOKEN}` },
    body: JSON.stringify({ name: 'a second runner', tokenLabel: 'sneaky' }),
  })
  expect(refused.status).toBe(403)
})

test('a service account belongs to one project and is not reachable from another', async ({ request }) => {
  const slug = `elsewhere-${unique()}`
  await request.post(`${API}/api/projects`, { data: { slug, name: 'another team' } })

  const made = await request.post(`${API}/api/projects/${PROJECT}/service-accounts`, {
    data: { name: `bound ${unique()}`, tokenLabel: 'bound' },
  })
  const { account } = await made.json()

  // Named under a project it does not belong to, it is simply not there.
  expect((await request.get(`${API}/api/projects/${slug}/service-accounts/${account.id}/tokens`)).status()).toBe(404)
  expect((await request.delete(`${API}/api/projects/${slug}/service-accounts/${account.id}`)).status()).toBe(404)
})
