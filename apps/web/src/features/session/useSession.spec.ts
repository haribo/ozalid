import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/shared/api'
import { useSession } from './useSession'

type Answer = { status: number; body?: unknown }

const answers = new Map<string, Answer>()

/** What the server says for one path, for the length of one test. */
function serverSays(path: string, answer: Answer) {
  answers.set(path, answer)
}

beforeEach(() => {
  answers.clear()
  vi.stubGlobal('fetch', async (request: Request) => {
    const path = new URL(request.url).pathname.replace(/^\/api/, '')
    const answer = answers.get(path) ?? { status: 404, body: problem(404) }
    if (answer.body === undefined) return new Response(null, { status: answer.status })
    return new Response(JSON.stringify(answer.body), {
      status: answer.status,
      headers: { 'content-type': 'application/json' },
    })
  })
})

function problem(status: number) {
  return { type: 'about:blank', title: 'refused', status }
}

const nina = { id: 'u1', name: 'nina', email: 'nina@exemple.fr', isAdmin: false }

describe('useSession', () => {
  it('reads who the browser is', async () => {
    serverSays('/me', { status: 200, body: nina })
    const { refresh, person, standing } = useSession()

    await refresh()

    expect(standing.value).toBe('in')
    expect(person.value?.name).toBe('nina')
  })

  it('treats a refused /me as signed out, never as a session that died', async () => {
    serverSays('/me', { status: 401, body: problem(401) })
    const { refresh, person, standing } = useSession()

    await refresh()

    expect(standing.value).toBe('out')
    expect(person.value).toBeNull()
  })

  it('calls a session expired when a signed-in browser is refused elsewhere', async () => {
    serverSays('/me', { status: 200, body: nina })
    const { refresh, standing } = useSession()
    await refresh()

    // Keyed by the path the browser really asks for; the watcher is handed the
    // template, which is what tells it this endpoint is not exempt.
    serverSays('/cases/c1', { status: 401, body: problem(401) })
    await api.GET('/cases/{caseId}', { params: { path: { caseId: 'c1' } } })

    expect(standing.value).toBe('expired')
  })

  it('leaves a signed-out browser signed out when it is refused', async () => {
    serverSays('/me', { status: 401, body: problem(401) })
    const { refresh, standing } = useSession()
    await refresh()

    serverSays('/cases/c1', { status: 401, body: problem(401) })
    await api.GET('/cases/{caseId}', { params: { path: { caseId: 'c1' } } })

    expect(standing.value).toBe('out')
  })

  it('says a spent link failed without changing what the browser is', async () => {
    serverSays('/me', { status: 401, body: problem(401) })
    const { refresh, claim, standing } = useSession()
    await refresh()

    serverSays('/sign-in/claim', { status: 401, body: problem(401) })

    await expect(claim('deadbeef')).resolves.toBe(false)
    expect(standing.value).toBe('out')
  })

  it('reports the link as sent whether or not the address is known', async () => {
    serverSays('/sign-in', { status: 202 })
    const { requestLink, linkSent, error } = useSession()

    await requestLink('inconnu@exemple.fr')

    expect(linkSent.value).toBe(true)
    expect(error.value).toBe('')
  })
})
