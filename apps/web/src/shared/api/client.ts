import createClient from 'openapi-fetch'
import type { paths } from './schema.gen'

/**
 * The only way this application talks to the server.
 *
 * Its types come from the OpenAPI document (backend ADR 0002), so a contract
 * change surfaces here as a type error rather than as a runtime failure. A
 * hand-written fetch against an endpoint bypasses that and is a defect.
 */
export const api = createClient<paths>({
  // Absolute, though the server is the one that served this page: a relative
  // base leaves it to whoever builds the Request to resolve, and outside a
  // browser nothing does.
  baseUrl: new URL('/api', globalThis.location.origin).toString(),
  // Read at call time rather than captured at import time. A global read once,
  // before anything has had a chance to wrap it, is a global nobody can wrap.
  fetch: (...call) => globalThis.fetch(...call),
})

/** Handed the endpoint that was refused, since a 401 does not mean the same thing on all of them. */
type Refused = (schemaPath: string) => void

const watchers = new Set<Refused>()

/**
 * Watch the requests the server answers with 401.
 *
 * There is one place to notice a credential the server no longer accepts,
 * because there are already five call sites and the sixth would forget. What a
 * refusal means is decided by the watcher, not here.
 *
 * Returns the function that stops watching.
 */
export function onUnauthenticated(watch: Refused): () => void {
  watchers.add(watch)
  return () => {
    watchers.delete(watch)
  }
}

api.use({
  onResponse({ response, schemaPath }) {
    if (response.status !== 401) return
    for (const watch of watchers) watch(schemaPath)
  },
})
