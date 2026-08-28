import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useReview } from './useReview'

type Answer = { status: number; body?: unknown }

const answers = new Map<string, Answer>()
const sent: { path: string; body: unknown }[] = []

function serverSays(path: string, answer: Answer) {
  answers.set(path, answer)
}

function problem(status: number) {
  return { type: 'about:blank', title: 'refused', status }
}

const grid = {
  caseId: 'c1',
  variants: [],
  steps: [],
  recordings: [],
}

beforeEach(() => {
  answers.clear()
  sent.length = 0
  serverSays('/cases/c1/captures', { status: 200, body: grid })
  serverSays('/cases/c1/comments', { status: 200, body: [] })

  vi.stubGlobal('fetch', async (request: Request) => {
    const path = new URL(request.url).pathname.replace(/^\/api/, '')
    if (request.method === 'POST') sent.push({ path, body: await request.clone().json() })
    const answer = answers.get(path) ?? { status: 404, body: problem(404) }
    if (answer.body === undefined) return new Response(null, { status: answer.status })
    return new Response(JSON.stringify(answer.body), {
      status: answer.status,
      headers: { 'content-type': 'application/json' },
    })
  })
})

describe('useReview, when the session dies mid-review', () => {
  it('holds the verdict it could not send, and sends it once the session is back', async () => {
    const review = useReview(() => 'c1')
    serverSays('/cases/c1/reviews', { status: 401, body: problem(401) })

    await review.validate('s1', 'v1')

    expect(review.held.value).not.toBeNull()
    // The bar in the shell says the session expired; a second message here
    // would say the same thing in other words.
    expect(review.error.value).toBe('')

    serverSays('/cases/c1/reviews', { status: 204 })
    await review.resume()

    expect(review.held.value).toBeNull()
    expect(sent.filter((call) => call.path === '/cases/c1/reviews')).toHaveLength(2)
    expect(sent.at(-1)?.body).toEqual({ validated: [{ stepId: 's1', variantId: 'v1' }] })
  })

  it('holds a judgment the same way', async () => {
    const review = useReview(() => 'c1')
    serverSays('/comments/k1/judgment', { status: 401, body: problem(401) })

    await review.judge('k1', true)
    expect(review.held.value).not.toBeNull()

    serverSays('/comments/k1/judgment', { status: 204 })
    await review.resume()

    expect(review.held.value).toBeNull()
    expect(sent.filter((call) => call.path === '/comments/k1/judgment')).toHaveLength(2)
  })

  it('holds nothing when the refusal is not about the session', async () => {
    const review = useReview(() => 'c1')
    serverSays('/cases/c1/reviews', { status: 500, body: problem(500) })

    await review.validate('s1', 'v1')

    expect(review.held.value).toBeNull()
    expect(review.error.value).toBe('refused')
  })

  it('does nothing when asked to resume with nothing held', async () => {
    const review = useReview(() => 'c1')

    await review.resume()

    expect(sent).toHaveLength(0)
  })
})
