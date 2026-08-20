import pluginVue from 'eslint-plugin-vue'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'

// Layer boundaries are architecture, not style: this config enforces
// frontend ADR 0002. A slice reaches downward, never sideways, never upward.
const LAYERS = ['app', 'pages', 'widgets', 'features', 'shared']

const upward = (layer: string) =>
  LAYERS.slice(0, LAYERS.indexOf(layer)).map((l) => `@/${l}/*`)

export default defineConfigWithVueTs(
  { ignores: ['dist/**', 'node_modules/**'] },
  pluginVue.configs['flat/recommended'],
  vueTsConfigs.recommended,
  ...LAYERS.map((layer) => ({
    files: [`src/${layer}/**/*.{ts,vue}`],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            ...upward(layer).map((p) => ({
              group: [p],
              message: `${layer} may not import upward (frontend ADR 0002)`,
            })),
            {
              group: [`@/${layer}/*/*`],
              message: `import a slice through its index.ts, never its internals (frontend ADR 0002)`,
            },
          ],
        },
      ],
    },
  })),
)
