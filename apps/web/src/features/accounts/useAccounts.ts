import { ref } from 'vue'
import { api, type components } from '@/shared/api'

type Account = components['schemas']['Account']

/**
 * The accounts of the instance, from the administration screen's side.
 *
 * Deactivated accounts are kept in the list rather than filtered out: this
 * screen answers "who exists here", and a name in the journal has to resolve to
 * something (product.md §8.2). The access screen of a project answers a
 * different question and drops them.
 */
export function useAccounts() {
  const accounts = ref<Account[]>([])
  const error = ref('')
  const busy = ref(false)

  async function load() {
    const result = await api.GET('/accounts')
    if (result.error) {
      error.value = result.error.title
      return
    }
    accounts.value = result.data
  }

  /** Make an account. It reaches nothing until somebody grants it a membership. */
  async function create(name: string, email: string, isAdmin: boolean): Promise<boolean> {
    error.value = ''
    busy.value = true
    const result = await api.POST('/accounts', { body: { name, email, isAdmin } })
    busy.value = false
    if (result.error) {
      error.value = result.error.title
      return false
    }
    await load()
    return true
  }

  /** Stop an account signing in. Everything it recorded stays and keeps naming it. */
  async function deactivate(accountId: string) {
    busy.value = true
    const result = await api.DELETE('/accounts/{accountId}', { params: { path: { accountId } } })
    busy.value = false
    if (result.error) {
      error.value = result.error.title
      return
    }
    await load()
  }

  return { accounts, error, busy, load, create, deactivate }
}
