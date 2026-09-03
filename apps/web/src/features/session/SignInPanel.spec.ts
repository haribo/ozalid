import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import SignInPanel from './SignInPanel.vue'

describe('SignInPanel', () => {
  // The label starts inside the field and lands on the border once something
  // is typed (#129). The classes are the behaviour here: they are what moves
  // the label, and the e2e suite only proves the label still resolves.
  it('keeps the label inside the field while it is empty', () => {
    const w = mount(SignInPanel)
    expect(w.get('label').classes()).toContain('top-1/2')
  })

  it('lands the label on the border once a value exists', async () => {
    const w = mount(SignInPanel)
    await w.get('input').setValue('nicolas@ozalid.org')
    const label = w.get('label')
    expect(label.classes()).toContain('-top-2')
    expect(label.classes()).not.toContain('top-1/2')
    // The icon follows the label: it shrinks with it rather than staying put.
    expect(w.get('label svg').attributes('width')).toBe('12')
  })

  it('still names the field for a screen reader wherever it sits', async () => {
    const w = mount(SignInPanel)
    await w.get('input').setValue('x@y.test')
    expect(w.get('label').attributes('for')).toBe(w.get('input').attributes('id'))
  })
})
