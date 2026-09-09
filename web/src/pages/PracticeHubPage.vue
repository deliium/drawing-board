<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useLocale } from '../composables/useLocale'
import { getLesson, HIRAGANA5_LESSON_ID, type Lesson } from '../services/curriculumApi'
import { listProgress, type ProgressItem } from '../services/progressApi'
import type { ApiError } from '../services/apiClient'

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

const { t, locale } = useLocale()

const lesson = ref<Lesson | null>(null)
const progressById = ref<Record<string, ProgressItem>>({})
const loading = ref(true)
const error = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = null
  if (isDev) console.debug('[PracticeHubPage] mount load')
  try {
    const [lessonRes, progressRes] = await Promise.all([
      getLesson(HIRAGANA5_LESSON_ID),
      listProgress({ lessonId: HIRAGANA5_LESSON_ID }),
    ])
    lesson.value = lessonRes
    const map: Record<string, ProgressItem> = {}
    for (const item of progressRes.items) {
      map[item.characterId] = item
    }
    progressById.value = map
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

function statusFor(id: string): string {
  const status = progressById.value[id]?.status ?? 'unseen'
  const key = `progress.${status}`
  return t(key)
}
</script>

<template>
  <div class="hub">
    <header class="top">
      <h1>{{ title }}</h1>
      <router-link to="/" class="quiet">{{ t('nav.board') }}</router-link>
    </header>

    <p v-if="loading" class="status">{{ t('hub.loading') }}</p>
    <div v-else-if="error" class="status error">
      <p>{{ error }}</p>
      <button type="button" @click="load()">{{ t('hub.retry') }}</button>
    </div>
    <p v-else-if="characters.length === 0" class="status">{{ t('hub.empty') }}</p>

    <ul v-else class="list">
      <li v-for="ch in characters" :key="ch.id">
        <router-link class="row" :to="`/practice/${encodeURIComponent(ch.id)}`">
          <span class="glyph" lang="ja">{{ ch.glyph }}</span>
          <span class="meta">
            <strong :lang="locale === 'ja' ? 'en' : undefined">{{ ch.romanization }}</strong>
            <span class="muted">
              {{ t('hub.strokes', { count: ch.strokeCount }) }} · {{ statusFor(ch.id) }}
            </span>
          </span>
        </router-link>
      </li>
    </ul>
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
  margin-bottom: var(--space-4);
  flex-wrap: wrap;
}

h1 {
  font-size: 1.35rem;
  font-weight: 600;
  margin: 0;
  color: var(--ink);
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
}

.muted {
  color: var(--ink-muted);
  font-size: 0.9rem;
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
</style>
