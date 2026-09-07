import { describe, expect, it } from 'vitest'
import { ballHolder, toneOfCase, TONE_LABELS } from './states'
import type { CaseState } from './states'

const ALL: CaseState[] = ['not-instrumented', 'to-review', 'refused', 'accepted']

describe('the state vocabulary', () => {
  it('hands the ball to the reviewer while something awaits judgment', () => {
    expect(ballHolder('to-review')).toBe('reviewer')
  })

  it('hands it to the dev while a comment awaits them', () => {
    expect(ballHolder('refused')).toBe('dev')
  })

  it('leaves a clean or uninstrumented case to nobody', () => {
    expect(ballHolder('accepted')).toBe('nobody')
    expect(ballHolder('not-instrumented')).toBe('nobody')
  })

  it('gives every state a tone, so no state can render unmarked', () => {
    for (const state of ALL) {
      expect(TONE_LABELS[toneOfCase(state)]).toBeTruthy()
    }
  })

  it('colours by who has to act, not by severity', () => {
    // refused and to-improve merged because the dev's action is the same; the
    // tone follows the ball, so both would read alike (ADR 0012).
    expect(toneOfCase('refused')).toBe('dev')
    expect(toneOfCase('to-review')).toBe('reviewer')
  })
})
