/**
 * Pushing ozalid's own screens into ozalid.
 *
 * The same flow `docs/pushing-evidence.md` describes for any client: hash the
 * bytes, upload what the store does not hold, then push one manifest naming
 * them. Nothing here is special because the client happens to be this project.
 */
import { createHash } from 'node:crypto'
import { expect } from '@playwright/test'

/** Where the captures go, and what proves who is pushing them. */
export const PUSH_API = process.env.OZALID_PUSH_API ?? ''
const TOKEN = process.env.OZALID_PUSH_TOKEN ?? ''
const PROJECT = process.env.OZALID_PUSH_PROJECT ?? 'ozalid'

/**
 * Where these captures were taken.
 *
 * Captures from different environments are never compared silently (ADR 0004),
 * so a run on a laptop must not be measured against one from CI.
 */
const ENVIRONMENT = process.env.GITHUB_ACTIONS ? 'github-actions' : 'local'

/** Whether this run has somewhere to push. Skipped rather than failed when not. */
export const pushes = Boolean(PUSH_API && TOKEN)

async function call(path: string, init?: RequestInit) {
  const where = `${init?.method ?? 'GET'} ${PUSH_API}/api${path}`
  let response: Response
  try {
    response = await fetch(`${PUSH_API}/api${path}`, {
      ...init,
      headers: { ...init?.headers, authorization: `Bearer ${TOKEN}` },
    })
  } catch (cause) {
    // `fetch failed` on its own says nothing about what was unreachable, and
    // this runs in CI where nobody can retry it by hand.
    throw new Error(`${where} — could not be reached: ${(cause as Error).message}`, { cause })
  }
  if (!response.ok) {
    // 401 here usually means the Authorization header never arrived as sent:
    // basic auth in front of the instance claims the same header a token needs.
    throw new Error(`${where} — ${response.status} ${await response.text()}`)
  }
  return response
}

/** The address a capture is stored under — its content, and nothing else. */
export const hashOf = (bytes: Buffer) =>
  `sha256:${createHash('sha256').update(bytes).digest('hex')}`

/** One screen, in one variant. */
export type Shot = { step: string; variant: Record<string, string>; bytes: Buffer }

/** The flow video of one variant. Never compared, never byte-stable — new
 * bytes on every pushing run is by design (ADR 0013). */
export type Recording = { variant: Record<string, string>; bytes: Buffer }

/**
 * The case these captures belong to, found by its title or made.
 *
 * Found rather than made every time: an edition is a new reading of the *same*
 * case, and a fresh case each run would leave a book full of one-run cases with
 * nothing to compare against.
 */
async function caseFor(title: string, category: string): Promise<string> {
  const listed = (await (await call(`/projects/${PROJECT}/cases`)).json()) as {
    id: string
    title: string
  }[]
  const found = listed.find((c) => c.title === title)
  if (found) return found.id

  const made = (await (
    await call(`/projects/${PROJECT}/cases`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ title, categoryId: await categoryFor(category) }),
    })
  ).json()) as { id: string }
  return made.id
}

/**
 * The branch a flow's captures hang from: named by the flow itself (#245) —
 * a case belongs to exactly one category, and "where it belongs" is what the
 * flow knows, not a constant of the pusher. Found or made at the root.
 */

async function categoryFor(name: string): Promise<string> {
  const tree = (await (await call(`/projects/${PROJECT}/categories`)).json()) as {
    id: string
    name: string
    parentId: string | null
  }[]
  const found = tree.find((c) => c.name === name && !c.parentId)
  if (found) return found.id

  const made = (await (
    await call(`/projects/${PROJECT}/categories`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ name }),
    })
  ).json()) as { id: string }
  return made.id
}

/** Upload what the store does not already hold, and push one edition. */
export async function push(
  title: string,
  shots: Shot[],
  recordings: Recording[] = [],
  category = 'account',
) {
  expect(shots.length, 'nothing was captured').toBeGreaterThan(0)
  const caseId = await caseFor(title, category)

  // One step per screen, one capture per variant. The order of the steps is the
  // order they were walked, which is what makes the grid read as the flow.
  const steps: { name: string; captures: unknown[] }[] = []
  for (const shot of shots) {
    let step = steps.find((s) => s.name === shot.step)
    if (!step) {
      step = { name: shot.step, captures: [] }
      steps.push(step)
    }
    step.captures.push({
      variant: shot.variant,
      hash: hashOf(shot.bytes),
      provenance: { environmentId: ENVIRONMENT },
    })
  }

  // The frugal way (#186, docs/pushing-evidence.md): push the manifest first,
  // and the refusal names every missing address in one response — no
  // capture-by-capture HEAD round-trips. On a run where nothing changed, the
  // first push simply succeeds and no bytes move at all.
  const manifest = JSON.stringify({
    cases: [
      {
        id: caseId,
        steps,
        ...(recordings.length
          ? { recordings: recordings.map((r) => ({ variant: r.variant, hash: hashOf(r.bytes) })) }
          : {}),
      },
    ],
  })

  // Until #223 the server names the missing captures and the missing
  // recordings in two separate refusals, so the loop uploads whatever each
  // one lists and pushes again. Three rounds cover both worlds; a fourth
  // refusal is a real error.
  for (let round = 0; ; round++) {
    const attempt = await fetch(`${PUSH_API}/api/projects/${PROJECT}/editions`, {
      method: 'POST',
      headers: { 'content-type': 'application/json', authorization: `Bearer ${TOKEN}` },
      body: manifest,
    })
    if (attempt.ok) break
    const refusalBody = (await attempt.json()) as { type?: string; missingContent?: string[] }
    if (!String(refusalBody.type).includes('missing-content') || round >= 2) {
      throw new Error(`POST /editions — ${attempt.status} ${JSON.stringify(refusalBody)}`)
    }

    const missing = new Set(refusalBody.missingContent ?? [])
    for (const { bytes } of [...shots, ...recordings]) {
      if (!missing.has(hashOf(bytes))) continue
      await call(`/projects/${PROJECT}/blobs/${hashOf(bytes)}`, {
        method: 'PUT',
        body: new Uint8Array(bytes),
      })
    }
  }

  // The branch loop (#175): pushing with OZALID_PUSH_DELIVER=1 is the claim
  // "this edition answers your remarks" — every ref-less draft on the pushed
  // case is delivered in the same breath, and the case advances onto these
  // very bytes. Off by default: an edition arriving proves nothing by itself
  // (product.md §7), the claim stays explicit.
  if (process.env.OZALID_PUSH_DELIVER) {
    const said = (await (await call(`/projects/${PROJECT}/cases/${caseId}/comments`)).json()) as {
      id: string
      state: string
      issues?: unknown[]
    }[]
    for (const remark of said) {
      if (remark.state === 'to-track' && (remark.issues ?? []).length === 0) {
        await call(`/projects/${PROJECT}/comments/${remark.id}/delivery`, {
          method: 'POST',
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify({}),
        })
      }
    }
  }
}
