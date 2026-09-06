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
    expect(w.get('label').classes()).toContain('top-0')
    expect(w.get('label').classes()).not.toContain('top-1/2')
  })

  it('reddens on invalid, in both shapes', () => {
    // The floating shape draws its border on a decorative fieldset so the
    // landed label can sit in a real gap, plate-free (#173).
    for (const [floating, edge] of [
      [false, 'input'],
      [true, 'fieldset'],
    ] as const) {
      const w = mount(TextField, { props: { label: 'name', floating, invalid: true } })
      expect(w.get(edge).classes().join(' ')).toContain('border-red')
    }
  })
})

// The landed label wears no plate: the border gap under it is cut by the
// fieldset's legend, so no colour guess about the ground survives (#173).
it('lands the label in a border gap, never on an opaque plate (#173)', () => {
  const w = mount(TextField, {
    props: { label: 'address', floating: true, modelValue: 'nina@acme.test' },
  })
  const label = w.get('label')
  expect(label.classes().join(' ')).not.toContain('bg-')
  // The legend mirrors the label to size the notch.
  expect(w.get('legend').text()).toBe('address')
})
