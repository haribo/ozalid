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
    { id: 's1', name: 'opens the link', position: 0, captures: [] },
    { id: 's2', name: 'arrives on their account', position: 1, captures: [] },
  ],
  recordings: [],
}

function comment(over: Partial<Comment> = {}): Comment {
  return {
    id: 'k1',
    stepId: 's1',
    body: 'the button is clipped',
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

  it('marks one column per variant the comment applies to', () => {
    // One defect over two variants is one row with two waiting marks, never
    // two rows.
    const w = mount(CommentRecap, {
      props: { grid, comments: [comment({ variantIds: ['v1', 'v2'] })] },
    })
    expect(w.findAll('tbody tr')).toHaveLength(1)
    expect(w.findAll('tbody [role="img"]')).toHaveLength(2)
  })

  it('says one mark per variant and per block', () => {
    // ADR 0022: the acceptance landed on v1 and released it from the
    // coverage; v2 still waits. One mark each, read from the history.
    const w = mount(CommentRecap, {
      props: {
        grid,
        comments: [
          comment({
            state: 'to-review',
            variantIds: ['v2'],
            issues: [
              { id: 'ref1', issueId: '173', title: 'let the label sit', state: 'to-review' },
            ],
            judgments: [
              { verdict: 'accepted', variantId: 'v1', actorId: 'nina', at: '2026-09-08T09:00:00Z' },
            ],
          }),
        ],
      },
    })
    const marks = w.findAll('tbody [role="img"]')
    expect(marks.map((m) => m.attributes('aria-label'))).toEqual([
      'accepted on desktop·light',
      'waiting on desktop·dark',
    ])
  })

  it('two standing refusals are two anchored lines', () => {
    // Each remark is a line, its ✗ in the refused variant's column — the
    // alignment says which is which, and the claim line carries no ✗.
    const w = mount(CommentRecap, {
      props: {
        grid,
        comments: [
          comment({
            state: 'refused',
            variantIds: ['v1', 'v2'],
            issues: [
              {
                id: 'ref1',
                issueId: '130',
                title: 'calm the sent state down',
                state: 'refused',
                refusals: [
                  { variantId: 'v1', remark: 'the frame is still green' },
                  { variantId: 'v2', remark: 'three green things remain' },
                ],
              },
            ],
          }),
        ],
      },
    })
    const rows = w.findAll('tbody tr')
    expect(rows).toHaveLength(3)
    expect(rows[0].findAll('[role="img"]')).toHaveLength(0)
    expect(rows[1].text()).toContain('the frame is still green')
    expect(rows[1].findAll('[role="img"]').map((m) => m.attributes('aria-label'))).toEqual([
      'refused on desktop·light',
    ])
    expect(rows[2].text()).toContain('three green things remain')
    expect(rows[2].findAll('[role="img"]').map((m) => m.attributes('aria-label'))).toEqual([
      'refused on desktop·dark',
    ])
  })

  it("shows a refusal's remark, because that is what the dev must read", () => {
    const w = mount(CommentRecap, {
      props: {
        grid,
        comments: [
          comment({
            state: 'refused',
            issues: [
              {
                id: 'ref1',
                issueId: '139',
                title: 'the button is clipped',
                state: 'refused',
                refusals: [{ variantId: 'v2', remark: 'still clipped on iPhone SE' }],
              },
            ],
          }),
        ],
      },
    })
    expect(w.text()).toContain('still clipped on iPhone SE')
  })

  it('keeps a discarded comment visible, with its reason', () => {
    // Nothing is deleted: "who removed this, and why?" must have an answer.
    const w = mount(CommentRecap, {
      props: { grid, comments: [comment({ state: 'discarded', discardReason: 'intentionnel' })] },
    })
    expect(w.text()).toContain('intentionnel')
    expect(w.text()).toContain('discarded')
  })

  it("gathers a step's comments under one capture, in the grid's order", () => {
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
    expect(steps.map((s) => s.text())).toEqual(['opens the link', 'arrives on their account'])
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
    expect(rows[0].classes()).not.toContain('opacity-70')
    expect(rows[1].classes()).toContain('opacity-70')
    expect(rows[2].classes()).toContain('opacity-70')
  })

  it('carries no counter: the rows already say it (#213)', () => {
    const w = mount(CommentRecap, {
      props: {
        grid,
        comments: [
          comment(),
          comment({ id: 'k2', state: 'accepted' }),
          comment({ id: 'k3', state: 'discarded' }),
        ],
      },
    })
    expect(w.text()).not.toContain('open of')
  })
})
