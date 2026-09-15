import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CaseTable from './CaseTable.vue'

import type { components } from '@/shared/api'

const cases: components['schemas']['Case'][] = [
  {
    id: 'abc123',
    projectId: 'p1',
    title: 'signing in',
    state: 'to-review',
    captures: { total: 3, accepted: 2, toJudge: 1, refused: 0 },
    lastEdition: '2026-09-10T09:00:00Z',
  } as components['schemas']['Case'],
]

describe('CaseTable', () => {
  it('says the state with the disc alone, the word in the legend (#238)', () => {
    const w = mount(CaseTable, {
      props: { slug: 'atlas', cases },
      global: { stubs: { RouterLink: true } },
    })
    // The row carries no state word — the disc's accessible name does.
    const row = w.find('tbody tr')
    expect(row.text()).not.toContain('to-review')
    expect(row.find('[aria-label="to review"]').exists()).toBe(true)
    // The colour code is taught once, under the table.
    expect(w.text()).toContain('not instrumented')
    expect(w.text()).toContain('refused')
  })
})
