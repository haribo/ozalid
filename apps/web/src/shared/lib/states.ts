/**
 * The vocabulary the server speaks, and the one visual language that carries
 * it. The same shape means the same thing at every level: a ticked circle on a
 * category gauge, on a case row and under a capture all say "judged, nothing
 * to say" (ADR 0012).
 */

/** Who holds the ball on a case. */
export type CaseState = 'not-instrumented' | 'to-review' | 'refused' | 'accepted'

/** What a capture is waiting for. */
export type CaptureStatus = 'to-review' | 'refused' | 'accepted' | 'moved'

/** The four marks the whole interface draws from. */
export type Tone = 'idle' | 'reviewer' | 'dev' | 'done'

const CASE_TONES: Record<CaseState, Tone> = {
  'not-instrumented': 'idle',
  'to-review': 'reviewer',
  refused: 'dev',
  accepted: 'done',
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
    case 'refused':
      return 'dev'
    default:
      return 'nobody'
  }
}

/** What each tone means, in words, for the title a screen reader reads. */
export const TONE_LABELS: Record<Tone, string> = {
  idle: 'not instrumented',
  reviewer: 'to review',
  dev: 'refused',
  done: 'accepted',
}

/**
 * How a settled capture is marked, wherever one is shown: the grid, the recap,
 * and the carousel stage since ADR 0026.
 *
 * One capture, one question — *has this been judged?* — and one answer, so two
 * surfaces cannot drift into saying it differently. A widget may not import
 * another widget (frontend ADR 0002), which is why this lives a layer down
 * rather than in whichever widget drew it first.
 */
export const SETTLED_INK: Record<string, string> = {
  accepted: 'text-emerald-700 dark:text-emerald-400',
  refused: 'text-amber-700 dark:text-amber-400',
}

export const SETTLED_TONE: Record<string, Tone> = { accepted: 'done', refused: 'dev' }

export const SETTLED_LABEL: Record<string, string> = {
  accepted: 'accepted',
  refused: 'refused',
}

/** The disc's own ground, so a mark laid over a capture is never taken for one
 * of the product's own colours (frontend ADR 0004). */
export const SETTLED_GROUND: Record<string, string> = {
  accepted: 'bg-emerald-50 dark:bg-emerald-950',
  refused: 'bg-amber-50 dark:bg-amber-950',
}

/**
 * The three readings a capture has for the one question a mark answers.
 *
 * `moved` is not judged: the bytes on display are ones nobody has looked at,
 * whatever was said about the ones before them.
 */
export function readingOf(status: string): 'moved' | 'judged' | 'pending' {
  if (status === 'moved') return 'moved'
  if (status === 'accepted' || status === 'refused') return 'judged'
  return 'pending'
}

/** Judged and settled: it steps back, because full intensity is reserved for
 * what still needs eyes. */
export function isSettled(status: string) {
  return readingOf(status) === 'judged'
}
