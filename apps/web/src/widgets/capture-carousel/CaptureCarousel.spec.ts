import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CaptureCarousel from './CaptureCarousel.vue'
import type { components } from '@/shared/api'

type Grid = components['schemas']['Grid']
type Comment = components['schemas']['Comment']

const grid: Grid = {
  caseId: 'c1',
  variants: [
    { id: 'v1', label: 'desktop·light', values: { viewport: 'desktop', theme: 'light' } },
    { id: 'v2', label: 'desktop·dark', values: { viewport: 'desktop', theme: 'dark' } },
  ],
  steps: [
    {
      id: 's1',
      name: 'ouvre le lien',
      position: 0,
      cells: [
        { variantId: 'v1', hash: 'sha256:a', status: 'to-review' },
        { variantId: 'v2', hash: 'sha256:b', status: 'validated' },
      ],
    },
  ],
  recordings: [],
}

function mountAt(variantId: string, comments: Comment[] = []) {
  return mount(CaptureCarousel, {
    props: { grid, comments, stepId: 's1', variantId },
  })
}

describe('CaptureCarousel', () => {
  it('says where it is in the walk', () => {
    expect(mountAt('v1').text()).toContain('1 / 2')
    expect(mountAt('v2').text()).toContain('2 / 2')
  })

  it('steps a judged capture back and stamps its verdict on it', () => {
    // Same reading as the grid, at full size: what keeps full intensity is what
    // still needs looking at.
    const judged = mountAt('v2')
    expect(judged.find('img').classes()).toContain('opacity-40')
    expect(judged.find('[aria-label="validée"]').exists()).toBe(true)

    const pending = mountAt('v1')
    expect(pending.find('img').classes()).not.toContain('opacity-40')
    expect(pending.find('[aria-label="validée"]').exists()).toBe(false)
  })

  it('validates on space, and asks the server rather than deciding alone', () => {
    const w = mountAt('v1')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: ' ' }))
    expect(w.emitted('validate')?.[0]).toEqual(['s1', 'v1'])
  })

  it('does not offer to validate a square that already is', () => {
    const w = mountAt('v2')
    const validate = w.findAll('button').find((b) => b.text().includes('valider'))
    expect(validate?.attributes('disabled')).toBeDefined()
  })

  it('walks the grid with the arrows and leaves on Escape', () => {
    const w = mountAt('v1')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight' }))
    expect(w.emitted('move')?.[0]).toEqual(['s1', 'v2'])

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(w.emitted('close')).toHaveLength(1)
  })

  it('refuses to add a comment with no text', async () => {
    const w = mountAt('v1')
    await w
      .findAll('button')
      .find((b) => b.text().includes('commenter'))!
      .trigger('click')

    const add = w.findAll('button').find((b) => b.text() === 'ajouter')
    expect(add?.attributes('disabled')).toBeDefined()
  })

  it('ticks the variant on screen to start with — it is the one being looked at', async () => {
    const w = mountAt('v2')
    await w
      .findAll('button')
      .find((b) => b.text().includes('commenter'))!
      .trigger('click')
    expect(w.text()).toContain('1 variante cochée')
  })

  it('judges the fix rather than the capture once a delivery is waiting', () => {
    const delivered: Comment = {
      id: 'k1',
      stepId: 's1',
      kind: 'defect',
      body: "l'avatar est écrasé",
      state: 'to-review',
      variantIds: ['v1'],
      authorId: 'nina',
      createdAt: '2026-08-24T09:00:00Z',
      judgments: [],
      issue: { id: '139' },
    }
    const w = mountAt('v1', [delivered])

    expect(w.text()).toContain('correction livrée')
    expect(w.text()).toContain('issue 139')
    // The two verdicts replace "validate": it is no longer the capture being
    // judged, it is the fix.
    expect(w.findAll('button').some((b) => b.text().includes('valider'))).toBe(false)
    expect(w.findAll('button').some((b) => b.text().includes('accepter'))).toBe(true)
  })

  it('asks for the remark before letting a refusal through', async () => {
    const delivered: Comment = {
      id: 'k1',
      stepId: 's1',
      kind: 'defect',
      body: 'x',
      state: 'to-review',
      variantIds: ['v1'],
      authorId: 'nina',
      createdAt: '2026-08-24T09:00:00Z',
      judgments: [],
    }
    const w = mountAt('v1', [delivered])

    const refuse = w.findAll('button').find((b) => b.text().includes('refuser'))!
    await refuse.trigger('click')
    // The first click opens the field; nothing is sent yet.
    expect(w.emitted('judge')).toBeUndefined()
    expect(w.find('textarea').exists()).toBe(true)

    await refuse.trigger('click')
    expect(w.emitted('judge')).toBeUndefined()
  })
})

describe('the group shortcuts', () => {
  it('are derived from the axes the project declared, never hard-coded', async () => {
    // A project with a role axis gets its values as shortcuts without anyone
    // writing them (ADR 0001).
    const withRole: Grid = {
      ...grid,
      variants: [
        { id: 'v1', label: 'admin·light', values: { role: 'admin', theme: 'light' } },
        { id: 'v2', label: 'member·dark', values: { role: 'member', theme: 'dark' } },
      ],
      steps: [
        {
          id: 's1',
          name: 'ouvre',
          position: 0,
          cells: [
            { variantId: 'v1', hash: 'sha256:a', status: 'to-review' },
            { variantId: 'v2', hash: 'sha256:b', status: 'to-review' },
          ],
        },
      ],
    }
    const w = mount(CaptureCarousel, {
      props: { grid: withRole, comments: [], stepId: 's1', variantId: 'v1' },
    })
    await w
      .findAll('button')
      .find((b) => b.text().includes('commenter'))!
      .trigger('click')

    const labels = w.findAll('button').map((b) => b.text())
    expect(labels).toContain('admin')
    expect(labels).toContain('member')
    expect(labels).toContain('light')
    expect(labels).toContain('tout')
  })

  it('ticks every variant sharing a value in one click', async () => {
    const w = mountAt('v1')
    await w
      .findAll('button')
      .find((b) => b.text().includes('commenter'))!
      .trigger('click')
    await w
      .findAll('button')
      .find((b) => b.text() === 'desktop')!
      .trigger('click')
    expect(w.text()).toContain('2 variantes cochées')
  })
})
