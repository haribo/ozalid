<script setup lang="ts">
/**
 * The gallery: every primitive of `shared/ui`, with edge-case data (#155,
 * after tribnest's ADR-0013).
 *
 * Dev-only — the route exists only in a dev build, so no end user ever sees
 * this and the production bundle does not carry it. One element = one
 * section: a title, the line saying where its contract lives, and a canvas
 * showing the states worth breaking.
 */
import {
  ActionIcon,
  AdminIcon,
  AppButton,
  EmptyState,
  KindIcon,
  MintedTokenPanel,
  MissingIcon,
  MovedIcon,
  RightsPill,
  StateGauge,
  StateIcon,
  StatePill,
  TextField,
  VariantHead,
} from '@/shared/ui'
import type { CaseState, Tone } from '@/shared/lib'

const CASE_STATES: CaseState[] = ['not-instrumented', 'to-review', 'to-fix', 'reviewed']
const TONES: Tone[] = ['idle', 'reviewer', 'dev', 'done']
const ADMIN_ICONS = [
  'people',
  'keys',
  'remove',
  'swap',
  'add',
  'copy',
  'warn',
  'person',
  'program',
] as const
const ACTION_ICONS = ['check', 'comment', 'accept', 'refuse'] as const

/** Edge cases the gauge must survive: empty slices dropped, one crowded. */
const GAUGE = [
  [
    { tone: 'done' as const, count: 12 },
    { tone: 'reviewer' as const, count: 3 },
    { tone: 'dev' as const, count: 1 },
    { tone: 'idle' as const, count: 0 },
  ],
  [{ tone: 'reviewer' as const, count: 1 }],
  [
    { tone: 'done' as const, count: 214 },
    { tone: 'dev' as const, count: 1 },
  ],
]
</script>

<template>
  <main class="mx-auto max-w-5xl px-6 py-8">
    <h1 class="text-title font-semibold">Design system</h1>
    <p class="mb-8 font-mono text-mono text-slate-500 dark:text-slate-400">
      every primitive of shared/ui, with the data that breaks it · dev build only
    </p>

    <section class="mb-10">
      <h2 class="text-body font-semibold">AppButton</h2>
      <p class="mb-3 font-mono text-label text-slate-500 uppercase">
        shared/ui/AppButton.vue · #155
      </p>
      <div class="flex flex-wrap items-center gap-2">
        <AppButton>Send the link</AppButton>
        <AppButton variant="secondary">Cancel</AppButton>
        <AppButton variant="destructive">refuse</AppButton>
        <AppButton variant="ghost">Use another address</AppButton>
      </div>
      <div class="mt-2 flex flex-wrap items-center gap-2">
        <AppButton size="sm">Add</AppButton>
        <AppButton size="lg">Add</AppButton>
        <AppButton disabled>Create</AppButton>
        <AppButton success>✓ validated</AppButton>
        <AppButton icon variant="secondary" label="retire this token"
          ><AdminIcon name="remove"
        /></AppButton>
        <AppButton variant="secondary"
          >a label long enough to wrap somewhere inconvenient</AppButton
        >
      </div>
    </section>

    <section class="mb-10">
      <h2 class="text-body font-semibold">
        StatePill · StateIcon · StateGauge · MovedIcon · MissingIcon
      </h2>
      <p class="mb-3 font-mono text-label text-slate-500 uppercase">
        the states of product.md §3 · frontend ADR 0004, a status is a glyph on a disc
      </p>
      <div class="mb-2 flex flex-wrap items-center gap-2">
        <StatePill v-for="s in CASE_STATES" :key="s" :state="s" />
      </div>
      <div class="mb-3 flex flex-wrap items-center gap-3">
        <StateIcon v-for="t in TONES" :key="t" :tone="t" :size="16" />
        <MovedIcon :size="14" />
        <MissingIcon :size="14" />
      </div>
      <div class="flex max-w-md flex-col gap-2">
        <StateGauge v-for="(parts, i) in GAUGE" :key="i" :parts="parts" />
      </div>
    </section>

    <section class="mb-10">
      <h2 class="text-body font-semibold">RightsPill · KindIcon · ActionIcon · AdminIcon</h2>
      <p class="mb-3 font-mono text-label text-slate-500 uppercase">
        rights of §8 · comment kinds of §6 · the gestures of the screens
      </p>
      <div class="mb-2 flex flex-wrap items-center gap-2">
        <RightsPill rights="member" />
        <RightsPill rights="reader" />
        <KindIcon kind="defect" :size="15" class="text-amber-700 dark:text-amber-400" />
        <KindIcon kind="improvement" :size="15" class="text-indigo-700 dark:text-indigo-300" />
      </div>
      <div class="flex flex-wrap items-center gap-3 text-slate-600 dark:text-slate-300">
        <ActionIcon v-for="n in ACTION_ICONS" :key="n" :name="n" :size="15" />
        <AdminIcon v-for="n in ADMIN_ICONS" :key="n" :name="n" :size="15" />
      </div>
    </section>

    <section class="mb-10">
      <h2 class="text-body font-semibold">VariantHead</h2>
      <p class="mb-3 font-mono text-label text-slate-500 uppercase">
        the axes a project declared — never hard-coded (ADR 0001)
      </p>
      <div class="flex flex-wrap items-center gap-4">
        <VariantHead label="dark·desktop" :values="{ theme: 'dark', viewport: 'desktop' }" />
        <VariantHead label="light·mobile" :values="{ theme: 'light', viewport: 'mobile' }" />
        <VariantHead compact label="dark·mobile" :values="{ theme: 'dark', viewport: 'mobile' }" />
        <VariantHead
          label="an·axis·nobody·predicted"
          :values="{ locale: 'fr', density: 'crowded' }"
        />
      </div>
    </section>

    <section class="mb-10">
      <h2 class="text-body font-semibold">TextField</h2>
      <p class="mb-3 font-mono text-label text-slate-500 uppercase">
        shared/ui/TextField.vue · two shapes, one contract (#129, #155)
      </p>
      <div class="flex max-w-sm flex-col gap-4">
        <TextField label="name" placeholder="checkout" />
        <TextField label="name" invalid model-value="checkout" />
        <TextField floating label="address" type="email" />
        <TextField floating label="address" model-value="nicolas@ozalid.org" />
      </div>
    </section>

    <section class="mb-10">
      <h2 class="text-body font-semibold">EmptyState</h2>
      <p class="mb-3 font-mono text-label text-slate-500 uppercase">
        an empty list is an answer, not a failure (#109, #112)
      </p>
      <div class="flex flex-col gap-3">
        <EmptyState>nothing here yet</EmptyState>
        <EmptyState prose>
          <b class="mb-1 block text-slate-900 dark:text-slate-100">You belong to no project.</b>
          Ask whoever runs the instance to add you.
        </EmptyState>
      </div>
    </section>

    <section class="mb-10">
      <h2 class="text-body font-semibold">MintedTokenPanel</h2>
      <p class="mb-3 font-mono text-label text-slate-500 uppercase">
        the one moment a token is readable · shown here with fake bytes
      </p>
      <MintedTokenPanel token="ozp_the-gallery-token-nobody-can-spend" @dismiss="() => {}" />
    </section>
  </main>
</template>
