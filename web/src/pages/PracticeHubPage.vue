<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useLocale } from '../composables/useLocale'
import { useRomanizationPreference } from '../composables/useRomanizationPreference'
import { useFeatureFlags } from '../composables/useFeatureFlags'
import { getLesson, HIRAGANA5_LESSON_ID, type Lesson } from '../services/curriculumApi'
import {
  clearPracticeData,
  getProgressNext,
  listProgress,
  type ProgressItem,
  type ProgressNext,
} from '../services/progressApi'
import type { ApiError } from '../services/apiClient'

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

const { t, locale } = useLocale()
const { romanizationVisible } = useRomanizationPreference()
const { features } = useFeatureFlags()

const lesson = ref<Lesson | null>(null)
const progressById = ref<Record<string, ProgressItem>>({})
const suggestion = ref<ProgressNext | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const clearBusy = ref(false)
const clearNotice = ref<string | null>(null)

async function load(opts?: { preserveClearNotice?: boolean }) {
  loading.value = true
  error.value = null
  if (!opts?.preserveClearNotice) clearNotice.value = null
  if (isDev) console.debug('[PracticeHubPage] mount load')
  try {
    const [lessonRes, progressRes, nextRes] = await Promise.all([
      getLesson(HIRAGANA5_LESSON_ID),
      listProgress({ lessonId: HIRAGANA5_LESSON_ID }),
      getProgressNext(HIRAGANA5_LESSON_ID),
    ])
    lesson.value = lessonRes
    const map: Record<string, ProgressItem> = {}
    for (const item of progressRes.items) {
      map[item.characterId] = item
    }
    progressById.value = map
    suggestion.value = nextRes
  } catch (err) {
    const apiErr = err as ApiError
    error.value = apiErr?.message || t('hub.loadError')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
})

const characters = computed(() => lesson.value?.characters ?? [])

const title = computed(() => {
  void locale.value
  if (locale.value === 'ja' && lesson.value?.titleJa) return lesson.value.titleJa
  return lesson.value?.title || t('hub.titleFallback')
})

function masteryFor(id: string): { label: string; reason: string } {
  if (!features.value.progress) {
    return { label: '', reason: '' }
  }
  const m = progressById.value[id]?.mastery
  const state = m?.state || 'not_started'
  const reasonCode = m?.reasonCode || 'no_assessed_attempts'
  return {
    label: t(`mastery.${state}`),
    reason: t(`mastery.reason.${reasonCode}`),
  }
}

function reviewDueHint(id: string): string | null {
  if (!features.value.review) return null
  const r = progressById.value[id]?.review
  if (!r?.isDue) return null
  return t('hub.reviewDue')
}

