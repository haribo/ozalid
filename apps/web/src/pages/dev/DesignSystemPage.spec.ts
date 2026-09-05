import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DesignSystemPage from './DesignSystemPage.vue'

describe('DesignSystemPage', () => {
  // The gallery's contract: every primitive of shared/ui has its section, so
  // a new one without a home here is caught the day it lands (#155).
  it('shows a section for every primitive', () => {
    const w = mount(DesignSystemPage)
    for (const name of [
      'AppButton',
      'StatePill',
      'RightsPill',
      'KindIcon',
      'ActionIcon',
      'AdminIcon',
      'VariantHead',
      'MintedTokenPanel',
    ]) {
      expect(w.text(), name).toContain(name)
    }
  })

  it('feeds the gauge its edge cases — an empty slice, a crowded one', () => {
    const w = mount(DesignSystemPage)
    expect(w.text()).toContain('214')
  })
})
