import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import CaptureCarousel, { type Walk } from './CaptureCarousel.vue'
import type { components } from '@/shared/api'

type Grid = components['schemas']['Grid']
type QueueEntry = components['schemas']['QueueEntry']

const variants = [
  { id: 'v1', label: 'dark·desktop', values: { theme: 'dark', viewport: 'desktop' } },
]

const grid: Grid = {
  caseId: 'c1',
  variants,
  steps: [
    {
      id: 's1',
      name: 'opens the form',
      position: 0,
      captures: [{ id: 'a1b2c3d4e5f6', variantId: 'v1', hash: 'sha256:a', status: 'to-review' }],
    },
  ],
  recordings: [{ id: '9f8e7d6c5b4a', variantId: 'v1', hash: 'sha256:v', status: 'to-review' }],
}

function mountAt(recording = false) {
  return mount(CaptureCarousel, {
    props: { slug: 'atlas', grid, comments: [], stepId: 's1', variantId: 'v1', recording },
  })
}

describe('naming what is on screen', () => {
  it('names the capture on screen', () => {
    // The id is what every other channel needs — a ticket, a thread, a request
    // against the API — and it lived only in the API response (#258).
    expect(mountAt().get('[data-test="named-id"]').text()).toContain('a1b2c3d4e5f6')
  })

  it('names the recording in the recording view', () => {
    // One rule for both views: the header names the object on screen, which
    // here is the recording, not the capture behind it (ADR 0023).
    const named = mountAt(true).get('[data-test="named-id"]').text()
    expect(named).toContain('9f8e7d6c5b4a')
    expect(named).not.toContain('a1b2c3d4e5f6')
  })

  it('copies the id it shows', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })

    await mountAt().get('[data-test="named-id"]').trigger('click')

    // The argument matters: an id copied truncated or stale sends somebody
    // looking at the wrong capture.
    expect(writeText).toHaveBeenCalledWith('a1b2c3d4e5f6')
  })

  it('names nothing when there is nothing on screen', () => {
    // Crossing into the next case of a walk, the address names a capture whose
    // grid has not arrived. No id, no affordance — not an empty box (#205).
    const walk: Walk = {
      entries: [] as QueueEntry[],
      caseId: 'case-b',
      trail: '',
      caseName: 'another flow',
      scope: 'Checkout',
    }
    const w = mount(CaptureCarousel, {
      props: { slug: 'atlas', grid, comments: [], stepId: 'unknown', variantId: 'v1', walk },
    })
    expect(w.find('[data-test="named-id"]').exists()).toBe(false)
  })
})
