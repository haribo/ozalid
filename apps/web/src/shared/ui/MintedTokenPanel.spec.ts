import { describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import MintedTokenPanel from './MintedTokenPanel.vue'

// Long enough that truncating it shows, readable enough that a secret scanner
// does not mistake it for one. What it contains never mattered to the test.
const TOKEN = 'ozp_the-token-this-panel-was-handed'

/** By label, not by position: the two buttons sit in different rows. */
const button = (w: VueWrapper, text: string) =>
  w.findAll('button').find((b) => b.text().includes(text))!

describe('MintedTokenPanel', () => {
  it('shows the token whole, because half a token opens nothing', () => {
    const w = mount(MintedTokenPanel, { props: { token: TOKEN } })
    expect(w.get('code').text()).toBe(TOKEN)
  })

  it('copies what it shows', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })

    const w = mount(MintedTokenPanel, { props: { token: TOKEN } })
    await button(w, 'Copy').trigger('click')

    // The argument matters: a panel that copies a truncated or stale value
    // hands somebody a credential that fails on first use, hours later.
    expect(writeText).toHaveBeenCalledWith(TOKEN)
    expect(w.text()).toContain('Copied')

    vi.unstubAllGlobals()
  })

  it('asks to be dismissed rather than dismissing itself', async () => {
    // The panel holds nothing; the screen that minted the token owns it. If
    // this component cleared its own state the token would be unreachable
    // while still on screen elsewhere.
    const w = mount(MintedTokenPanel, { props: { token: TOKEN } })
    await button(w, 'I have copied the token').trigger('click')
    expect(w.emitted('dismiss')).toHaveLength(1)
  })
})
