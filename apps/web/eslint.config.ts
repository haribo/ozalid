import pluginVue from 'eslint-plugin-vue'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import prettier from 'eslint-config-prettier'

// Layer boundaries are architecture, not style: this config enforces
// frontend ADR 0002. A slice reaches downward, never sideways, never upward.
const LAYERS = ['app', 'pages', 'widgets', 'features', 'shared']

const upward = (layer: string) =>
  LAYERS.slice(0, LAYERS.indexOf(layer)).map((l) => `@/${l}/*`)

export default defineConfigWithVueTs(
  { ignores: ['dist/**', 'node_modules/**', 'src/shared/api/schema.gen.ts'] },
  pluginVue.configs['flat/recommended'],
  vueTsConfigs.recommended,
  // Last, so prettier owns formatting and eslint owns correctness. Without
  // this the two disagree on line breaks and every file reports warnings
  // nobody can fix.
  prettier,
  {
    // Four type roles and nothing else (#145): an arbitrary text size is how
    // the scale rotted the first time. The roles live in style.css's @theme.
    files: ['src/**/*.vue'],
    rules: {
      'vue/no-restricted-class': ['error', '/^text-\\[/', 'border-dashed'],
      // A raw <button> is how the cursor, the disabled look and the focus
      // ring drifted apart (#155): AppButton is the one way. Toggle chips are
      // the documented exception, disabled inline where they live.
      'vue/no-restricted-html-elements': [
        'error',
        { element: 'button', message: 'use AppButton (shared/ui) — #155' },
      ],
    },
  },
  {
    // The primitives themselves: the one raw <button>, the one dashed border.
    files: ['src/shared/ui/AppButton.vue', 'src/shared/ui/EmptyState.vue', 'src/shared/ui/StateIcon.vue'],
    rules: {
      'vue/no-restricted-html-elements': 'off',
      'vue/no-restricted-class': ['error', '/^text-\\[/'],
    },
  },
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
