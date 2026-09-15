import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import CopyableId from './CopyableId.vue'

const ID = 'a1b2c3d4e5f6'

describe('CopyableId', () => {
  it('shows the whole id, never a truncation', () => {
    // A shortened id sends somebody looking at nothing: it is twelve
    // characters precisely so it fits.
    expect(
      mount(CopyableId, { props: { value: ID, label: 'Copy the capture id' } }).text(),
    ).toContain(ID)
  })

  it('copies exactly what it shows', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })

    const w = mount(CopyableId, { props: { value: ID, label: 'Copy the capture id' } })
    await w.get('button').trigger('click')

    expect(writeText).toHaveBeenCalledWith(ID)
  })

  it('says what it copies, for whoever cannot see it', () => {
    // "Copy" alone tells a screen reader nothing about which of the page's
    // ids is under the cursor.
    const w = mount(CopyableId, { props: { value: ID, label: 'Copy the recording id' } })
    expect(w.get('button').attributes('aria-label')).toBe('Copy the recording id')
  })
})
