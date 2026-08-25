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
          { variantId: 'v1', hash: 'sha256:aaa', status: 'validated' },
          { variantId: 'v2', hash: 'sha256:bbb', status: 'to-fix' },
        ],
      },
      // Not every variant exists at every step.
      {
        id: 's2',
        name: 'submits',
        position: 1,
        cells: [{ variantId: 'v1', hash: 'sha256:ccc', status: 'to-review' }],
      },
    ],
    recordings: [],
    ...over,
  }
}

describe('CaseGrid', () => {
  it('says so plainly when a case has never been captured', () => {
    // Not being instrumented is a legitimate state, not an error (ADR 0012).
    const w = mount(CaseGrid, { props: { grid: grid({ steps: [], variants: [] }) } })
    expect(w.text()).toContain('aucune capture')
    expect(w.find('table').exists()).toBe(false)
  })

  it('draws one column per variant and one row per step', () => {
    const w = mount(CaseGrid, { props: { grid: grid() } })
    expect(w.findAll('thead th')).toHaveLength(3) // step + 2 variants
    expect(w.findAll('tbody tr')).toHaveLength(2)
  })

  it('fetches an image through the API, never from a guessed origin', () => {
    const w = mount(CaseGrid, { props: { grid: grid() } })
    expect(w.find('img').attributes('src')).toBe('/api/blobs/sha256:aaa')
  })

  it('says where the review stands on each square', () => {
    const w = mount(CaseGrid, { props: { grid: grid() } })
    const text = w.text()
    expect(text).toContain('validée')
    expect(text).toContain('commentée')
    expect(text).toContain('à juger')
  })

  it('asks to open the carousel rather than editing anything', () => {
    // The grid shows and navigates; judging happens in front of the capture.
    const w = mount(CaseGrid, { props: { grid: grid() } })
    expect(w.find('input').exists()).toBe(false)

    w.findAll('button')[0].trigger('click')
    expect(w.emitted('open')?.[0]).toEqual(['s1', 'v1'])
  })

  it('marks the capture currently open, so returning to the grid finds it', () => {
    const w = mount(CaseGrid, {
      props: { grid: grid(), openCell: { stepId: 's1', variantId: 'v2' } },
    })
    const marked = w.findAll('button').filter((b) => b.classes().includes('ring-2'))
    expect(marked).toHaveLength(1)
    expect(marked[0].attributes('aria-label')).toContain('mobile·dark')
  })

  it('gives every capture an alt naming its step and variant', () => {
    const w = mount(CaseGrid, { props: { grid: grid() } })
    expect(w.find('img').attributes('alt')).toBe('opens the form — desktop·light')
  })

  it('keeps a portrait variant portrait', () => {
    // The size now lives on the button that frames the image, since the button
    // is what the reviewer aims at.
    const w = mount(CaseGrid, { props: { grid: grid() } })
    const frames = w.findAll('button')
    expect(frames[0].classes().join(' ')).toContain('w-[116px]')
    expect(frames[1].classes().join(' ')).toContain('w-[60px]')
  })

  it('marks a missing capture as an anomaly, never as a neutral blank', () => {
    // A case is meant to be complete: a step missing a variant its siblings
    // carry is a failed run, and saying "no capture" there hides it (ADR 0016).
    const w = mount(CaseGrid, { props: { grid: grid() } })
    const second = w.findAll('tbody tr')[1]
    expect(second.findAll('img')).toHaveLength(1)
    expect(second.text()).toContain('manquante')
    expect(second.find('[aria-label="capture manquante"]').exists()).toBe(true)
  })

  it('rings each capture with its own verdict, and tints no cell behind it', () => {
    // The colour belongs to the capture, not to the ground around it: this
    // interface frames someone else's product.
    const w = mount(CaseGrid, { props: { grid: grid() } })
    const frames = w.findAll('button')
    expect(frames[0].classes().join(' ')).toContain('border-emerald-600')
    expect(frames[1].classes().join(' ')).toContain('border-amber-600')
    expect(frames[2].classes().join(' ')).toContain('border-slate-300')

    const cells = w.findAll('tbody td')
    expect(cells.some((c) => c.classes().join(' ').includes('bg-emerald'))).toBe(false)
    expect(cells.some((c) => c.classes().join(' ').includes('bg-amber'))).toBe(false)
  })

  it('steps a judged capture back, and leaves what needs eyes at full strength', () => {
    const w = mount(CaseGrid, { props: { grid: grid() } })
    const images = w.findAll('img')
    expect(images[0].classes()).toContain('opacity-40') // validated
    expect(images[1].classes()).toContain('opacity-40') // commented
    expect(images[2].classes()).not.toContain('opacity-40') // still to judge
  })

  it('leaves the recording without a verdict ring, since nothing judges it', () => {
    // A recording is not comparable, so it carries no state (ADR 0013).
    const w = mount(CaseGrid, {
      props: { grid: grid({ recordings: [{ variantId: 'v1', hash: 'sha256:vid' }] }) },
    })
    const link = w.find('a[href="/api/blobs/sha256:vid"]')
    const classes = link.classes().join(' ')
    expect(classes).not.toContain('emerald')
    expect(classes).not.toContain('amber')
  })

  it('only shows the recording row when a recording exists', () => {
    const without = mount(CaseGrid, { props: { grid: grid() } })
    expect(without.text()).not.toContain('enregistrement')

    const withOne = mount(CaseGrid, {
      props: { grid: grid({ recordings: [{ variantId: 'v1', hash: 'sha256:vid' }] }) },
    })
    expect(withOne.text()).toContain('enregistrement')
  })
})
