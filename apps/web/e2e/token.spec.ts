/**
 * What a caller without a token gets, through the real server.
 *
 * The three refusals are checked separately because they are three different
 * mistakes, and a client that cannot tell them apart cannot fix any of them.
 */
import { expect, test } from '@playwright/test'

const API = process.env.OZALID_API ?? 'http://localhost:8091'
const PROJECT = process.env.OZALID_E2E_PROJECT ?? 'e2e'
const TOKEN = process.env.OZALID_E2E_TOKEN ?? ''

const anEdition = { cases: [{ id: 'whatever', steps: [] }] }

async function push(headers: Record<string, string>) {
  return fetch(`${API}/api/projects/${PROJECT}/editions`, {
    method: 'POST',
    headers: { 'content-type': 'application/json', ...headers },
    body: JSON.stringify(anEdition),
  })
}

test('pushing evidence with no token is refused', async () => {
  const response = await push({})
  expect(response.status).toBe(401)

  const problem = await response.json()
  // The refusal says what to do about it. One that only says "no" makes every
  // client author read the source.
  expect(problem.detail).toContain('Authorization: Bearer')
})

test('a token nobody knows is refused the same way as no token at all', async () => {
  // Telling them apart would let a caller learn which tokens exist by watching
  // how the server replies.
  const unknown = await push({ authorization: 'Bearer ozp_thisonewasneverminted' })
  expect(unknown.status).toBe(401)

  const malformed = await push({ authorization: TOKEN })
  expect(malformed.status).toBe(401)
})

test('a token reaching outside its project is refused, and not the same way', async () => {
  // Distinct from 401 on purpose: one is fixed by presenting a credential, the
  // other by being granted something (ADR 0018).
  const elsewhere = await fetch(`${API}/api/projects/somewhere-else/editions`, {
    method: 'POST',
    headers: { 'content-type': 'application/json', authorization: `Bearer ${TOKEN}` },
    body: JSON.stringify(anEdition),
  })
  expect(elsewhere.status).toBe(403)
})

test('reading a project needs a credential, like everything else', async () => {
  // This asserted the opposite until #55: every endpoint was open, because the
  // web client had no way to prove itself. It does now, so nothing is public —
  // reading included (product.md §8.1).
  const anonymous = await fetch(`${API}/api/projects/${PROJECT}/cases`)
  expect(anonymous.status).toBe(401)

  // And the same read works with the suite's token, so the refusal above is
  // about the credential and not about the address.
  const withToken = await fetch(`${API}/api/projects/${PROJECT}/cases`, {
    headers: { authorization: `Bearer ${TOKEN}` },
  })
  expect(withToken.status).toBe(200)
})
