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
        { id: 'cap2', variantId: 'v2', hash: 'sha256:b', status: 'accepted' },
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

const press = (key: string) => window.dispatchEvent(new KeyboardEvent('keydown', { key }))

const half = (w: ReturnType<typeof mountAt>, text: string) =>
  w.findAll('button').find((b) => b.text().includes(text))!

/** The remark fixture: a draft is the reviewer's own, no issue attached. */
const draft: Comment = {
  id: 'k9',
  stepId: 's1',
  body: 'a draft nobody tracked yet',
  state: 'to-track',
  variantIds: ['v1'],
  authorId: 'nina',
  createdAt: '2026-08-24T09:00:00Z',
  judgments: [],
}

describe('CaptureCarousel', () => {
  it('says which step it is on — the counter counts steps, not squares (#149)', () => {
    expect(mountAt('v1').text()).toContain('1 / 1')
    expect(mountAt('v2').text()).toContain('1 / 1')
  })

  it('never marks the capture: no veil, no disc — the verdict lives in the bar (ADR 0020)', () => {
    const judged = mountAt('v2')
    expect(judged.find('img').classes()).not.toContain('opacity-40')
    expect(judged.find('[aria-label="accepted"]').exists()).toBe(false)
    // The bar is where the verdict reads.
    expect(half(judged, '✓ accepted').attributes('aria-pressed')).toBe('true')
  })

  it('says on the image that it moved, and by how much', () => {
    const moved: Grid = {
      ...grid,
      steps: [
        {
          ...grid.steps[0],
          cells: [
            {
              id: 'cap1',
              variantId: 'v1',
              hash: 'sha256:a',
              status: 'to-review',
              freshness: 'to-re-review',
              movedPixels: 143,
            },
            grid.steps[0].cells[1],
          ],
        },
      ],
    }
    const w = mount(CaptureCarousel, {
      props: { slug: 'atlas', grid: moved, comments: [], stepId: 's1', variantId: 'v1' },
    })
    expect(w.text()).toContain('moved')
    expect(w.text()).toContain('143 px')
  })

  it('changes variant on the vertical arrows and leaves on Escape', () => {
    const w = mountAt('v1')
    press('ArrowRight')
    expect(w.emitted('move')).toBeUndefined()
    press('ArrowDown')
    expect(w.emitted('move')?.[0]).toEqual(['s1', 'v2'])

    press('Escape')
    expect(w.emitted('close')).toHaveLength(1)
  })
})

describe('the verdict pair (ADR 0020)', () => {
  it('accepts on space, and asks the server rather than deciding alone', () => {
    const w = mountAt('v1')
    press(' ')
    expect(w.emitted('accept')?.[0]).toEqual(['s1', 'v1', false])
  })

  it('takes an acceptance back on the same key — the pair un-presses', () => {
    const w = mountAt('v2')
    press(' ')
    expect(w.emitted('unaccept')?.[0]).toEqual(['s1', 'v2'])
    expect(w.emitted('accept')).toBeUndefined()
  })

  it('un-presses on click too: the filled half is the way back', async () => {
    const w = mountAt('v2')
    await half(w, '✓ accepted').trigger('click')
    expect(w.emitted('unaccept')?.[0]).toEqual(['s1', 'v2'])
  })
})

