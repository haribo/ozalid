import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CommentRecap from './CommentRecap.vue'
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
    { id: 's1', name: 'ouvre le lien', position: 0, cells: [] },
    { id: 's2', name: 'arrive sur son compte', position: 1, cells: [] },
  ],
  recordings: [],
}

function comment(over: Partial<Comment> = {}): Comment {
  return {
    id: 'k1',
    stepId: 's1',
    kind: 'defect',
    body: 'le bouton est coupé',
    state: 'to-track',
    variantIds: ['v2'],
    authorId: 'nina',
    createdAt: '2026-08-24T09:00:00Z',
    judgments: [],
    ...over,
  }
}

describe('CommentRecap', () => {
  it('shows nothing at all when nothing has been said', () => {
    const w = mount(CommentRecap, { props: { grid, comments: [] } })
    expect(w.find('table').exists()).toBe(false)
  })

  it('carries no control whatsoever', () => {
    // A state, not a command panel: judging happens in front of the capture,
    // and linking or delivering are API calls the dev makes.
    const w = mount(CommentRecap, { props: { grid, comments: [comment()] } })
    expect(w.find('button').exists()).toBe(false)
    expect(w.find('input').exists()).toBe(false)
  })

  it('ticks one column per variant the comment applies to', () => {
    // One defect over two variants is one row with two ticks, never two rows.
    const w = mount(CommentRecap, {
      props: { grid, comments: [comment({ variantIds: ['v1', 'v2'] })] },
    })
    expect(w.findAll('tbody tr')).toHaveLength(1)
    expect(w.findAll('tbody td svg[role="img"]').length).toBeGreaterThanOrEqual(2)
  })

  it("shows a refusal's remark, because that is what the dev must read", () => {
    const w = mount(CommentRecap, {
      props: {
        grid,
        comments: [
          comment({
            state: 'refused',
            judgments: [
              {
                verdict: 'refused',
                remark: 'toujours coupé sur iPhone SE',
                actorId: 'nina',
                at: '2026-08-24T10:00:00Z',
              },
            ],
          }),
        ],
      },
    })
    expect(w.text()).toContain('toujours coupé sur iPhone SE')
  })

  it('keeps a discarded comment visible, with its reason', () => {
    // Nothing is deleted: "who removed this, and why?" must have an answer.
    const w = mount(CommentRecap, {
      props: { grid, comments: [comment({ state: 'discarded', discardReason: 'intentionnel' })] },
    })
    expect(w.text()).toContain('intentionnel')
    expect(w.text()).toContain('discarded')
  })

  it('gathers a step\'s comments under one cell, in the grid\'s order', () => {
    // Read as a continuation of the grid, not as a separate list.
    const w = mount(CommentRecap, {
      props: {
        grid,
        comments: [
          comment({ id: 'k1', stepId: 's2' }),
          comment({ id: 'k2', stepId: 's1' }),
          comment({ id: 'k3', stepId: 's1' }),
        ],
      },
    })
    const steps = w.findAll('tbody td[rowspan]')
    expect(steps.map((s) => s.text())).toEqual(['ouvre le lien', 'arrive sur son compte'])
    expect(steps.map((s) => s.attributes('rowspan'))).toEqual(['2', '1'])
  })

  it('asks to open the capture a step was commented on', () => {
    const w = mount(CommentRecap, {
      props: { grid, comments: [comment({ variantIds: ['v2'] })] },
    })
    w.find('tbody td[rowspan] a').trigger('click')
    expect(w.emitted('open')?.[0]).toEqual(['s1', 'v2'])
  })

  it('steps back a comment nobody is waiting on, without hiding it', () => {
    // Nothing is ever deleted; what needs a move simply keeps the eye.
    const w = mount(CommentRecap, {
      props: {
        grid,
        comments: [
          comment({ id: 'k1', state: 'to-review' }),
          comment({ id: 'k2', state: 'tracked' }),
          comment({ id: 'k3', state: 'discarded' }),
        ],
      },
    })
    const rows = w.findAll('tbody tr')
    expect(rows[0].classes()).not.toContain('opacity-50')
    expect(rows[1].classes()).toContain('opacity-50')
    expect(rows[2].classes()).toContain('opacity-50')
  })

  it('gives a kind its own shape, never the one a state already uses', () => {
    // A plain circle already means "waiting for the reviewer"; an improvement
    // is not a state.
    const w = mount(CommentRecap, {
      props: { grid, comments: [comment({ kind: 'improvement' })] },
    })
    expect(w.find('[aria-label="amélioration"]').exists()).toBe(true)
    expect(w.find('[aria-label="à relire"]').exists()).toBe(false)
  })

  it('counts what is still open, not what exists', () => {
    const w = mount(CommentRecap, {
      props: {
        grid,
        comments: [
          comment(),
          comment({ id: 'k2', state: 'validated' }),
          comment({ id: 'k3', state: 'discarded' }),
        ],
      },
    })
    expect(w.text()).toContain('1 ouvert sur 3')
  })
})
