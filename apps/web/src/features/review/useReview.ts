import { ref, shallowRef } from 'vue'
import { api, type components } from '@/shared/api'

type Grid = components['schemas']['Grid']
type Comment = components['schemas']['Comment']

/**
 * Everything a review session does, from the client's side.
 *
 * Each verdict is sent as it is made. Nothing accumulates in the browser: a
 * review started on one machine continues on another, and closing the tab
 * loses nothing. A case with eight captures of twelve judged is not a broken
 * state — it is `to-review`, exactly what the server computes.
 *
 * The single exception is a write the server refused for want of a session:
 * that one is held, and `resume` sends it once the session is back. It waits
 * seconds, not a session, and closing the tab still loses it.
 */
export function useReview(slug: () => string, caseId: () => string) {
  const grid = ref<Grid | null>(null)
  const comments = ref<Comment[]>([])
  const error = ref('')
  const saving = ref(false)
  /** The write a dead session refused, waiting for a live one. */
  const held = shallowRef<(() => Promise<void>) | null>(null)

  async function load() {
    const [evidence, said] = await Promise.all([
      api.GET('/projects/{slug}/cases/{caseId}/captures', {
        params: { path: { slug: slug(), caseId: caseId() } },
      }),
      api.GET('/projects/{slug}/cases/{caseId}/comments', {
        params: { path: { slug: slug(), caseId: caseId() } },
      }),
    ])
    if (evidence.error) {
      error.value = evidence.error.title
      return
    }
    grid.value = evidence.data
    comments.value = said.error ? [] : said.data
  }

  /** Accept one capture — and, switching from a refusal, withdraw the draft
   * remark in the same write: one gesture, never two (ADR 0020). */
  async function accept(stepId: string, variantId: string, withdraw = false) {
    await send({
      accepted: [{ stepId, variantId }],
      ...(withdraw ? { unrefused: [{ stepId, variantId }] } : {}),
    })
  }

  /** Take an acceptance back — a misclick, or a second look (#156). */
  async function unaccept(stepId: string, variantId: string) {
    await send({ unaccepted: [{ stepId, variantId }] })
  }

  /** Refuse: the remark over the variants it covers — and, switching from an
   * acceptance, take it back in the same write. */
  async function refuse(input: {
    stepId: string
    body: string
    variantIds: string[]
    unaccept: boolean
  }) {
    await send({
      comments: [{ stepId: input.stepId, body: input.body, variantIds: input.variantIds }],
      ...(input.unaccept
        ? { unaccepted: input.variantIds.map((variantId) => ({ stepId: input.stepId, variantId })) }
        : {}),
    })
  }

  /** Withdraw a draft refusal: the reviewer's own remark goes with it. */
  async function unrefuse(stepId: string, variantId: string) {
    await send({ unrefused: [{ stepId, variantId }] })
  }

  /** Edit a draft remark — the author reworking their own words (ADR 0020). */
  async function edit(commentId: string, body: string, variantIds: string[]) {
    saving.value = true
    const result = await api.PATCH('/projects/{slug}/comments/{commentId}', {
      params: { path: { slug: slug(), commentId } },
      body: { body, variantIds },
    })
    saving.value = false
    if (result.error) {
      if (expired(result.response)) {
        held.value = () => edit(commentId, body, variantIds)
        return
      }
      error.value = result.error.title
      return
    }
    held.value = null
    await load()
  }

  /** Accept one delivered fix, or refuse it with a remark.
   *
   * Per ref: a comment may carry several issues, each judged on its own
   * round (#138). */
  async function judge(
    commentId: string,
    issueId: string,
    accept: boolean,
    remark: string,
    variantId: string,
  ) {
    saving.value = true
    const result = await api.POST('/projects/{slug}/comments/{commentId}/judgment', {
      params: { path: { slug: slug(), commentId } },
      body: { accept, remark, issueId, variantId },
    })
    saving.value = false
    if (result.error) {
      if (expired(result.response)) {
        held.value = () => judge(commentId, issueId, accept, remark, variantId)
        return
      }
      error.value = result.error.title
      return
    }
    held.value = null
    await load()
  }

  /** Who else holds the case (ADR 0005, #95): null while the caller does —
   * or while nobody does. Filled from the claim's 423 and the case read. */
  const lockedBy = ref<{ name: string; since: string } | null>(null)

  /** Claim the case, or keep holding it — the same call is the heartbeat.
   * A 423 means somebody else reviews: the page turns read-only. */
  async function claim(fresh = false) {
    const result = await api.POST('/projects/{slug}/cases/{caseId}/lock', {
      params: { path: { slug: slug(), caseId: caseId() } },
      body: { fresh },
    })
    if (result.response.status === 423) {
      const found = await api.GET('/projects/{slug}/cases/{caseId}', {
        params: { path: { slug: slug(), caseId: caseId() } },
      })
      lockedBy.value = found.data?.held
        ? { name: found.data.held.name, since: found.data.held.since }
        : { name: 'somebody', since: '' }
      return
    }
    if (!result.error) lockedBy.value = null
  }

  /** Let the case go — leaving the page is letting go. */
  async function release() {
    await api.DELETE('/projects/{slug}/cases/{caseId}/lock', {
      params: { path: { slug: slug(), caseId: caseId() } },
    })
  }

  /** Judge the flow video on screen (ADR 0023): the verdict lands on
   * exactly these bytes, and a new edition resets it. */
  async function judgeRecording(recordingId: string, accept: boolean, remark: string) {
    saving.value = true
    const result = await api.POST('/projects/{slug}/recordings/{recordingId}/judgment', {
      params: { path: { slug: slug(), recordingId } },
      body: { accept, remark: remark || undefined },
    })
    saving.value = false
    if (result.error) {
      if (expired(result.response)) {
        held.value = () => judgeRecording(recordingId, accept, remark)
        return
      }
      error.value = result.error.title
      return
    }
    held.value = null
    await load()
  }

  /** Take the video's verdict back, symmetrically. */
  async function unjudgeRecording(recordingId: string) {
    saving.value = true
    const result = await api.DELETE('/projects/{slug}/recordings/{recordingId}/judgment', {
      params: { path: { slug: slug(), recordingId } },
    })
    saving.value = false
    if (result.error) {
      if (expired(result.response)) {
        held.value = () => unjudgeRecording(recordingId)
        return
      }
      error.value = result.error.title
      return
    }
    held.value = null
    await load()
  }

  /** Take a judgment back — the reviewer reconsiders an acceptance or a
   * refusal, and the ref returns to their court (#167, #171). */
  async function unjudge(commentId: string, issueId: string, variantId: string) {
    saving.value = true
    const result = await api.DELETE('/projects/{slug}/comments/{commentId}/judgment', {
      params: { path: { slug: slug(), commentId } },
      body: { issueId, variantId: variantId || undefined },
    })
    saving.value = false
    if (result.error) {
      if (expired(result.response)) {
        held.value = () => unjudge(commentId, issueId, variantId)
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
    const result = await api.POST('/projects/{slug}/cases/{caseId}/reviews', {
      params: { path: { slug: slug(), caseId: caseId() } },
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

  return {
    grid,
    comments,
    error,
    saving,
    held,
    load,
    claim,
    release,
    lockedBy,
    accept,
    unaccept,
    refuse,
    unrefuse,
    edit,
    judge,
    unjudge,
    judgeRecording,
    unjudgeRecording,
    resume,
  }
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
