import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import StateGauge from './StateGauge.vue'

describe('StateGauge', () => {
  it('drops empty segments, because an empty slice is noise', () => {
    const w = mount(StateGauge, {
      props: {
        parts: [
          { tone: 'done', count: 3 },
          { tone: 'dev', count: 0 },
          { tone: 'reviewer', count: 1 },
        ],
      },
    })
    expect(w.findAll('.basis-0')).toHaveLength(2)
  })

  it('sizes each segment by its count', () => {
    const w = mount(StateGauge, {
      props: {
        parts: [
          { tone: 'done', count: 9 },
          { tone: 'reviewer', count: 1 },
        ],
      },
    })
    const grown = w.findAll('.basis-0').map((s) => s.attributes('style'))
    expect(grown[0]).toContain('flex-grow: 9')
    expect(grown[1]).toContain('flex-grow: 1')
  })

  it('spells out each count, since an icon nobody has learnt is decoration', () => {
    const w = mount(StateGauge, {
      props: { parts: [{ tone: 'done', count: 12, label: 'reviewed' }] },
    })
    expect(w.html()).toContain('12 reviewed')
  })

  it('shows a dash rather than an empty bar when nothing is counted', () => {
    const w = mount(StateGauge, { props: { parts: [{ tone: 'done', count: 0 }] } })
    expect(w.text()).toBe('—')
  })
})
