/**
 * Who reaches a project, and who decides.
 *
 * The test that matters here is the last one: an administrator who grants a
 * membership on a project must still not be able to read it. That separation is
 * what lets the running of an instance be handed to somebody without handing
 * them every team's work (`product.md` §8.2), and it is worth nothing unless
 * something checks it.
 */
import { expect, test } from '@playwright/test'

const API = process.env.OZALID_API ?? 'http://localhost:8091'
const PROJECT = process.env.OZALID_E2E_PROJECT ?? 'e2e'
const TOKEN = process.env.OZALID_E2E_TOKEN ?? ''

const unique = () => `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`

async function anAccount(request: { post: (url: string, o: object) => Promise<{ json: () => Promise<{ id: string }> }> }) {
  const made = await request.post(`${API}/api/accounts`, {
    data: { name: `hired ${unique()}`, email: `member-${unique()}@example.test` },
  })
  return (await made.json()).id
}

test('a person is granted, listed, promoted and taken off again', async ({ request }) => {
  const accountId = await anAccount(request)

  expect((await request.put(`${API}/api/projects/${PROJECT}/members/${accountId}`, {
    data: { rights: 'reader' },
  })).status()).toBe(204)

  const listed = await request.get(`${API}/api/projects/${PROJECT}/members`)
  expect(listed.status()).toBe(200)
  const entry = (await listed.json()).find((m: { accountId: string }) => m.accountId === accountId)
  expect(entry, 'the person was granted but is not listed').toBeTruthy()
  expect(entry.rights).toBe('reader')
  expect(entry.isPerson).toBe(true)

  // Granting again changes the rights rather than failing: revoking first would
  // be two calls, and forgetting the second is how somebody loses their access.
  expect((await request.put(`${API}/api/projects/${PROJECT}/members/${accountId}`, {
    data: { rights: 'member' },
  })).status()).toBe(204)
  const promoted = await request.get(`${API}/api/projects/${PROJECT}/members`)
  expect((await promoted.json()).find((m: { accountId: string }) => m.accountId === accountId).rights).toBe('member')

  expect((await request.delete(`${API}/api/projects/${PROJECT}/members/${accountId}`)).status()).toBe(204)
  // Idempotent: they wanted them off, and they are.
  expect((await request.delete(`${API}/api/projects/${PROJECT}/members/${accountId}`)).status()).toBe(204)

  const after = await request.get(`${API}/api/projects/${PROJECT}/members`)
  expect((await after.json()).some((m: { accountId: string }) => m.accountId === accountId)).toBe(false)
})

test('the list holds programs as well as people', async ({ request }) => {
  // A service account holds a membership like anyone else (ADR 0019), and the
  // suite's own runner is one. Hiding it would make the list a lie about who
  // can see the book.
  const listed = await request.get(`${API}/api/projects/${PROJECT}/members`)
  const programs = (await listed.json()).filter((m: { isPerson: boolean }) => !m.isPerson)
  expect(programs.length, 'no service account listed, though one pushes this suite').toBeGreaterThan(0)
  expect(programs[0].email).toBeUndefined()
})

test('rights that are not rights are refused', async ({ request }) => {
  const accountId = await anAccount(request)

  for (const rights of ['admin', 'reviewer', 'owner', '']) {
    const refused = await request.put(`${API}/api/projects/${PROJECT}/members/${accountId}`, {
      data: { rights },
    })
    expect(refused.status(), `rights: ${rights || '(empty)'}`).toBe(400)
  }
})

test('a project or a person nobody has answers the same way', async ({ request }) => {
  const accountId = await anAccount(request)

  const noProject = await request.put(`${API}/api/projects/nobody-has-this/members/${accountId}`, {
    data: { rights: 'member' },
  })
  const noPerson = await request.put(`${API}/api/projects/${PROJECT}/members/nobody-has-this`, {
    data: { rights: 'member' },
  })

  // Which of the two is missing is not the caller's business: telling them
  // apart would let somebody map an instance by watching which refusals differ.
  expect(noProject.status()).toBe(404)
  expect(noPerson.status()).toBe(404)
})

test('a service account grants nothing, however good its token', async () => {
  const refused = await fetch(`${API}/api/projects/${PROJECT}/members/whoever`, {
    method: 'PUT',
    headers: { 'content-type': 'application/json', authorization: `Bearer ${TOKEN}` },
    body: JSON.stringify({ rights: 'member' }),
  })
  // It holds `member` on this project, which is deliberately not enough:
  // granting belongs to administration (product.md §8.2).
  expect(refused.status).toBe(403)
})

test('an administrator grants on a project they still cannot read', async ({ request }) => {
  const slug = `outsiders-${unique()}`
  expect((await request.post(`${API}/api/projects`, {
    data: { slug, name: 'a team the administrator is not on' },
  })).status()).toBe(201)

  // Creating it does not join it. This is the line §8.2 turns on: an
  // administrator who could read every project would see every team's work.
  expect((await request.get(`${API}/api/projects/${slug}/cases`)).status()).toBe(403)
  expect((await request.get(`${API}/api/projects/${slug}/members`)).status()).toBe(403)

  // And they can still put somebody on it, which is the whole point of the
  // separation: administering is not reading.
  const accountId = await anAccount(request)
  expect((await request.put(`${API}/api/projects/${slug}/members/${accountId}`, {
    data: { rights: 'member' },
  })).status()).toBe(204)

  // Still not theirs to read, membership granted or not.
  expect((await request.get(`${API}/api/projects/${slug}/cases`)).status()).toBe(403)
})
