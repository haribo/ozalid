/**
 * The frugal way of pushing evidence, walked end to end.
 *
 * Push the manifest first, be told exactly which bytes are missing, upload
 * those, push again. It is what lets a suite of two hundred captures send three
 * images when three moved — and it returned a 500 until #64, which is why it is
 * watched here rather than trusted.
 */
import { expect, test } from '@playwright/test'
import { createHash } from 'node:crypto'

const API = process.env.OZALID_API ?? 'http://localhost:8091'
const PROJECT = process.env.OZALID_E2E_PROJECT ?? 'e2e'
const TOKEN = process.env.OZALID_E2E_TOKEN ?? ''

const auth = { authorization: `Bearer ${TOKEN}` }

async function api(path: string, init?: RequestInit) {
  return fetch(`${API}/api${path}`, {
    ...init,
    headers: { ...init?.headers, ...auth },
  })
}

test('a client is told what to upload, uploads it, and is accepted', async ({ page }) => {
  const created = await api(`/projects/${PROJECT}/cases`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ title: `frugal — ${Date.now()}` }),
  })
  const kase = (await created.json()) as { id: string }

  // A capture nothing has ever stored: painted here, uploaded nowhere yet.
  await page.setContent(
    `<!doctype html><body style="margin:0;width:80px;height:40px;background:#${Date.now().toString(16).slice(-6)}">`,
  )
  const bytes = await page.screenshot({ clip: { x: 0, y: 0, width: 80, height: 40 } })
  const hash = `sha256:${createHash('sha256').update(bytes).digest('hex')}`

  const manifest = {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      cases: [
        {
          id: kase.id,
          steps: [
            {
              name: 'opens',
              captures: [
                { variant: { theme: 'light' }, hash, provenance: { environmentId: 'ci' } },
              ],
            },
          ],
        },
      ],
    }),
  }

  // 1. Pushed before anything was uploaded: refused, and the refusal names it.
  const refused = await api(`/projects/${PROJECT}/editions`, manifest)
  expect(refused.status).toBe(409)
  const problem = await refused.json()
  expect(problem.missingContent).toEqual([hash])

  // 2. Upload exactly what was named.
  const uploaded = await api(`/projects/${PROJECT}/blobs/${hash}`, {
    method: 'PUT',
    body: new Uint8Array(bytes),
  })
  expect(uploaded.status).toBe(201)

  // 3. The same manifest, unchanged, is now accepted.
  const accepted = await api(`/projects/${PROJECT}/editions`, manifest)
  expect(accepted.status).toBe(201)

  // 4. And on a run where nothing changed, the first push already succeeds:
  //    no bytes move at all.
  const again = await api(`/projects/${PROJECT}/editions`, manifest)
  expect(again.status).toBe(201)
})
