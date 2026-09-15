import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CaptureCarousel from './CaptureCarousel.vue'
import type { components } from '@/shared/api'

type Grid = components['schemas']['Grid']
type Comment = components['schemas']['Comment']
type Status = components['schemas']['GridCapture']['status']

const variants = [{ id: 'v1', label: 'dark·desktop', values: { theme: 'dark' } }]

function gridWith(status: Status): Grid {
  return {
    caseId: 'c1',
    variants,
    steps: [
      {
        id: 's1',
        name: 'confirms the order',
        position: 0,
        captures: [{ id: 'cap1', variantId: 'v1', hash: 'sha256:a', status, movedPixels: 4812 }],
      },
    ],
    recordings: [],
  }
}

function mountWith(status: Status, comments: Comment[] = []) {
  return mount(CaptureCarousel, {
    props: { slug: 'atlas', grid: gridWith(status), comments, stepId: 's1', variantId: 'v1' },
  })
}

const marked = (w: ReturnType<typeof mountWith>) => ({
  veil: w.find('img').classes().includes('opacity-40'),
  disc: w.find('[data-test="stage-mark"]').exists(),
})

describe('marking a judged capture on the stage (ADR 0026)', () => {
  it('marks an accepted capture, and a refused one', () => {
    expect(marked(mountWith('accepted'))).toEqual({ veil: true, disc: true })
    expect(mountWith('accepted').find('[aria-label="accepted"]').exists()).toBe(true)

    expect(marked(mountWith('refused'))).toEqual({ veil: true, disc: true })
    expect(mountWith('refused').find('[aria-label="refused"]').exists()).toBe(true)
  })

  it('leaves a capture awaiting a verdict at full strength', () => {
    expect(marked(mountWith('to-review'))).toEqual({ veil: false, disc: false })
  })

  it('leaves a moved capture at full strength, and keeps the badge that says why', () => {
    // It needs eyes again: the mark of a settled capture would say the
    // opposite, and the badge already says why it came back.
    expect(marked(mountWith('moved'))).toEqual({ veil: false, disc: false })
    expect(mountWith('moved').text()).toContain('moved')
    expect(mountWith('moved').text()).toContain('4812')
  })

  it('reads the capture status, not the verdict pair', () => {
    // A capture accepted long ago, with a delivered fix now awaiting judgment:
    // the pair goes empty, and a mark taken from it would make the stage
    // contradict the grid about the same capture.
    const delivered: Comment = {
      id: 'k1',
      stepId: 's1',
      body: 'the avatar is squashed',
      state: 'to-review',
      variantIds: ['v1'],
      authorId: 'nina',
      createdAt: '2026-09-14T09:00:00Z',
      judgments: [],
      issues: [{ id: 'ref1', issueId: '139', state: 'to-review', title: 'unsquash the avatar' }],
    }
    const w = mountWith('accepted', [delivered])

    expect(marked(w)).toEqual({ veil: true, disc: true })
    // Neither half of the pair is filled while the fix awaits a verdict.
    expect(w.findAll('[aria-pressed="true"]')).toHaveLength(0)
  })
})
