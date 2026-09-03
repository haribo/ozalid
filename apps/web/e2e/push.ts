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

/**
 * The case these captures belong to, found by its title or made.
 *
 * Found rather than made every time: an edition is a new reading of the *same*
 * case, and a fresh case each run would leave a book full of one-run cases with
 * nothing to compare against.
 */
async function caseFor(title: string): Promise<string> {
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
      body: JSON.stringify({ title, categoryId: await categoryFor(CATEGORY) }),
    })
  ).json()) as { id: string }
  return made.id
}

/**
 * The branch these captures hang from.
 *
 * A case belongs to exactly one category, and the catalogue only lists the
 * cases of a named one — a case filed nowhere is a case no screen can show
 * (#115). Named here rather than configured: these are ozalid's own screens,
 * and where they belong is not a deployment decision.
 */
const CATEGORY = 'ozalid'

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
export async function push(title: string, shots: Shot[]) {
  expect(shots.length, 'nothing was captured').toBeGreaterThan(0)
  const caseId = await caseFor(title)

  for (const shot of shots) {
    const hash = hashOf(shot.bytes)
    const held = await fetch(`${PUSH_API}/api/projects/${PROJECT}/blobs/${hash}`, {
      method: 'HEAD',
      headers: { authorization: `Bearer ${TOKEN}` },
    })
    if (held.status === 404) {
      await call(`/projects/${PROJECT}/blobs/${hash}`, {
        method: 'PUT',
        body: new Uint8Array(shot.bytes),
      })
    }
  }

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

  await call(`/projects/${PROJECT}/editions`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ cases: [{ id: caseId, steps }] }),
  })
}