describe('the refuse sheet (ADR 0020)', () => {
  it('opens under the image and sends nothing until the remark is written', async () => {
    const w = mountAt('v1')
    await half(w, 'refuse').trigger('click')
    expect(w.emitted('refuse')).toBeUndefined()
    expect(w.text()).toContain('Refuse the capture')

    const send = w.find('form').findAll('button').at(-1)!
    expect(send.text()).toBe('refuse')
    expect(send.attributes('disabled')).toBeDefined()
  })

  it('ticks the variant on screen to start with, and the group shortcuts derive from the axes', async () => {
    const w = mountAt('v1')
    await half(w, 'refuse').trigger('click')
    const boxes = w.findAll('input[type="checkbox"]')
    expect(boxes.map((b) => (b.element as HTMLInputElement).checked)).toEqual([true, false])

    const labels = w.findAll('button').map((b) => b.text())
    expect(labels).toContain('desktop')
    expect(labels).toContain('light')
    expect(labels).toContain('all')
  })

  it('refuses with the remark over the ticked variants', async () => {
    const w = mountAt('v1')
    await half(w, 'refuse').trigger('click')
    await w.find('textarea').setValue('too much green')
    await w
      .findAll('button')
      .find((b) => b.text() === 'all')!
      .trigger('click')
    await w.find('form').findAll('button').at(-1)!.trigger('click')
    expect(w.emitted('refuse')).toEqual([
      [{ stepId: 's1', body: 'too much green', variantIds: ['v1', 'v2'], unaccept: false }],
    ])
    // Sent: the sheet is gone.
    expect(w.find('textarea').exists()).toBe(false)
  })

  it('switches verdicts in one gesture: refusing an accepted square takes the acceptance back', async () => {
    const w = mountAt('v2')
    await half(w, 'refuse').trigger('click')
    await w.find('textarea').setValue('second look')
    await w.find('form').findAll('button').at(-1)!.trigger('click')
    expect(w.emitted('refuse')?.[0]).toEqual([
      { stepId: 's1', body: 'second look', variantIds: ['v2'], unaccept: true },
    ])
  })

  it('cancel is a true no-op, and Escape closes the sheet before the carousel', async () => {
    const w = mountAt('v1')
    await half(w, 'refuse').trigger('click')
    await w.find('textarea').setValue('words I regret')
    press(' ')
    expect(w.emitted('accept')).toBeUndefined()

    press('Escape')
    await w.vm.$nextTick()
    expect(w.find('textarea').exists()).toBe(false)
    expect(w.emitted('close')).toBeUndefined()
    expect(w.emitted('refuse')).toBeUndefined()
  })
})

describe("drafts are the reviewer's own (ADR 0020)", () => {
  it('shows the draft as a card, and the card opens the edit sheet pre-filled', async () => {
    const w = mountAt('v1', [draft])
    expect(w.text()).toContain('a draft nobody tracked yet')

    await half(w, 'a draft nobody tracked yet').trigger('click')
    expect(w.text()).toContain('Edit the remark')
    expect((w.find('textarea').element as HTMLTextAreaElement).value).toBe(
      'a draft nobody tracked yet',
    )

    await w.find('textarea').setValue('a draft, reworded')
    await w.find('form').findAll('button').at(-1)!.trigger('click')
    expect(w.emitted('edit')).toEqual([['k9', 'a draft, reworded', ['v1']]])
  })

  it('lets the issue title speak once tracked — the draft retires, unedited', () => {
    const w = mountAt('v1', [
      {
        ...draft,
        state: 'tracked',
        issues: [{ id: 'r1', issueId: '9', state: 'tracked', title: 'one green thing' }],
      },
    ])
    expect(w.text()).not.toContain('a draft nobody tracked yet')
    expect(w.text()).toContain('#9 one green thing')
  })

  it('withdraws the draft refusal from the filled half', async () => {
    const refused: Grid = {
      ...grid,
      steps: [
        {
          ...grid.steps[0],
          cells: [
            { id: 'cap1', variantId: 'v1', hash: 'sha256:a', status: 'refused' },
            grid.steps[0].cells[1],
          ],
        },
      ],
    }
    const w = mount(CaptureCarousel, {
      props: { slug: 'atlas', grid: refused, comments: [draft], stepId: 's1', variantId: 'v1' },
    })
    await half(w, '✗ refused').trigger('click')
    expect(w.emitted('unrefuse')?.[0]).toEqual(['s1', 'v1'])

    // And accepting from there is one gesture: the draft goes with it.
    await half(w, 'accept').trigger('click')
    expect(w.emitted('accept')?.[0]).toEqual(['s1', 'v1', true])
  })
})

