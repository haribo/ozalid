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
      name: 'opens the link',
      position: 0,
      cells: [
        { id: 'cap1', variantId: 'v1', hash: 'sha256:a', status: 'to-review' },
        { id: 'cap2', variantId: 'v2', hash: 'sha256:b', status: 'validated' },
      ],
    },
  ],
  recordings: [],
}

function mountAt(variantId: string, comments: Comment[] = []) {
  return mount(CaptureCarousel, {
    props: { slug: 'atlas', grid, comments, stepId: 's1', variantId },
  })
}

describe('CaptureCarousel', () => {
  it('says which step it is on — the counter counts steps, not squares (#149)', () => {
    expect(mountAt('v1').text()).toContain('1 / 1')
    expect(mountAt('v2').text()).toContain('1 / 1')
  })

  it('steps a judged capture back and stamps its verdict on it', () => {
    // Same reading as the grid, at full size: what keeps full intensity is what
    // still needs looking at.
    const judged = mountAt('v2')
    expect(judged.find('img').classes()).toContain('opacity-40')
    expect(judged.find('[aria-label="validated"]').exists()).toBe(true)

    const pending = mountAt('v1')
    expect(pending.find('img').classes()).not.toContain('opacity-40')
    expect(pending.find('[aria-label="validated"]').exists()).toBe(false)
  })

  it('says on the image that it moved, and by how much', () => {
    // The reviewer may have walked here with the keyboard and never seen the
    // grid's mark. The pixel count is what makes the project's threshold
    // judgeable rather than guessed.
    const moved: Grid = {
      ...grid,
      steps: [
        {
          ...grid.steps[0],
          cells: [
            {
              id: 'cap6',
              variantId: 'v1',
              hash: 'sha256:a',
              status: 'validated',
              freshness: 'to-re-review',
              movedPixels: 143,
            },
            {
              id: 'cap3',
              variantId: 'v2',
              hash: 'sha256:b',
              status: 'validated',
              freshness: 'current',
            },
          ],
        },
      ],
    }
    const w = mount(CaptureCarousel, {
      props: { slug: 'atlas', grid: moved, comments: [], stepId: 's1', variantId: 'v1' },
    })
    expect(w.text()).toContain('moved')
    expect(w.text()).toContain('143 px')
    // Back to full strength: it needs eyes again, whatever its verdict says.
    expect(w.find('img').classes()).not.toContain('opacity-40')
  })

  it('validates on space, and asks the server rather than deciding alone', () => {
    const w = mountAt('v1')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: ' ' }))
    expect(w.emitted('validate')?.[0]).toEqual(['s1', 'v1'])
  })

  it('does not offer to validate a square that already is', () => {
    const w = mountAt('v2')
    const validate = w.findAll('button').find((b) => b.text().includes('validate'))
    expect(validate?.attributes('disabled')).toBeDefined()
  })

  it('changes variant on the vertical arrows and leaves on Escape', () => {
    const w = mountAt('v1')
    // One step: right goes nowhere (#149); down switches the lens.
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight' }))
    expect(w.emitted('move')).toBeUndefined()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown' }))
    expect(w.emitted('move')?.[0]).toEqual(['s1', 'v2'])

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(w.emitted('close')).toHaveLength(1)
  })

  it('refuses to add a comment with no text', async () => {
    const w = mountAt('v1')
    await w
      .findAll('button')
      .find((b) => b.text().includes('comment'))!
      .trigger('click')

    const add = w.findAll('button').find((b) => b.text() === 'add')
    expect(add?.attributes('disabled')).toBeDefined()
  })

  it('ticks the variant on screen to start with — it is the one being looked at', async () => {
    const w = mountAt('v2')
    await w
      .findAll('button')
      .find((b) => b.text().includes('comment'))!
      .trigger('click')
    expect(w.text()).toContain('1 variant ticked')
  })

  it('judges the fix rather than the capture once a delivery is waiting', () => {
    const delivered: Comment = {
      id: 'k1',
      stepId: 's1',
      kind: 'defect',
      body: 'the avatar is squashed',
      state: 'to-review',
      variantIds: ['v1'],
      authorId: 'nina',
      createdAt: '2026-08-24T09:00:00Z',
      judgments: [],
      issue: { id: '139' },
      issues: [{ id: 'ref1', issueId: '139', state: 'to-review' }],
    }
    const w = mountAt('v1', [delivered])

    expect(w.text()).toContain('fix delivered')
    expect(w.text()).toContain('issue 139')
    // The two verdicts replace "validate": it is no longer the capture being
    // judged, it is the fix.
    expect(w.findAll('button').some((b) => b.text().includes('validate'))).toBe(false)
    expect(w.findAll('button').some((b) => b.text().includes('accept'))).toBe(true)
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
      issues: [{ id: 'ref1', issueId: '139', state: 'to-review' }],
    }
    const w = mountAt('v1', [delivered])

    const refuse = w.findAll('button').find((b) => b.text().includes('refuse'))!
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
          name: 'opens',
          position: 0,
          cells: [
            { id: 'cap4', variantId: 'v1', hash: 'sha256:a', status: 'to-review' },
            { id: 'cap5', variantId: 'v2', hash: 'sha256:b', status: 'to-review' },
          ],
        },
      ],
    }
    const w = mount(CaptureCarousel, {
      props: { slug: 'atlas', grid: withRole, comments: [], stepId: 's1', variantId: 'v1' },
    })
    await w
      .findAll('button')
      .find((b) => b.text().includes('comment'))!
      .trigger('click')

    const labels = w.findAll('button').map((b) => b.text())
    expect(labels).toContain('admin')
    expect(labels).toContain('member')
    expect(labels).toContain('light')
    expect(labels).toContain('all')
  })

  it('ticks every variant sharing a value in one click', async () => {
    const w = mountAt('v1')
    await w
      .findAll('button')
      .find((b) => b.text().includes('comment'))!
      .trigger('click')
    await w
      .findAll('button')
      .find((b) => b.text() === 'desktop')!
      .trigger('click')
    expect(w.text()).toContain('2 variants ticked')
  })
})

