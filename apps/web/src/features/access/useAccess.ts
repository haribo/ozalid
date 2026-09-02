import { ref } from 'vue'
import { api, type components } from '@/shared/api'

type Membership = components['schemas']['Membership']
type Rights = components['schemas']['Rights']
type ServiceToken = components['schemas']['ServiceToken']
type MintedToken = components['schemas']['MintedToken']

/**
 * Who reaches one project, and with what.
 *
 * People and programs in one list, because they see the same thing: a service
 * account holds a membership like anyone else (ADR 0019). A deactivated account
 * never appears — the server drops it, since this answers "who reaches this
 * project" and a deactivated account reaches nothing.
 */
export function useAccess(slug: () => string) {
  const members = ref<Membership[]>([])
  const minted = ref<MintedToken | null>(null)
  const error = ref('')
  const busy = ref(false)

  async function load() {
    const result = await api.GET('/projects/{slug}/members', {
      params: { path: { slug: slug() } },
    })
    if (result.error) {
      error.value = result.error.title
      return
    }
    members.value = result.data
  }

  /** Put somebody on the project, or change what they may do there. */
  async function grant(accountId: string, rights: Rights) {
    busy.value = true
    const result = await api.PUT('/projects/{slug}/members/{accountId}', {
      params: { path: { slug: slug(), accountId } },
      body: { rights },
    })
    busy.value = false
    if (result.error) {
      error.value = result.error.title
      return
    }
    await load()
  }

  async function revoke(accountId: string) {
    busy.value = true
    const result = await api.DELETE('/projects/{slug}/members/{accountId}', {
      params: { path: { slug: slug(), accountId } },
    })
    busy.value = false
    if (result.error) {
      error.value = result.error.title
      return
    }
    await load()
  }

  /**
   * Make a program an account on this project, with its first token.
   *
   * One call, because the three things it does are one thing: a service account
   * with no membership reaches nothing and a service account with no token
   * cannot present itself, so neither is a state worth existing in. It belongs
   * to this project and never moves (ADR 0018) — which is why there is nothing
   * to pick from, only something to name.
   *
   * The token comes back here and nowhere else.
   */
  async function createProgram(name: string, rights: Rights, tokenLabel: string): Promise<boolean> {
    error.value = ''
    busy.value = true
    const result = await api.POST('/projects/{slug}/service-accounts', {
      params: { path: { slug: slug() } },
      body: { name, rights, tokenLabel },
    })
    busy.value = false
    if (result.error) {
      error.value = result.error.title
      return false
    }
    minted.value = result.data.token
    await load()
    return true
  }

  /** Let go of the minted token. Nothing kept it, so it is gone for good. */
  function forgetMinted() {
    minted.value = null
  }

  /** Retire a program. Its tokens stop opening anything at once. */
  async function retireProgram(serviceAccountId: string) {
    busy.value = true
    const result = await api.DELETE('/projects/{slug}/service-accounts/{serviceAccountId}', {
      params: { path: { slug: slug(), serviceAccountId } },
    })
    busy.value = false
    if (result.error) {
      error.value = result.error.title
      return
    }
    await load()
  }

  return {
    members,
    minted,
    error,
    busy,
    load,
    grant,
    revoke,
    createProgram,
    forgetMinted,
    retireProgram,
  }
}

/**
 * One program's credentials.
 *
 * `token` on a minted one is the single moment it can be read: only its hash is
 * stored, so nothing here can give it back. It is held in memory, never
 * anywhere else, and a reload loses it — which is the honest behaviour.
 */
export function useTokens(slug: () => string, serviceAccountId: () => string) {
  const tokens = ref<ServiceToken[]>([])
  const minted = ref<MintedToken | null>(null)
  const error = ref('')
  const busy = ref(false)

  async function load() {
    const result = await api.GET('/projects/{slug}/service-accounts/{serviceAccountId}/tokens', {
      params: { path: { slug: slug(), serviceAccountId: serviceAccountId() } },
    })
    if (result.error) {
      error.value = result.error.title
      return
    }
    tokens.value = result.data
  }

  /** Add a token. It does not retire the others: that is what makes a rotation survivable. */
  async function mint(label: string) {
    error.value = ''
    busy.value = true
    const result = await api.POST('/projects/{slug}/service-accounts/{serviceAccountId}/tokens', {
      params: { path: { slug: slug(), serviceAccountId: serviceAccountId() } },
      body: { label },
    })
    busy.value = false
    if (result.error) {
      error.value = result.error.title
      return
    }
    minted.value = result.data
    await load()
  }

  async function retire(tokenId: string) {
    busy.value = true
    const result = await api.DELETE(
      '/projects/{slug}/service-accounts/{serviceAccountId}/tokens/{tokenId}',
      { params: { path: { slug: slug(), serviceAccountId: serviceAccountId(), tokenId } } },
    )
    busy.value = false
    if (result.error) {
      error.value = result.error.title
      return
    }
    await load()
  }

  /** Let go of the minted token. Nothing kept it, so it is gone for good. */
  function dismiss() {
    minted.value = null
  }

  return { tokens, minted, error, busy, load, mint, retire, dismiss }
}
