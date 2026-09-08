/**
 * One comment, several issues, each on its own round (#138).
 *
 * Through the HTTP API, against a real server: the moves the carousel makes
 * are these calls, and the 400 on ambiguity is what keeps the server from
 * guessing which fix was delivered.
 */
import { expect, test } from '@playwright/test'
import { seed } from './fixture'

const API = process.env.OZALID_API ?? 'http://localhost:8091'

test('a comment splits into two issues, and each lives its own round', async ({
  page,
  request,
}) => {
  const seeded = await seed(page)
  const grid = await (
    await request.get(`${API}/api/projects/${seeded.slug}/cases/${seeded.caseId}/captures`)
  ).json()
  const step = grid.steps[0]

  // The reviewer files one comment...
  await request.post(`${API}/api/projects/${seeded.slug}/cases/${seeded.caseId}/reviews`, {
    data: {
      comments: [
        {
          stepId: step.id,
          body: 'two problems in one breath',
          variantIds: [step.captures[0].variantId],
        },
      ],
    },
  })
  const comments = await (
    await request.get(`${API}/api/projects/${seeded.slug}/cases/${seeded.caseId}/comments`)
  ).json()
  const comment = comments.find((c: { body: string }) => c.body === 'two problems in one breath')

  // ...which the triage splits into two issues.
  for (const [id, title] of [
    ['201', 'first half'],
    ['202', 'second half'],
  ]) {
    expect(
      (
        await request.post(`${API}/api/projects/${seeded.slug}/comments/${comment.id}/reference`, {
          data: { id, title, url: `https://example.test/${id}` },
        })
      ).status(),
    ).toBe(200)
  }

  const read = async () =>
    (
      await (
        await request.get(`${API}/api/projects/${seeded.slug}/cases/${seeded.caseId}/comments`)
      ).json()
    ).find((c: { id: string }) => c.id === comment.id)

  let c = await read()
  expect(c.issues).toHaveLength(2)
  expect(c.state).toBe('tracked')

  // Delivering without naming the ref is refused: the server will not guess.
  const vague = await request.post(
    `${API}/api/projects/${seeded.slug}/comments/${comment.id}/delivery`,
    { data: {} },
  )
  expect(vague.status()).toBe(400)

  // Delivering the first, judging it — the second never moves.
  const first = c.issues[0].id
  expect(
    (
      await request.post(`${API}/api/projects/${seeded.slug}/comments/${comment.id}/delivery`, {
        data: { issueId: first },
      })
    ).status(),
  ).toBe(200)
  c = await read()
  expect(c.state).toBe('to-review')

  expect(
    (
      await request.post(`${API}/api/projects/${seeded.slug}/comments/${comment.id}/judgment`, {
        data: { accept: true, issueId: first, variantId: step.captures[0].variantId },
      })
    ).status(),
  ).toBe(200)
  c = await read()
  expect(c.issues.find((r: { id: string }) => r.id === first).state).toBe('accepted')
  expect(c.issues.find((r: { id: string }) => r.id !== first).state).toBe('tracked')

  // One half judged is not a settled comment: the other is undelivered.
  expect(c.state).toBe('tracked')
})

/**
 * A judgment lands on the capture on screen (ADR 0022, #208): one remark over
 * both variants, and each variant gets its own verdict. Accepting on one
 * releases only that coverage; refusing on the other opens a partial round;
 * accepting the last one settles the remark.
 */
test('a fix is judged variant by variant, and the last acceptance settles it', async ({
  page,
  request,
}) => {
  const seeded = await seed(page)
  const grid = await (
    await request.get(`${API}/api/projects/${seeded.slug}/cases/${seeded.caseId}/captures`)
  ).json()
  const step = grid.steps[0]
  const [light, dark] = step.captures.map((c: { variantId: string }) => c.variantId)

  await request.post(`${API}/api/projects/${seeded.slug}/cases/${seeded.caseId}/reviews`, {
    data: {
      comments: [{ stepId: step.id, body: 'the label overflows', variantIds: [light, dark] }],
    },
  })
  const comments = await (
    await request.get(`${API}/api/projects/${seeded.slug}/cases/${seeded.caseId}/comments`)
  ).json()
  const comment = comments.find((c: { body: string }) => c.body === 'the label overflows')

  await request.post(`${API}/api/projects/${seeded.slug}/comments/${comment.id}/reference`, {
    data: { id: '173', title: 'let the floating label sit on the border' },
  })
  await request.post(`${API}/api/projects/${seeded.slug}/comments/${comment.id}/delivery`, {
    data: {},
  })

  const read = async () =>
    (
      await (
        await request.get(`${API}/api/projects/${seeded.slug}/cases/${seeded.caseId}/comments`)
      ).json()
    ).find((c: { id: string }) => c.id === comment.id)
  const statuses = async () => {
    const g = await (
      await request.get(`${API}/api/projects/${seeded.slug}/cases/${seeded.caseId}/captures`)
    ).json()
    return Object.fromEntries(
      g.steps[0].captures.map((c: { variantId: string; status: string }) => [
        c.variantId,
        c.status,
      ]),
    )
  }

  // Accepting on the light variant releases it — and only it.
  expect(
    (
      await request.post(`${API}/api/projects/${seeded.slug}/comments/${comment.id}/judgment`, {
        data: { accept: true, variantId: light },
      })
    ).status(),
  ).toBe(200)
  let c = await read()
  expect(c.variantIds).toEqual([dark])
  expect(c.issues[0].state).toBe('to-review')
  expect((await statuses())[light]).toBe('accepted')

  // Refusing on the dark variant opens a partial round on what is left.
  expect(
    (
      await request.post(`${API}/api/projects/${seeded.slug}/comments/${comment.id}/judgment`, {
        data: { accept: false, remark: 'still overflows on dark', variantId: dark },
      })
    ).status(),
  ).toBe(200)
  c = await read()
  expect(c.variantIds).toEqual([dark])
  expect(c.issues[0].state).toBe('refused')
  expect((await statuses())[light]).toBe('accepted')

  // The next delivery is judged on the remaining coverage; accepting the
  // last variant settles the remark and the case reads reviewed.
  await request.post(`${API}/api/projects/${seeded.slug}/comments/${comment.id}/delivery`, {
    data: {},
  })
  expect(
    (
      await request.post(`${API}/api/projects/${seeded.slug}/comments/${comment.id}/judgment`, {
        data: { accept: true, variantId: dark },
      })
    ).status(),
  ).toBe(200)
  c = await read()
  expect(c.state).toBe('accepted')
  expect(c.issues[0].state).toBe('accepted')
  expect(await statuses()).toEqual({ [light]: 'accepted', [dark]: 'accepted' })
})
