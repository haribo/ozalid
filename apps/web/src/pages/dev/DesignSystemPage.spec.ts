import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DesignSystemPage from './DesignSystemPage.vue'
import * as ui from '@/shared/ui'

describe('DesignSystemPage', () => {
  // Derived from the exports, not from a list somebody keeps by hand: the
  // first version hardcoded the names, two primitives landed after it, and
  // nothing noticed — precisely what this test claims to prevent (#155).
  it('shows a section for every primitive shared/ui exports', () => {
    const text = mount(DesignSystemPage).text()
    for (const name of Object.keys(ui)) {
      expect(text, name).toContain(name)
    }
  })

  it('feeds the gauge its edge cases — an empty slice, a crowded one', () => {
    const w = mount(DesignSystemPage)
    expect(w.text()).toContain('214')
  })
})
