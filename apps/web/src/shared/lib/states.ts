/**
 * The vocabulary the server speaks, and the one visual language that carries
 * it. The same shape means the same thing at every level: a ticked circle on a
 * category gauge, on a case row and under a capture all say "judged, nothing
 * to say" (ADR 0012).
 */

/** Who holds the ball on a case. */
export type CaseState = 'not-instrumented' | 'to-review' | 'to-fix' | 'reviewed'

/** What a capture is waiting for. */
export type CaptureStatus = 'to-review' | 'to-fix' | 'validated'

/**
 * Whether a capture still shows what was approved — an overlay on top of the
 * status, never a replacement for it (product.md §3.3).
 *
 * `undefined` is a third answer, and the important one: there is nothing to
 * compare against, because nobody approved this square in this capture's
 * environment (ADR 0017). It does not mean unchanged.
 */
export type Freshness = 'current' | 'to-re-review'

/** Whether a capture is asking to be looked at again. */
export function hasMoved(freshness: Freshness | undefined): boolean {
  return freshness === 'to-re-review'
}

/** The four marks the whole interface draws from. */
export type Tone = 'idle' | 'reviewer' | 'dev' | 'done'

const CASE_TONES: Record<CaseState, Tone> = {
  'not-instrumented': 'idle',
  'to-review': 'reviewer',
  'to-fix': 'dev',
  reviewed: 'done',
}

/** The tone a case state carries. Hue encodes who has to act, not severity. */
export function toneOfCase(state: CaseState): Tone {
  return CASE_TONES[state] ?? 'idle'
}

/** Who has to act on a case in this state. */
export function ballHolder(state: CaseState): 'reviewer' | 'dev' | 'nobody' {
  switch (state) {
    case 'to-review':
      return 'reviewer'
    case 'to-fix':
      return 'dev'
    default:
      return 'nobody'
  }
}

/** What each tone means, in words, for the title a screen reader reads. */
export const TONE_LABELS: Record<Tone, string> = {
  idle: 'not instrumented',
  reviewer: 'to review',
  dev: 'to fix',
  done: 'reviewed',
}
