/** The four case states, mirroring the server's vocabulary (ADR 0012). */
export type CaseState = 'not-instrumented' | 'to-review' | 'to-fix' | 'reviewed'

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
