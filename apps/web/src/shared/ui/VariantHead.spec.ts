import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import VariantHead from './VariantHead.vue'

describe('VariantHead', () => {
  it('spells out a value it has no icon for, even compact (#220)', () => {
    // Two locale columns with no icons must never render identically.
    const w = mount(VariantHead, {
      props: { label: 'fr·desktop', values: { locale: 'fr', viewport: 'desktop' }, compact: true },
    })
    expect(w.text()).toContain('fr')
    expect(w.attributes('title')).toBe('fr\u00b7desktop')
  })

  it('keeps the shapes alone when every value has one', () => {
    const w = mount(VariantHead, {
      props: {
        label: 'desktop·light',
        values: { viewport: 'desktop', theme: 'light' },
        compact: true,
      },
    })
    expect(w.text()).toBe('')
    expect(w.findAll('svg')).toHaveLength(2)
  })
})
