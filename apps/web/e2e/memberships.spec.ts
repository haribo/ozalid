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

async function anAccount(request: {
  post: (url: string, o: object) => Promise<{ json: () => Promise<{ id: string }> }>
}) {
  const made = await request.post(`${API}/api/accounts`, {
    data: { name: `hired ${unique()}`, email: `member-${unique()}@example.test` },
  })
  return (await made.json()).id
}

test('a person is granted, listed, promoted and taken off again', async ({ request }) => {
  const accountId = await anAccount(request)

  expect(
    (
      await request.put(`${API}/api/projects/${PROJECT}/members/${accountId}`, {
        data: { rights: 'reader' },
      })
    ).status(),
  ).toBe(204)

  const listed = await request.get(`${API}/api/projects/${PROJECT}/members`)
  expect(listed.status()).toBe(200)
  const entry = (await listed.json()).find((m: { accountId: string }) => m.accountId === accountId)
  expect(entry, 'the person was granted but is not listed').toBeTruthy()
  expect(entry.rights).toBe('reader')
  expect(entry.isPerson).toBe(true)

  // Granting again changes the rights rather than failing: revoking first would
  // be two calls, and forgetting the second is how somebody loses their access.
  expect(
    (
      await request.put(`${API}/api/projects/${PROJECT}/members/${accountId}`, {
        data: { rights: 'member' },
      })
    ).status(),
  ).toBe(204)
  const promoted = await request.get(`${API}/api/projects/${PROJECT}/members`)
  expect(
    (await promoted.json()).find((m: { accountId: string }) => m.accountId === accountId).rights,
  ).toBe('member')

  expect(
    (await request.delete(`${API}/api/projects/${PROJECT}/members/${accountId}`)).status(),
  ).toBe(204)
  // Idempotent: they wanted them off, and they are.
  expect(
    (await request.delete(`${API}/api/projects/${PROJECT}/members/${accountId}`)).status(),
  ).toBe(204)

  const after = await request.get(`${API}/api/projects/${PROJECT}/members`)
  expect((await after.json()).some((m: { accountId: string }) => m.accountId === accountId)).toBe(
    false,
  )
})

test('the list holds programs as well as people', async ({ request }) => {
  // A service account holds a membership like anyone else (ADR 0019), and the
  // suite's own runner is one. Hiding it would make the list a lie about who
  // can see the book.
  const listed = await request.get(`${API}/api/projects/${PROJECT}/members`)
  const programs = (await listed.json()).filter((m: { isPerson: boolean }) => !m.isPerson)
  expect(
    programs.length,
    'no service account listed, though one pushes this suite',
  ).toBeGreaterThan(0)
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

test('an administrator reaches a project they created and never joined', async ({ request }) => {
  const slug = `outsiders-${unique()}`
  expect(
    (
      await request.post(`${API}/api/projects`, {
        data: { slug, name: 'a project its creator never joined' },
      })
    ).status(),
  ).toBe(201)

  // Creating it still does not write a membership row: what follows is
  // administration reaching past membership, not a grant somebody forgot.
  const members = await request.get(`${API}/api/projects/${slug}/members`)
  expect(members.status()).toBe(200)
  expect(await members.json()).toEqual([])

  // Read and write both, which is the decision: read alone would give screens
  // where everything shows and every button fails (product.md §8.2).
  expect((await request.get(`${API}/api/projects/${slug}/cases`)).status()).toBe(200)
  const made = await request.post(`${API}/api/projects/${slug}/cases`, {
    data: { title: 'written by somebody who is not a member' },
  })
  expect(made.status()).toBe(201)

  // And administering it still works, which never depended on membership.
  const accountId = await anAccount(request)
  expect(
    (
      await request.put(`${API}/api/projects/${slug}/members/${accountId}`, {
        data: { rights: 'member' },
      })
    ).status(),
  ).toBe(204)
})

test('a person sees the projects they belong to, and an admin sees them all', async ({
  request,
}) => {
  const slug = `unseen-${unique()}`
  await request.post(`${API}/api/projects`, { data: { slug, name: 'a team nobody is on' } })

  // The administrator belongs to the suite's project and not to this one, and
  // sees both — the names here, and what is inside them too (product.md §8.2).
  const mine = await request.get(`${API}/api/projects`)
  expect(mine.status()).toBe(200)
  const seen = (await mine.json()).map((p: { slug: string }) => p.slug)
  expect(seen).toContain(PROJECT)
  expect(seen).toContain(slug)
  expect((await request.get(`${API}/api/projects/${slug}/cases`)).status()).toBe(200)

  // The suite's token belongs to one project and sees that one.
  const asProgram = await fetch(`${API}/api/projects`, {
    headers: { authorization: `Bearer ${TOKEN}` },
  })
  const forProgram = (await asProgram.json()).map((p: { slug: string }) => p.slug)
  expect(forProgram).toEqual([PROJECT])

  expect((await fetch(`${API}/api/projects`)).status).toBe(401)
})

test('a deactivated account leaves the access list, and a program says how many keys it holds', async ({
  request,
}) => {
  const accountId = await anAccount(request)
  await request.put(`${API}/api/projects/${PROJECT}/members/${accountId}`, {
    data: { rights: 'member' },
  })

  const before = await (await request.get(`${API}/api/projects/${PROJECT}/members`)).json()
  expect(before.some((m: { accountId: string }) => m.accountId === accountId)).toBe(true)

  await request.delete(`${API}/api/accounts/${accountId}`)

  // This page answers "who reaches this project", and a deactivated account
  // reaches nothing. It stays on /accounts, which answers another question.
  const after = await (await request.get(`${API}/api/projects/${PROJECT}/members`)).json()
  expect(after.some((m: { accountId: string }) => m.accountId === accountId)).toBe(false)
  const onAccounts = await (await request.get(`${API}/api/accounts`)).json()
  expect(onAccounts.some((a: { id: string }) => a.id === accountId)).toBe(true)

  // A person carries no token count; a program carries one, and zero is the
  // value worth seeing.
  for (const m of after) {
    expect(m.isPerson ? m.tokens === undefined : typeof m.tokens === 'number').toBe(true)
  }
})

test('the add form says why there is nobody to add', async ({ page }) => {
  // An empty list is an answer, not a failure. Before this, the form opened on
  // a select holding only "choisir…" and there was no way to tell the two
  // apart — which is what a production instance with one account showed (#109).
  const slug = `alone-${unique()}`
  await page.request.post(`${API}/api/projects`, { data: { slug, name: 'nobody to add here' } })

  // Everybody who exists is already a member, so nobody is addable.
  const accounts = await (await page.request.get(`${API}/api/accounts`)).json()
  for (const a of accounts.filter((x: { deactivatedAt?: string }) => !x.deactivatedAt)) {
    await page.request.put(`${API}/api/projects/${slug}/members/${a.id}`, {
      data: { rights: 'member' },
    })
  }

  await page.goto(`/projects/${slug}/access`)
  await page.getByRole('button', { name: 'Ajouter' }).first().click()

  await expect(
    page.getByText("Tous les comptes de l'instance sont déjà sur ce projet."),
  ).toBeVisible()
  await expect(page.getByRole('link', { name: 'Créer un compte' })).toBeVisible()
  await expect(page.getByLabel('compte')).toHaveCount(0)
})
