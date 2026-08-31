import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CaseGrid from './CaseGrid.vue'
import type { components } from '@/shared/api'

type Grid = components['schemas']['Grid']

const VARIANTS = [
  { id: 'v1', label: 'desktop·light', values: { viewport: 'desktop', theme: 'light' } },
  { id: 'v2', label: 'mobile·dark', values: { viewport: 'mobile', theme: 'dark' } },
]

function grid(over: Partial<Grid> = {}): Grid {
  return {
    caseId: 'abc123456789',
    variants: VARIANTS,
    steps: [
      {
        id: 's1',
        name: 'opens the form',
        position: 0,
        cells: [
          { id: 'cap1', variantId: 'v1', hash: 'sha256:aaa', status: 'validated' },
          { id: 'cap2', variantId: 'v2', hash: 'sha256:bbb', status: 'to-fix' },
        ],
      },
      // Not every variant exists at every step.
      {
        id: 's2',
        name: 'submits',
        position: 1,
        cells: [{ id: 'cap3', variantId: 'v1', hash: 'sha256:ccc', status: 'to-review' }],
      },
    ],
    recordings: [],
    ...over,
  }
}

// The grid's own cells. Assertions must be scoped to them: the legend under the
// table carries every word, so a check against the whole component would pass
// whatever the cells say.
const cells = (w: ReturnType<typeof mount>) => w.findAll('tbody td')

// One square that has moved, from a chosen verdict.
const oneMoved = (status: 'validated' | 'to-fix') => ({
  steps: [
    {
      id: 's1',
      name: 'opens the form',
      position: 0,
      cells: [
        {
          id: 'cap4',
          variantId: 'v1',
          hash: 'sha256:aaa',
          status,
          freshness: 'to-re-review' as const,
        },
      ],
    },
  ],
})

// One validated square, at a chosen freshness.
const oneValidated = (freshness: 'current' | 'to-re-review') => ({
  steps: [
    {
      id: 's1',
      name: 'opens the form',
      position: 0,
      cells: [
        {
          id: 'cap5',
          variantId: 'v1',
          hash: 'sha256:aaa',
          status: 'validated' as const,
          freshness,
        },
      ],
    },
  ],
})

