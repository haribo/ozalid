import { describe, expectTypeOf, it } from 'vitest'
import type { paths } from './schema.gen'

type Health = paths['/health']['get']['responses']['200']['content']['application/json']

describe('the generated contract', () => {
  it('describes the health payload the server actually returns', () => {
    expectTypeOf<Health>().toMatchObjectType<{ status: 'ok'; version: string }>()
  })
})
