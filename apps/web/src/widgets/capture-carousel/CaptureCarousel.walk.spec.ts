import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CaptureCarousel, { type Walk } from './CaptureCarousel.vue'
import type { components } from '@/shared/api'

type Grid = components['schemas']['Grid']
type QueueEntry = components['schemas']['QueueEntry']

const variants = [
  { id: 'v1', label: 'desktop·light', values: { theme: 'light' } },
  { id: 'v2', label: 'desktop·dark', values: { theme: 'dark' } },
]

/** The grid of the case the walk is currently standing in. */
const grid: Grid = {
  caseId: 'case-a',
  variants,
  steps: [
    {
      id: 's1',
      name: 'opens the form',
      position: 0,
      captures: [
        { id: 'cap1', variantId: 'v1', hash: 'sha256:a', status: 'to-review' },
        { id: 'cap2', variantId: 'v2', hash: 'sha256:b', status: 'to-review' },
      ],
    },
  ],
  recordings: [],
}

function entry(caseId: string, caseTitle: string, stepId: string, variantId: string): QueueEntry {
  return {
    caseId,
    caseTitle,
    stepId,
    stepName: 'opens the form',
    stepPosition: 0,
    variant: variants.find((v) => v.id === variantId)!,
    capture: { id: `cap-${caseId}-${variantId}`, variantId, hash: 'sha256:x', status: 'to-review' },
  }
}

/** Two cases, two variants each: the walk crosses a case boundary at index 2. */
const entries: QueueEntry[] = [
  entry('case-a', 'pay by card', 's1', 'v1'),
  entry('case-a', 'pay by card', 's1', 'v2'),
  entry('case-b', 'choose a carrier', 's9', 'v1'),
]

function walkAt(caseId: string): Walk {
  return {
    entries,
    caseId,
    trail: 'Checkout › Payment',
    caseName: caseId === 'case-a' ? 'pay by card' : 'choose a carrier',
  }
}

function mountWalking(variantId: string, stepId = 's1', caseId = 'case-a') {
  return mount(CaptureCarousel, {
    props: {
      slug: 'atlas',
      grid,
      comments: [],
      stepId,
      variantId,
      walk: walkAt(caseId),
    },
  })
}

const press = (key: string) => window.dispatchEvent(new KeyboardEvent('keydown', { key }))

describe('walking a queue', () => {
  it('walks the queue in order rather than the case grid', () => {
    const w = mountWalking('v1')
    press('ArrowRight')
    expect(w.emitted('moveEntry')?.[0][0]).toMatchObject({
      caseId: 'case-a',
      variant: { id: 'v2' },
    })
  })

  it('crosses into the next case on the same key', () => {
    const w = mountWalking('v2')
    press('ArrowRight')
    // The reviewer presses the same key; the case changes under them, which
    // the banner is what says (#205).
    expect(w.emitted('moveEntry')?.[0][0]).toMatchObject({
      caseId: 'case-b',
      caseTitle: 'choose a carrier',
    })
  })

  it('stops at both ends instead of wrapping', () => {
    const first = mountWalking('v1')
    press('ArrowLeft')
    expect(first.emitted('moveEntry')).toBeUndefined()

    const last = mountWalking('v1', 's9', 'case-b')
    press('ArrowRight')
    expect(last.emitted('moveEntry')).toBeUndefined()
  })

  it('counts the queue, not the flow', () => {
    const w = mountWalking('v2')
    expect(w.text()).toContain('2 / 3 to judge')
    expect(w.text()).not.toContain('1 / 1 ·')
  })

  it('names the case in the banner and says nothing about its status', () => {
    const banner = mountWalking('v1').find('[data-test="walk-banner"]')
    expect(banner.text()).toContain('pay by card')
    expect(banner.text()).toContain('Checkout › Payment')
    // One fact, one word: the verdict pair announces to-review and the badge
    // announces moved. A third word in the banner is the regression the
    // mockup round settled (#205, product.md §3.6).
    expect(banner.text()).not.toContain('to-review')
    expect(banner.text()).not.toContain('moved')
  })

  it('waits for the next case rather than judging blind', async () => {
    // The address already names case-b's capture; the grid on screen is still
    // case-a's, so there is nothing to look at yet (#205).
    const w = mountWalking('v1', 's9', 'case-b')
    expect(w.find('[data-test="walk-loading"]').exists()).toBe(true)
    expect(w.find('img').exists()).toBe(false)

    press(' ')
    await w.vm.$nextTick()
    expect(w.emitted('accept')).toBeUndefined()
  })

  it('leaves the step walk untouched when no queue was handed in', () => {
    const w = mount(CaptureCarousel, {
      props: { slug: 'atlas', grid, comments: [], stepId: 's1', variantId: 'v1' },
    })
    expect(w.text()).toContain('1 / 1')
    expect(w.find('[data-test="walk-banner"]').exists()).toBe(false)
    press('ArrowDown')
    expect(w.emitted('move')?.[0]).toEqual(['s1', 'v2'])
    expect(w.emitted('moveEntry')).toBeUndefined()
  })
})
