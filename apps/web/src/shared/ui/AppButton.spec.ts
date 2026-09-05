import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AppButton from './AppButton.vue'

describe('AppButton', () => {
  it('emits on click, and stops once disabled', async () => {
    const w = mount(AppButton, { slots: { default: 'Create' } })
    await w.trigger('click')
    expect(w.emitted('click')).toHaveLength(1)

    await w.setProps({ disabled: true })
    await w.trigger('click')
    expect(w.emitted('click')).toHaveLength(1)
  })

  it('still clicks in success, and holds its width so the row does not move', async () => {
    const w = mount(AppButton, { slots: { default: 'to validate' } })
    await w.setProps({ success: true })
    await w.trigger('click')
    // The click passes through: whether success is inert feedback or a toggle
    // is the caller's decision — a mouse user must be able to take a
    // validation back (#156).
    expect(w.emitted('click')).toHaveLength(1)
    // The width at the switch is pinned, so the shorter label shifts nothing.
    expect(w.attributes('style')).toContain('min-width')
  })

  it('defaults to type=button, so a stray form never submits', () => {
    expect(mount(AppButton).attributes('type')).toBe('button')
  })

  it('squares itself for an icon and speaks through its label', () => {
    const w = mount(AppButton, { props: { icon: true, size: 'md', label: 'retire this token' } })
    expect(w.classes()).toContain('h-7')
    expect(w.attributes('aria-label')).toBe('retire this token')
  })

  it('shows a pointer, never the arrow of a dead element', () => {
    // The Tailwind v4 preflight sets cursor:default on buttons; this is the
    // one place that decides otherwise for the whole product.
    expect(mount(AppButton).classes()).toContain('cursor-pointer')
    expect(mount(AppButton, { props: { disabled: true } }).classes()).toContain(
      'cursor-not-allowed',
    )
  })
})
