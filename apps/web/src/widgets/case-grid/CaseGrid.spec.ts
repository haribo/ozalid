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
          { variantId: 'v1', hash: 'sha256:aaa' },
          { variantId: 'v2', hash: 'sha256:bbb' },
        ],
      },
      // Not every variant exists at every step.
      { id: 's2', name: 'submits', position: 1, cells: [{ variantId: 'v1', hash: 'sha256:ccc' }] },
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

  it('leaves a cell with no capture empty instead of drawing a placeholder image', () => {
    const w = mount(CaseGrid, { props: { grid: grid() } })
    const second = w.findAll('tbody tr')[1]
    expect(second.findAll('img')).toHaveLength(1)
    expect(second.text()).toContain('pas de capture')
  })

  it('fetches an image through the API, never from a guessed origin', () => {
    const w = mount(CaseGrid, { props: { grid: grid() } })
    expect(w.find('img').attributes('src')).toBe('/api/blobs/sha256:aaa')
  })

  it('gives every capture an alt naming its step and variant', () => {
    const w = mount(CaseGrid, { props: { grid: grid() } })
    expect(w.find('img').attributes('alt')).toBe('opens the form — desktop·light')
  })

  it('keeps a portrait variant portrait', () => {
    const w = mount(CaseGrid, { props: { grid: grid() } })
    const imgs = w.findAll('img')
    expect(imgs[0].classes().join(' ')).toContain('w-[116px]')
    expect(imgs[1].classes().join(' ')).toContain('w-[60px]')
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
