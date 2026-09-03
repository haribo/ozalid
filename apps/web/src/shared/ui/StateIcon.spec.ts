import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import StateIcon from './StateIcon.vue'
import { TONE_LABELS, type Tone } from '@/shared/lib'

describe('StateIcon', () => {
  // The label is the whole of what a screen reader gets: the icon is a shape.
  // Nothing asserted these words until translating the interface changed all
  // four of them and no test noticed (#120).
  it('names its tone for a screen reader', () => {
    for (const tone of Object.keys(TONE_LABELS) as Tone[]) {
      const w = mount(StateIcon, { props: { tone } })
      expect(w.attributes('aria-label'), tone).toBe(TONE_LABELS[tone])
    }
  })

  it('says the words a reviewer would say', () => {
    // Spelled out rather than read from the map: a test that compares the map
    // with itself passes whatever the map says, which is what let `relu`
    // survive the sweep to English.
    expect(TONE_LABELS).toEqual({
      idle: 'not instrumented',
      reviewer: 'to review',
      dev: 'to fix',
      done: 'reviewed',
    })
  })

  it('lets a caller override the label, for a count', () => {
    const w = mount(StateIcon, { props: { tone: 'done', label: '12 reviewed' } })
    expect(w.attributes('aria-label')).toBe('12 reviewed')
  })
})
