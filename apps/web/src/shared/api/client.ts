import createClient from 'openapi-fetch'
import type { paths } from './schema.gen'

/**
 * The only way this application talks to the server.
 *
 * Its types come from the OpenAPI document (backend ADR 0002), so a contract
 * change surfaces here as a type error rather than as a runtime failure. A
 * hand-written fetch against an endpoint bypasses that and is a defect.
 */
export const api = createClient<paths>({ baseUrl: '/api' })
