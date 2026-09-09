<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useLocale } from '../composables/useLocale'
import { formatCorrectionDisplay } from '../i18n/corrections'
import {
  listAttempts,
  type AttemptHistoryItem,
} from '../services/attemptsApi'
import type { ApiError } from '../services/apiClient'

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

const { t, locale } = useLocale()
const route = useRoute()

const items = ref<AttemptHistoryItem[]>([])
const nextCursor = ref<string | null>(null)
const loading = ref(true)
const loadingMore = ref(false)
const error = ref<string | null>(null)

const characterFilter = computed(() => {
  const raw = route.query.characterId
  return typeof raw === 'string' && raw.trim() ? raw.trim() : ''
})

async function load(reset: boolean) {
  if (reset) {
    loading.value = true
    error.value = null
    items.value = []
    nextCursor.value = null
  } else {
    loadingMore.value = true
  }
  if (isDev) console.debug('[PracticeHistoryPage] load', { reset, cursor: nextCursor.value })
  try {
    const res = await listAttempts({
      characterId: characterFilter.value || undefined,
      limit: 20,
      cursor: reset ? undefined : nextCursor.value || undefined,
    })
    items.value = reset ? res.items : [...items.value, ...res.items]
    nextCursor.value = res.nextCursor ?? null
  } catch (err) {
    const apiErr = err as ApiError
    error.value = apiErr?.message || t('history.loadError')
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

onMounted(() => {
  void load(true)
})

watch(characterFilter, () => {
  void load(true)
})

function formatWhen(iso: string): string {
  void locale.value
  try {
    return new Date(iso).toLocaleString(locale.value === 'ja' ? 'ja' : 'en')
  } catch {
    return iso
  }
}

function outcomeLabel(item: AttemptHistoryItem): string {
  if (item.status === 'abandoned') return t('history.abandoned')
  if (item.pass) return t('history.pass')
  return t('history.fail')
}

function feedbackLine(item: AttemptHistoryItem): string {
  if (!item.feedback?.length) return ''
  return item.feedback
    .slice(0, 2)
    .map((fb) => formatCorrectionDisplay(fb.code, fb.message, { glyph: item.glyph }))
    .filter(Boolean)
    .join(' · ')
}
</script>

<template>
  <div class="history">
    <header class="top">
      <h1>{{ t('history.title') }}</h1>
      <router-link to="/practice" class="quiet">{{ t('history.toHub') }}</router-link>
    </header>

    <div class="filters" role="group" :aria-label="t('history.title')">
      <router-link
        class="chip"
        :class="{ active: !characterFilter }"
        to="/practice/history"
      >
        {{ t('history.filterAll') }}
      </router-link>
      <router-link
        v-if="characterFilter"
        class="chip active"
        :to="`/practice/history?characterId=${encodeURIComponent(characterFilter)}`"
      >
        {{ t('history.filterCharacter') }}
      </router-link>
    </div>

    <p v-if="loading" class="status">{{ t('history.loading') }}</p>
    <div v-else-if="error" class="status error">
      <p>{{ error }}</p>
      <button type="button" @click="load(true)">{{ t('history.retry') }}</button>
    </div>
    <div v-else-if="items.length === 0" class="status empty">
      <p>{{ t('history.empty') }}</p>
      <router-link class="cta" to="/practice">{{ t('history.practiceCta') }}</router-link>
    </div>

    <ul v-else class="list">
      <li v-for="item in items" :key="item.id">
        <router-link class="row" :to="`/practice/${encodeURIComponent(item.characterId)}`">
          <span class="glyph" lang="ja">{{ item.glyph }}</span>
          <span class="meta">
            <strong>{{ outcomeLabel(item) }}</strong>
            <span class="muted">{{ formatWhen(item.startedAt) }}</span>
            <span
              v-if="item.score != null && item.scoreKind === 'match'"
              class="muted"
            >
              {{ t('history.match', { score: item.score.toFixed(2) }) }}
            </span>
            <span v-if="feedbackLine(item)" class="feedback">{{ feedbackLine(item) }}</span>
          </span>
        </router-link>
      </li>
    </ul>

    <div v-if="nextCursor && !loading" class="more">
      <button type="button" :disabled="loadingMore" @click="load(false)">
        {{ loadingMore ? t('history.loading') : t('history.loadMore') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.history {
  max-width: var(--content-max);
  margin: 0 auto;
  padding: var(--space-4) 0;
}

.top {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-3);
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
  min-height: var(--touch-min);
  display: inline-flex;
  align-items: center;
}

.filters {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.chip {
  min-height: var(--touch-min);
  display: inline-flex;
  align-items: center;
  padding: 0 var(--space-3);
  border: 1px solid var(--rule);
  border-radius: var(--radius-sm);
  text-decoration: none;
  color: var(--ink-muted);
  font-size: 0.95rem;
}

.chip.active {
  color: var(--ink);
  border-color: var(--ink-muted);
  background: color-mix(in srgb, var(--paper-raised) 80%, transparent);
}

.status {
  text-align: center;
  color: var(--ink-muted);
}

.error {
  color: var(--danger);
}

.empty .cta {
  display: inline-flex;
  margin-top: var(--space-3);
  min-height: var(--touch-min);
  align-items: center;
  color: var(--ink);
}

.list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.row {
  display: flex;
  gap: var(--space-3);
  align-items: flex-start;
  min-height: var(--touch-min);
  padding: var(--space-3) var(--space-1);
  border-bottom: 1px solid var(--rule);
  text-decoration: none;
  color: inherit;
}

.glyph {
  font-size: 1.75rem;
  width: 2.25rem;
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

.feedback {
  font-size: 0.9rem;
  color: var(--ink);
}

.more {
  display: flex;
  justify-content: center;
  margin-top: var(--space-4);
}

button {
  min-height: var(--touch-min);
  padding: 0 var(--space-3);
  cursor: pointer;
  font: inherit;
  border: 1px solid var(--rule);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
}
</style>