function formatWhen(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  try {
    return new Intl.DateTimeFormat(locale.value === 'ja' ? 'ja' : 'en', {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(d)
  } catch {
    return iso
  }
}

function suggestionHref(): string | null {
  const id = suggestion.value?.characterId
  if (!id) return null
  return `/practice/${encodeURIComponent(id)}`
}

function suggestionText(): string {
  const s = suggestion.value
  if (!s) return ''
  if (!s.characterId) {
    let primary = t(`next.reason.${s.reasonCode}`)
    if (primary === `next.reason.${s.reasonCode}`) primary = t('hub.suggestedNone')
    if (s.reasonCode === 'all_caught_up' && s.nextDueAt) {
      return `${primary} ${t('hub.suggestedCaughtUpNext', { when: formatWhen(s.nextDueAt) })}`
    }
    return primary
  }
  const why = t(`next.reason.${s.reasonCode}`)
  const glyph = s.glyph || ''
  return glyph ? `${glyph} — ${why}` : why
}

async function onClearPractice() {
  if (clearBusy.value) return
  if (!window.confirm(t('hub.clearConfirm'))) return
  clearBusy.value = true
  clearNotice.value = null
  if (isDev) console.debug('[PracticeHubPage] clear practice data')
  try {
    await clearPracticeData()
    clearNotice.value = t('hub.clearSuccess')
    await load({ preserveClearNotice: true })
  } catch (err) {
    const apiErr = err as ApiError
    clearNotice.value = apiErr?.message || t('hub.clearError')
  } finally {
    clearBusy.value = false
  }
}
</script>

<template>
  <div class="hub">
    <header class="top">
      <h1>{{ title }}</h1>
    </header>

    <p v-if="features.progress" class="hint">{{ t('hub.masteryHint') }}</p>

    <p v-if="loading" class="status">{{ t('hub.loading') }}</p>
    <div v-else-if="error" class="status error">
      <p>{{ error }}</p>
      <button type="button" @click="load()">{{ t('hub.retry') }}</button>
    </div>
    <template v-else>
      <section v-if="suggestion" class="suggest" aria-live="polite">
        <h2>{{ t('hub.suggestedNext') }}</h2>
        <p>
          <router-link v-if="suggestionHref()" class="suggest-link" :to="suggestionHref()!">
            {{ suggestionText() }}
          </router-link>
          <span v-else>{{ suggestionText() }}</span>
        </p>
      </section>

      <p v-if="characters.length === 0" class="status">{{ t('hub.empty') }}</p>

      <ul v-else class="list">
        <li v-for="ch in characters" :key="ch.id">
          <router-link class="row" :to="`/practice/${encodeURIComponent(ch.id)}`">
            <span class="glyph" lang="ja">{{ ch.glyph }}</span>
            <span class="meta">
              <strong
                v-if="romanizationVisible"
                :lang="locale === 'ja' ? 'en' : undefined"
              >{{ ch.romanization }}</strong>
              <span class="muted">
                {{ t('hub.strokes', { count: ch.strokeCount }) }}
                <template v-if="features.progress && masteryFor(ch.id).label">
                  · {{ masteryFor(ch.id).label }}
                </template>
              </span>
              <span v-if="features.progress && masteryFor(ch.id).reason" class="reason">{{
                masteryFor(ch.id).reason
              }}</span>
              <span v-if="reviewDueHint(ch.id)" class="review-hint">{{ reviewDueHint(ch.id) }}</span>
            </span>
          </router-link>
        </li>
      </ul>

      <div v-if="features.progress" class="privacy">
        <button type="button" class="dangerish" :disabled="clearBusy" @click="onClearPractice">
          {{ clearBusy ? t('hub.clearing') : t('hub.clearData') }}
        </button>
        <p v-if="clearNotice" class="notice" role="status" aria-live="polite">{{ clearNotice }}</p>
      </div>
    </template>
  </div>
</template>

<style scoped>
.hub {
  max-width: var(--content-max);
  margin: 0 auto;
  padding: var(--space-4) 0;
}

.top {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-2);
  flex-wrap: wrap;
}

h1 {
  font-size: 1.35rem;
  font-weight: 600;
  margin: 0;
  color: var(--ink);
}

h2 {
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 var(--space-1);
  color: var(--ink);
}

.hint {
  color: var(--ink-muted);
  font-size: 0.9rem;
  margin: 0 0 var(--space-3);
}

.quiet {
  color: var(--ink-muted);
  text-decoration: none;
  font-size: 0.95rem;
  min-height: var(--touch-min);
  display: inline-flex;
  align-items: center;
}

.quiet:hover {
  text-decoration: underline;
  color: var(--ink);
}

.suggest {
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--rule);
}

.suggest-link {
  color: var(--ink);
  text-decoration: none;
  min-height: var(--touch-min);
  display: inline-flex;
  align-items: center;
}

.suggest-link:hover {
  text-decoration: underline;
}

.status {
  text-align: center;
  color: var(--ink-muted);
}

.error {
  color: var(--danger);
}

.list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-height: var(--touch-min);
  padding: var(--space-3) var(--space-1);
  border-bottom: 1px solid var(--rule);
  text-decoration: none;
  color: inherit;
}

.row:hover {
  background: color-mix(in srgb, var(--paper-raised) 70%, transparent);
}

.glyph {
  font-size: 2rem;
  width: 2.5rem;
  text-align: center;
  font-family: var(--font-ja);
}

.meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.muted {
  color: var(--ink-muted);
  font-size: 0.9rem;
}

.reason {
  color: var(--ink-muted);
  font-size: 0.85rem;
}

.review-hint {
  color: var(--ink-muted);
  font-size: 0.85rem;
}

.privacy {
  margin-top: var(--space-5);
  padding-top: var(--space-3);
  border-top: 1px solid var(--rule);
}

.dangerish {
  color: var(--danger);
}

.notice {
  margin-top: var(--space-2);
  color: var(--ink-muted);
  font-size: 0.95rem;
}

button {
  margin-top: var(--space-2);
  min-height: var(--touch-min);
  padding: 0 var(--space-3);
  cursor: pointer;
  font: inherit;
  border: 1px solid var(--rule);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
}

button:disabled {
  opacity: 0.6;
  cursor: default;
}
</style>
