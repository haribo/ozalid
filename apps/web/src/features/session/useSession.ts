import { ref } from 'vue'
import { api, onUnauthenticated, type paths } from '@/shared/api'

/** The signed-in person, exactly as `GET /me` describes them. */
export type Person = paths['/me']['get']['responses'][200]['content']['application/json']

/**
 * What this browser is, as far as the server is concerned.
 *
 * `expired` is not `out`: it says a session was working a moment ago and
 * stopped. The two lead to different screens, and telling somebody mid-review
 * that they were never signed in would be a lie.
 */
export type Standing = 'unknown' | 'out' | 'in' | 'expired'

/**
 * Endpoints where a 401 is the normal answer rather than a session that died.
 *
 * `GET /me` returns 401 to say "nobody", which is what a signed-out visitor
 * gets on first load. `POST /sign-in/claim` returns 401 for a link that is
 * spent, expired or invented. Neither means a session was lost.
 */
const NOT_AN_EXPIRY = new Set(['/me', '/sign-in/claim'])

/** How the tab that claims a link tells the tab that was waiting. */
const CHANNEL = 'ozalid.session'

const person = ref<Person | null>(null)
const standing = ref<Standing>('unknown')
const linkSent = ref(false)
const busy = ref(false)
const error = ref('')

onUnauthenticated((schemaPath) => {
  if (NOT_AN_EXPIRY.has(schemaPath)) return
  // Only a working session can expire. A signed-out visitor whose request is
  // refused has learned nothing new.
  if (standing.value === 'in') standing.value = 'expired'
})

/** Ask the server who this browser is. The one call that decides `standing`. */
async function refresh(): Promise<void> {
  const result = await api.GET('/me')
  if (result.error) {
    person.value = null
    standing.value = 'out'
    return
  }
  person.value = result.data
  standing.value = 'in'
}

/**
 * Send a sign-in link to an address.
 *
 * A success here says the request was accepted, never that the address has an
 * account — the server deliberately answers the same either way, and the
 * screen must not undo that by showing something different.
 */
async function requestLink(email: string): Promise<void> {
  error.value = ''
  busy.value = true
  const result = await api.POST('/sign-in', { body: { email } })
  busy.value = false
  if (result.error) {
    error.value = result.error.title
    return
  }
  linkSent.value = true
}

/** Spend a link. Answers whether it worked, since a dead link is a screen of its own. */
async function claim(link: string): Promise<boolean> {
  busy.value = true
  const result = await api.POST('/sign-in/claim', { body: { link } })
  busy.value = false
  if (result.error) return false
  await refresh()
  announce()
  return true
}

async function signOut(): Promise<void> {
  await api.POST('/sign-out')
  person.value = null
  standing.value = 'out'
  linkSent.value = false
}

/** Back to the form, after a link was sent to the wrong address. */
function askAgain(): void {
  linkSent.value = false
  error.value = ''
}

/**
 * A link is followed in the tab the mail client opens, not the one that was
 * waiting. Without this the original tab keeps showing an expired session next
 * to a session that works, and the verdict it is holding is never sent.
 */
function announce(): void {
  if (typeof BroadcastChannel === 'undefined') return
  const channel = new BroadcastChannel(CHANNEL)
  // A BroadcastChannel takes no origin: it is already scoped to this one. The
  // rule below is written for window.postMessage, where the argument matters.
  // oxlint-disable-next-line unicorn/require-post-message-target-origin
  channel.postMessage('signed-in')
  channel.close()
}

/** Listen for a sign-in that happened in another tab. Returns the function that stops listening. */
function listen(): () => void {
  if (typeof BroadcastChannel === 'undefined') return () => {}
  const channel = new BroadcastChannel(CHANNEL)
  channel.addEventListener('message', () => {
    void refresh()
  })
  return () => {
    channel.close()
  }
}

/**
 * One session for the whole application, deliberately module-level.
 *
 * The shell, the router and every page must agree on who is signed in; a
 * per-caller copy would let them disagree. This is the exception to how the
 * other composables here work, not the pattern to follow.
 */
export function useSession() {
  return {
    person,
    standing,
    linkSent,
    busy,
    error,
    refresh,
    requestLink,
    claim,
    signOut,
    askAgain,
    listen,
  }
}
