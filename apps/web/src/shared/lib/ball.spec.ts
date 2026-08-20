import { describe, expect, it } from 'vitest'
import { ballHolder } from './ball'

describe('ballHolder', () => {
  it('gives the ball to the reviewer while something awaits judgment', () => {
    expect(ballHolder('to-review')).toBe('reviewer')
  })

  it('gives the ball to the dev while a comment awaits them', () => {
    expect(ballHolder('to-fix')).toBe('dev')
  })

  it('leaves a clean or uninstrumented case to nobody', () => {
    expect(ballHolder('reviewed')).toBe('nobody')
    expect(ballHolder('not-instrumented')).toBe('nobody')
  })
})
