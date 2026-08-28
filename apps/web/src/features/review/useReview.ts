import { ref, shallowRef } from 'vue'
import { api, type components } from '@/shared/api'

type Grid = components['schemas']['Grid']
type Comment = components['schemas']['Comment']

/**
 * Everything a review session does, from the client's side.
 *
 * Each verdict is sent as it is made. Nothing accumulates in the browser: a
 * review started on one machine continues on another, and closing the tab
 * loses nothing. A case with eight squares of twelve judged is not a broken
 * state — it is `to-review`, exactly what the server computes.
 *
 * The single exception is a write the server refused for want of a session:
 * that one is held, and `resume` sends it once the session is back. It waits
 * seconds, not a session, and closing the tab still loses it.
 */
export function useReview(caseId: () => string) {
  const grid = ref<Grid | null>(null)
  const comments = ref<Comment[]>([])
  const error = ref('')
  const saving = ref(false)
  /** The write a dead session refused, waiting for a live one. */
  const held = shallowRef<(() => Promise<void>) | null>(null)

  async function load() {
    const [evidence, said] = await Promise.all([
      api.GET('/cases/{caseId}/captures', { params: { path: { caseId: caseId() } } }),
      api.GET('/cases/{caseId}/comments', { params: { path: { caseId: caseId() } } }),
    ])
    if (evidence.error) {
      error.value = evidence.error.title
      return
    }
    grid.value = evidence.data
    comments.value = said.error ? [] : said.data
  }

  /** Validate one square: looked at, nothing to say. */
  async function validate(stepId: string, variantId: string) {
    await send({ validated: [{ stepId, variantId }] })
  }

  /** Report something on one step, over the variants it applies to. */
  async function comment(input: {
    stepId: string
    kind: 'defect' | 'improvement'
    body: string
    variantIds: string[]
  }) {
    await send({ comments: [input] })
  }

  /** Accept a delivered fix, or refuse it with a remark. */
  async function judge(commentId: string, accept: boolean, remark?: string) {
    saving.value = true
    const result = await api.POST('/comments/{commentId}/judgment', {
      params: { path: { commentId } },
      body: { accept, remark },
    })
    saving.value = false
    if (result.error) {
      if (expired(result.response)) {
        held.value = () => judge(commentId, accept, remark)
        return
      }
      error.value = result.error.title
      return
    }
    held.value = null
    await load()
  }

  /**
   * Send again what a dead session refused.
   *
   * Called when the session comes back, so the verdict that was in flight is
   * not something to remember and redo by hand.
   */
  async function resume() {
    const again = held.value
    if (!again) return
    held.value = null
    await again()
  }

  async function send(body: components['schemas']['ReviewSave']) {
    saving.value = true
    const result = await api.POST('/cases/{caseId}/reviews', {
      params: { path: { caseId: caseId() } },
      body,
    })
    saving.value = false
    if (result.error) {
      if (expired(result.response)) {
        held.value = () => send(body)
        return
      }
      error.value = result.error.title
      return
    }
    held.value = null
    // Reload rather than patch locally: the server decides what a verdict
    // means for the case, and guessing here would be a second copy of a rule
    // that lives in exactly one place (ADR 0012).
    await load()
  }

  return { grid, comments, error, saving, held, load, validate, comment, judge, resume }
}

/**
 * A refusal for want of a credential, as opposed to anything else that failed.
 *
 * It carries no message to the reviewer: the shell already says the session
 * expired, and a second sentence saying the same thing in other words is how
 * an interface stops being read.
 */
function expired(response: Response): boolean {
  return response.status === 401
}
