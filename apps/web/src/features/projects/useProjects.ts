import { ref } from 'vue'
import { api, type components } from '@/shared/api'

type Project = components['schemas']['Project']

/**
 * The projects this browser may see.
 *
 * Membership decides it, and administering the instance adds every project's
 * name — never its content (product.md §8.2). Belonging to none is a legitimate
 * state, so an empty list is an answer rather than a failure.
 */
export function useProjects() {
  const projects = ref<Project[]>([])
  const error = ref('')
  const loading = ref(true)

  async function load() {
    loading.value = true
    const result = await api.GET('/projects')
    loading.value = false
    if (result.error) {
      error.value = result.error.title
      return
    }
    projects.value = result.data
  }

  /**
   * Create one. It has no members until somebody is granted a membership on it.
   *
   * The intake policy is not offered here: `per-case` is what the document
   * defaults to, and a project that refuses runs while a case is to-review is a
   * choice to make once somebody has met the need for it (product.md §7).
   */
  async function create(slug: string, name: string): Promise<boolean> {
    error.value = ''
    const result = await api.POST('/projects', {
      body: { slug, name, intakePolicy: 'per-case' },
    })
    if (result.error) {
      error.value = result.error.title
      return false
    }
    await load()
    return true
  }

  return { projects, error, loading, load, create }
}