describe('CaseGrid', () => {
  it('says so plainly when a case has never been captured', () => {
    // Not being instrumented is a legitimate state, not an error (ADR 0012).
    const w = mount(CaseGrid, { props: { slug: 'atlas', grid: grid({ steps: [], variants: [] }) } })
    expect(w.text()).toContain('aucune capture')
    expect(w.find('table').exists()).toBe(false)
  })

  it('draws one column per variant and one row per step', () => {
    const w = mount(CaseGrid, { props: { slug: 'atlas', grid: grid() } })
    expect(w.findAll('thead th')).toHaveLength(3) // step + 2 variants
    expect(w.findAll('tbody tr')).toHaveLength(2)
  })

  it('fetches an image through the API, never from a guessed origin', () => {
    const w = mount(CaseGrid, { props: { slug: 'atlas', grid: grid() } })
    // Through the capture, not the content address: a hash names no project
    // and cannot be authorised (product.md §8.1, #71).
    expect(w.find('img').attributes('src')).toBe('/api/projects/atlas/captures/cap1')
  })

  it('says where the review stands on each square', () => {
    // Read from the cells, never from the component: the legend below repeats
    // every word, and a check that cannot fail is not a check.
    const w = mount(CaseGrid, { props: { slug: 'atlas', grid: grid() } })
    // The accessible name sits on the disc, and the glyph inside it is
    // decorative — one mark, one name.
    const marks = cells(w).map((c) => c.findAll('[role="img"]')[0]?.attributes('aria-label'))
    expect(marks).toEqual([
      'validée',
      'commentée',
      // A square nobody has judged carries no mark at all — bare is the
      // reading, and it is the only one that leaves every pixel visible.
      undefined,
      'manquante',
    ])
  })

  it('asks to open the carousel rather than editing anything', () => {
    // The grid shows and navigates; judging happens in front of the capture.
    const w = mount(CaseGrid, { props: { slug: 'atlas', grid: grid() } })
    expect(w.find('input').exists()).toBe(false)

    w.findAll('button')[0].trigger('click')
    expect(w.emitted('open')?.[0]).toEqual(['s1', 'v1'])
  })

  it('marks the capture currently open, so returning to the grid finds it', () => {
    const w = mount(CaseGrid, {
      props: { slug: 'atlas', grid: grid(), openCell: { stepId: 's1', variantId: 'v2' } },
    })
    const marked = w.findAll('button').filter((b) => b.classes().includes('ring-2'))
    expect(marked).toHaveLength(1)
    expect(marked[0].attributes('aria-label')).toContain('mobile·dark')
  })

  it('gives every capture an alt naming its step and variant', () => {
    const w = mount(CaseGrid, { props: { slug: 'atlas', grid: grid() } })
    expect(w.find('img').attributes('alt')).toBe('opens the form — desktop·light')
  })

  it('keeps a portrait variant portrait', () => {
    // The size now lives on the button that frames the image, since the button
    // is what the reviewer aims at.
    const w = mount(CaseGrid, { props: { slug: 'atlas', grid: grid() } })
    const frames = w.findAll('button')
    expect(frames[0].classes().join(' ')).toContain('w-[116px]')
    expect(frames[1].classes().join(' ')).toContain('w-[60px]')
  })

  it('marks a missing capture as an anomaly, never as a neutral blank', () => {
    // A case is meant to be complete: a step missing a variant its siblings
    // carry is a failed run, and drawing it neutrally would hide it (ADR 0016).
    const w = mount(CaseGrid, { props: { slug: 'atlas', grid: grid() } })
    const hole = cells(w)[3]
    expect(hole.find('img').exists()).toBe(false)
    expect(hole.find('[aria-label="manquante"]').exists()).toBe(true)
    expect(hole.find('.border-dashed').exists()).toBe(true)
  })

  it('rings each capture with its own verdict, and tints no cell behind it', () => {
    // The colour belongs to the capture, not to the ground around it: this
    // interface frames someone else's product.
    const w = mount(CaseGrid, { props: { slug: 'atlas', grid: grid() } })
    const frames = w.findAll('button')
    expect(frames[0].classes().join(' ')).toContain('border-emerald-600')
    expect(frames[1].classes().join(' ')).toContain('border-amber-600')
    expect(frames[2].classes().join(' ')).toContain('border-slate-300')

    expect(cells(w).some((c) => c.classes().join(' ').includes('bg-emerald'))).toBe(false)
    expect(cells(w).some((c) => c.classes().join(' ').includes('bg-amber'))).toBe(false)
  })

  it('steps a judged capture back, and leaves what needs eyes at full strength', () => {
    const w = mount(CaseGrid, { props: { slug: 'atlas', grid: grid() } })
    const images = w.findAll('img')
    expect(images[0].classes()).toContain('opacity-40') // validated
    expect(images[1].classes()).toContain('opacity-40') // commented
    expect(images[2].classes()).not.toContain('opacity-40') // still to judge
  })

  it('leaves the recording without a verdict ring, since nothing judges it', () => {
    // A recording is not comparable, so it carries no state (ADR 0013).
    const w = mount(CaseGrid, {
      props: {
        slug: 'atlas',
        grid: grid({ recordings: [{ id: 'rec1', variantId: 'v1', hash: 'sha256:vid' }] }),
      },
    })
    const link = w.find('a[href="/api/projects/atlas/recordings/rec1"]')
    const classes = link.classes().join(' ')
    expect(classes).not.toContain('emerald')
    expect(classes).not.toContain('amber')
  })

  it('renders a capture that moved as one to judge, carrying why it came back', () => {
    // For the only question the grid asks, it has not been validated — not the
    // bytes on display (frontend ADR 0003).
    const w = mount(CaseGrid, {
      props: { slug: 'atlas', grid: grid(oneValidated('to-re-review')) },
    })
    const cell = cells(w)[0]

    expect(cell.find('img').classes()).not.toContain('opacity-40')
    expect(cell.find('[aria-label="a bougé"]').exists()).toBe(true)
    // The verdict it used to carry is exactly what the grid no longer reports.
    expect(cell.find('[aria-label="validée"]').exists()).toBe(false)
    expect(cell.find('button').classes().join(' ')).not.toContain('emerald')
  })

  it('reads a moved capture the same whatever verdict it used to carry', () => {
    // Validated-and-moved and commented-and-moved are one cell: what separated
    // them is what the grid stopped reporting (frontend ADR 0003).
    const fromValidated = mount(CaseGrid, {
      props: { slug: 'atlas', grid: grid(oneMoved('validated')) },
    })
    const fromCommented = mount(CaseGrid, {
      props: { slug: 'atlas', grid: grid(oneMoved('to-fix')) },
    })
    expect(cells(fromValidated)[0].html()).toBe(cells(fromCommented)[0].html())
  })

  it('says nothing about freshness when there is nothing to compare against', () => {
    // Absent is a third answer, not "unchanged" (ADR 0017).
    const w = mount(CaseGrid, { props: { slug: 'atlas', grid: grid() } })
    for (const cell of cells(w)) {
      expect(cell.find('[aria-label="a bougé"]').exists()).toBe(false)
    }
  })

  it('puts every status on a disc, and no action on one', () => {
    // A status is a glyph on a disc; a gesture is not (frontend ADR 0003). The
    // grid shows only statuses, so every mark it draws wears one.
    const w = mount(CaseGrid, { props: { slug: 'atlas', grid: grid() } })
    for (const mark of w.findAll('tbody [role="img"]')) {
      expect(mark.classes()).toContain('rounded-full')
    }
  })

  it('gives freshness its own shape, never a state icon', () => {
    const w = mount(CaseGrid, {
      props: { slug: 'atlas', grid: grid(oneValidated('to-re-review')) },
    })
    const cell = cells(w)[0]
    const mark = cell.find('[role="img"]')
    expect(mark.attributes('aria-label')).toBe('a bougé')
    // Two arrows on the disc, not a check: a capture can be validated and moved
    // at once, and one mark must not be mistakable for the other.
    expect(mark.findAll('path')).toHaveLength(2)
    expect(mark.classes()).toContain('rounded-full')
  })

  it('only shows the recording row when a recording exists', () => {
    const without = mount(CaseGrid, { props: { slug: 'atlas', grid: grid() } })
    expect(without.text()).not.toContain('enregistrement')

    const withOne = mount(CaseGrid, {
      props: {
        slug: 'atlas',
        grid: grid({ recordings: [{ id: 'cap7', variantId: 'v1', hash: 'sha256:vid' }] }),
      },
    })
    expect(withOne.text()).toContain('enregistrement')
  })
})