describe('arrows walk the steps (#149)', () => {
  const three: Grid = {
    caseId: 'c1',
    variants: grid.variants,
    steps: [
      grid.steps[0],
      {
        id: 's2',
        name: 'types',
        position: 1,
        cells: [
          { id: 'cap3', variantId: 'v1', hash: 'sha256:c', status: 'to-review' },
          { id: 'cap4', variantId: 'v2', hash: 'sha256:d', status: 'to-review' },
        ],
      },
      {
        id: 's3',
        name: 'lands',
        position: 2,
        // v2 only: walking right at v1 skips this step.
        cells: [{ id: 'cap5', variantId: 'v2', hash: 'sha256:e', status: 'to-review' }],
      },
    ],
    recordings: [],
  }
  const at = (stepId: string, variantId: string) =>
    mount(CaptureCarousel, {
      props: { slug: 'atlas', grid: three, comments: [], stepId, variantId },
      attachTo: document.body,
    })

  const press = (key: string) => window.dispatchEvent(new KeyboardEvent('keydown', { key }))

  it('moves to the next step and keeps the variant', () => {
    const w = at('s1', 'v2')
    press('ArrowRight')
    expect(w.emitted('move')).toEqual([['s2', 'v2']])
    w.unmount()
  })

  it('skips a step that lacks the variant', () => {
    const w = at('s2', 'v1')
    // s3 has no v1: right goes nowhere rather than switching variant.
    press('ArrowRight')
    expect(w.emitted('move')).toBeUndefined()
    w.unmount()
  })

  it('counts steps, and follows the walk', () => {
    expect(at('s2', 'v2').text()).toContain('2 / 3')
    expect(at('s3', 'v2').text()).toContain('3 / 3')
  })
})
