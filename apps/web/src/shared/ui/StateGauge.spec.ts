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

  it('sizes each segment by its count, in the fixed order (#236)', () => {
    // Whatever the caller sends, the reviewer's work leads: position
    // identifies the status as much as hue does (WCAG 1.4.1).
    const w = mount(StateGauge, {
      props: {
        parts: [
          { tone: 'done', count: 9 },
          { tone: 'reviewer', count: 1 },
        ],
      },
    })
    const grown = w.findAll('.basis-0').map((s) => s.attributes('style'))
    expect(grown[0]).toContain('flex-grow: 1')
    expect(grown[1]).toContain('flex-grow: 9')
  })

  it('carries each count inside its own segment, minimum width held (#236)', () => {
    const w = mount(StateGauge, {
      props: {
        parts: [
          { tone: 'done', count: 214, label: 'accepted' },
          { tone: 'dev', count: 1, label: 'refused' },
        ],
      },
    })
    const segments = w.findAll('.basis-0')
    expect(segments.map((s) => s.text())).toEqual(['214', '1'])
    // The sliver keeps room for its digit: 1 refused out of 215 never hides.
    expect(segments[1].classes().join(' ')).toContain('min-w-')
    expect(w.find('[aria-label="1 refused"]').exists()).toBe(true)
  })

  it('spells out each count, since an icon nobody has learnt is decoration', () => {
    const w = mount(StateGauge, {
      props: { parts: [{ tone: 'done', count: 12, label: 'accepted' }] },
    })
    expect(w.html()).toContain('12 accepted')
  })

  it('shows a dash rather than an empty bar when nothing is counted', () => {
    const w = mount(StateGauge, { props: { parts: [{ tone: 'done', count: 0 }] } })
    expect(w.text()).toBe('—')
  })
})
