import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TextField from './TextField.vue'

describe('TextField', () => {
  it('names the field wherever the label sits', () => {
    const std = mount(TextField, { props: { label: 'name' } })
    expect(std.get('label').text()).toContain('name')

    const float = mount(TextField, { props: { label: 'address', floating: true } })
    expect(float.get('label').attributes('for')).toBe(float.get('input').attributes('id'))
  })

  it('keeps the floating label inside the field while it is empty', () => {
    const w = mount(TextField, { props: { label: 'address', floating: true } })
    expect(w.get('label').classes()).toContain('top-1/2')
  })

  it('lands the floating label on the border once a value exists', async () => {
    const w = mount(TextField, { props: { label: 'address', floating: true } })
    await w.get('input').setValue('nicolas@ozalid.org')
    expect(w.get('label').classes()).toContain('-top-2')
    expect(w.get('label').classes()).not.toContain('top-1/2')
  })

  it('reddens on invalid, in both shapes', () => {
    for (const floating of [false, true]) {
      const w = mount(TextField, { props: { label: 'name', floating, invalid: true } })
      expect(w.get('input').classes().join(' ')).toContain('border-red')
    }
  })
})