describe('a delivered fix takes the pair (#170, #171)', () => {
  const delivered: Comment = {
    ...draft,
    id: 'k1',
    body: 'the avatar is squashed',
    state: 'to-review',
    issue: { id: '139' },
    issues: [{ id: 'ref1', issueId: '139', state: 'to-review', title: 'unsquash the avatar' }],
  }

  it('judges the fix rather than the capture once a delivery is waiting', async () => {
    const w = mountAt('v1', [delivered])
    expect(w.text()).toContain('fix delivered · issue #139')
    expect(w.text()).toContain('unsquash the avatar')

    await half(w, 'accept').trigger('click')
    expect(w.emitted('judge')).toEqual([['k1', 'ref1', true, '']])
    expect(w.emitted('accept')).toBeUndefined()
  })

  it('refuses the fix through the sheet — remark mandatory, no variant ticks', async () => {
    const w = mountAt('v1', [delivered])
    await half(w, 'refuse').trigger('click')
    expect(w.text()).toContain('Refuse the fix — issue #139')
    expect(w.find('input[type="checkbox"]').exists()).toBe(false)

    await w.find('textarea').setValue('still squashed on mobile')
    await w.find('form').findAll('button').at(-1)!.trigger('click')
    expect(w.emitted('judge')).toEqual([['k1', 'ref1', false, 'still squashed on mobile']])
  })

  it('names the act in the issue word and reopens the judgment from the filled half', async () => {
    const accepted = mountAt('v2', [
      {
        ...delivered,
        state: 'accepted',
        variantIds: ['v2'],
        issues: [{ id: 'ref1', issueId: '139', state: 'accepted', title: 'unsquash the avatar' }],
      },
    ])
    expect(accepted.text()).toContain('issue #139 accepted')
    await half(accepted, '✓ accepted').trigger('click')
    expect(accepted.emitted('unjudge')).toEqual([['k1', 'ref1']])

    const refused = mountAt('v1', [
      {
        ...delivered,
        state: 'refused',
        issues: [{ id: 'ref1', issueId: '139', state: 'refused', title: 'unsquash the avatar' }],
      },
    ])
    expect(refused.text()).toContain('issue #139 refused — back to the developer')
    await half(refused, '✗ refused').trigger('click')
    expect(refused.emitted('unjudge')).toEqual([['k1', 'ref1']])
  })

  it('shows one issue at a time — the others live in the recap (#171)', () => {
    const twoRefs: Comment = {
      ...delivered,
      issues: [
        { id: 'ref1', issueId: '129', state: 'to-review', title: 'set the sign-in label' },
        { id: 'ref2', issueId: '131', state: 'tracked', title: 'the empty mailbox hint overflows' },
      ],
    }
    const w = mountAt('v1', [twoRefs])
    expect(w.text()).toContain('set the sign-in label')
    expect(w.text()).not.toContain('the empty mailbox hint overflows')
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

  it('moves to the next step and keeps the variant', () => {
    const w = at('s1', 'v2')
    press('ArrowRight')
    expect(w.emitted('move')).toEqual([['s2', 'v2']])
    w.unmount()
  })

  it('skips a step that lacks the variant', () => {
    const w = at('s2', 'v1')
    press('ArrowRight')
    expect(w.emitted('move')).toBeUndefined()
    w.unmount()
  })

  it('counts steps, and follows the walk', () => {
    expect(at('s2', 'v2').text()).toContain('2 / 3')
    expect(at('s3', 'v2').text()).toContain('3 / 3')
  })
})

describe('the branch loop: a remark without an issue (#175)', () => {
  const remark: Comment = {
    id: 'k5',
    stepId: 's1',
    body: 'too much green everywhere',
    state: 'to-review',
    variantIds: ['v1'],
    authorId: 'nina',
    createdAt: '2026-09-07T09:00:00Z',
    judgments: [],
    issues: [],
  }

  it('judges the delivered remark by its own words — no number', async () => {
    const w = mountAt('v1', [remark])
    expect(w.text()).toContain('fix delivered')
    expect(w.text()).not.toContain('issue #')
    expect(w.text()).toContain('too much green everywhere')

    await half(w, 'accept').trigger('click')
    expect(w.emitted('judge')).toEqual([['k5', '', true, '']])
  })

  it('refuses it through the sheet, remark mandatory, no variant ticks', async () => {
    const w = mountAt('v1', [remark])
    await half(w, 'refuse').trigger('click')
    expect(w.text()).toContain('Refuse the fix')
    expect(w.find('input[type="checkbox"]').exists()).toBe(false)

    await w.find('textarea').setValue('still three green things')
    await w.find('form').findAll('button').at(-1)!.trigger('click')
    expect(w.emitted('judge')).toEqual([['k5', '', false, 'still three green things']])
  })

  it('reopens a settled remark from the filled half', async () => {
    const accepted = mountAt('v2', [{ ...remark, state: 'accepted', variantIds: ['v2'] }])
    await half(accepted, '✓ accepted').trigger('click')
    expect(accepted.emitted('unjudge')).toEqual([['k5', '']])

    const refusedGrid: Grid = {
      ...grid,
      steps: [
        {
          ...grid.steps[0],
          cells: [
            { id: 'cap1', variantId: 'v1', hash: 'sha256:a', status: 'refused' },
            grid.steps[0].cells[1],
          ],
        },
      ],
    }
    const refused = mount(CaptureCarousel, {
      props: {
        slug: 'atlas',
        grid: refusedGrid,
        comments: [{ ...remark, state: 'refused' }],
        stepId: 's1',
        variantId: 'v1',
      },
    })
    await half(refused, '✗ refused').trigger('click')
    expect(refused.emitted('unjudge')).toEqual([['k5', '']])
  })
})
