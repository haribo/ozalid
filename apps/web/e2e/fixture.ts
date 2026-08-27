/**
 * The evidence a run would have produced, pushed through the real API.
 *
 * The suite seeds the way a client does — upload the bytes, push a manifest —
 * because a fixture written straight into the database would prove the
 * interface reads what the test wrote, not what the product produces.
 *
 * It takes the simple way of `docs/pushing-evidence.md`: every capture is sent
 * on every run, including bytes the store already holds. A suite this size does
 * not care, and the frugal way — push first, be told what is missing, upload
 * only that — would put a retry loop in a fixture whose job is to be obvious.
 *
 * The captures are made by the browser the suite already runs. Hand-rolling a
 * PNG encoder here would be ninety lines proving that ninety lines encode a
 * PNG; a screenshot is a real one, produced the way real ones are.
 */
import { createHash } from 'node:crypto'
import type { Page } from '@playwright/test'

const API = process.env.OZALID_API ?? 'http://localhost:8091'

/**
 * The suite's own project, and the token that reaches it.
 *
 * A token belongs to exactly one project (ADR 0018), which is why the suite
 * works inside a single one and gives each test its own **case** rather than
 * its own project. That is also how a real client is shaped, so the suite
 * exercises the shape it is meant to watch.
 */
const PROJECT = process.env.OZALID_E2E_PROJECT ?? 'e2e'
const TOKEN = process.env.OZALID_E2E_TOKEN ?? ''

export type Seeded = { slug: string; caseId: string }

async function call(path: string, init?: RequestInit) {
  const response = await fetch(`${API}/api${path}`, {
    ...init,
    headers: { ...init?.headers, ...(TOKEN ? { authorization: `Bearer ${TOKEN}` } : {}) },
  })
  if (!response.ok) {
    throw new Error(`${init?.method ?? 'GET'} ${path} — ${response.status} ${await response.text()}`)
  }
  return response
}

const post = (body: unknown): RequestInit => ({
  method: 'POST',
  headers: { 'content-type': 'application/json' },
  body: JSON.stringify(body),
})

/** A screen the product might have, with its call to action at `left`. */
async function screen(page: Page, left: number): Promise<Buffer> {
  await page.setContent(`<!doctype html><style>
    body { margin:0; width:320px; height:200px; background:#f7f8fb; font:14px system-ui }
    .bar { height:26px; background:#fff; border-bottom:1px solid #e3e4ed }
    .card { margin:22px 20px; padding:16px; background:#fff; border:1px solid #e3e4ed }
    .line { height:8px; background:#dfe1ea; margin-bottom:8px }
    .cta { margin-left:${left}px; width:96px; height:26px; background:#3d3a9e }
  </style><div class="bar"></div><div class="card">
    <div class="line" style="width:60%"></div><div class="line" style="width:85%"></div>
    <div class="cta"></div></div>`)
  return page.screenshot({ clip: { x: 0, y: 0, width: 320, height: 200 } })
}

async function upload(bytes: Buffer): Promise<string> {
  const hash = `sha256:${createHash('sha256').update(bytes).digest('hex')}`
  await call(`/blobs/${hash}`, { method: 'PUT', body: new Uint8Array(bytes) })
  return hash
}

const STEPS = [
  'demande la réinitialisation',
  'ouvre le lien reçu par e-mail',
  'arrive sur son compte',
]
const LIGHT = { viewport: 'desktop', theme: 'light' }
const DARK = { viewport: 'desktop', theme: 'dark' }

function manifest(caseId: string, first: string, second: string) {
  return post({
    cases: [
      {
        id: caseId,
        steps: STEPS.map((name) => ({
          name,
          captures: [
            { variant: LIGHT, hash: first, provenance: { environmentId: 'ci' } },
            { variant: DARK, hash: second, provenance: { environmentId: 'ci' } },
          ],
        })),
      },
    ],
  })
}

/**
 * One case in the suite's project, with three steps and two variants, taken in
 * once.
 *
 * The title carries the clock so two runs never collide: this suite writes for
 * real, against a database it owns.
 */
export async function seed(page: Page): Promise<Seeded> {
  const kase = (await (
    await call(
      `/projects/${PROJECT}/cases`,
      post({ title: `réinitialiser un mot de passe oublié — ${Date.now()}` }),
    )
  ).json()) as { id: string }

  const still = await upload(await screen(page, 0))
  await call(`/projects/${PROJECT}/editions`, manifest(kase.id, still, still))
  return { slug: PROJECT, caseId: kase.id }
}

/** A second edition where the call to action slid on the dark variant only. */
export async function moveTheDarkVariant(page: Page, seeded: Seeded): Promise<void> {
  const still = await upload(await screen(page, 0))
  const moved = await upload(await screen(page, 40))
  await call(`/projects/${seeded.slug}/editions`, manifest(seeded.caseId, still, moved))
}

/** Report one defect over every variant of one step, through the API. */
export async function commentOnStep(seeded: Seeded, stepIndex: number, body: string): Promise<void> {
  const grid = (await (await call(`/cases/${seeded.caseId}/captures`)).json()) as {
    steps: { id: string; cells: { variantId: string }[] }[]
  }
  const step = grid.steps[stepIndex]
  await call(
    `/cases/${seeded.caseId}/reviews`,
    post({
      comments: [
        {
          stepId: step.id,
          kind: 'defect',
          body,
          variantIds: step.cells.map((c) => c.variantId),
        },
      ],
    }),
  )
}

/** Validate every square, the way a reviewer who had nothing to say would. */
export async function validateEverything(seeded: Seeded): Promise<void> {
  const grid = (await (await call(`/cases/${seeded.caseId}/captures`)).json()) as {
    steps: { id: string; cells: { variantId: string }[] }[]
  }
  await call(
    `/cases/${seeded.caseId}/reviews`,
    post({
      validated: grid.steps.flatMap((s) =>
        s.cells.map((c) => ({ stepId: s.id, variantId: c.variantId })),
      ),
    }),
  )
}
