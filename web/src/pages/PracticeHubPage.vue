<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { getLesson, HIRAGANA5_LESSON_ID, type Lesson } from '../services/curriculumApi'
import { listProgress, type ProgressItem } from '../services/progressApi'
import type { ApiError } from '../services/apiClient'

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

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
    error.value = apiErr?.message || 'Could not load the lesson.'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
})

const characters = computed(() => lesson.value?.characters ?? [])

function statusFor(id: string): string {
  return progressById.value[id]?.status ?? 'unseen'
}
</script>

<template>
  <div class="hub">
    <header class="top">
      <h1>{{ lesson?.title || 'Hiragana practice' }}</h1>
      <router-link to="/" class="quiet">Free board</router-link>
    </header>

    <p v-if="loading" class="status">Loading…</p>
    <div v-else-if="error" class="status error">
      <p>{{ error }}</p>
      <button type="button" @click="load()">Retry</button>
    </div>
    <p v-else-if="characters.length === 0" class="status">No characters in this lesson.</p>

    <ul v-else class="list">
      <li v-for="ch in characters" :key="ch.id">
        <router-link class="row" :to="`/practice/${encodeURIComponent(ch.id)}`">
          <span class="glyph" lang="ja">{{ ch.glyph }}</span>
          <span class="meta">
            <strong>{{ ch.romanization }}</strong>
            <span class="muted">{{ ch.strokeCount }} strokes · {{ statusFor(ch.id) }}</span>
          </span>
        </router-link>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.hub {
  max-width: 36rem;
  margin: 0 auto;
  padding: 16px;
}
.top {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}
h1 {
  font-size: 1.35rem;
  font-weight: 600;
  margin: 0;
}
.quiet {
  color: #475569;
  text-decoration: none;
  font-size: 0.95rem;
}
.quiet:hover {
  text-decoration: underline;
}
.status {
  text-align: center;
  opacity: 0.85;
}
.error {
  color: #9f1239;
}
.list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 4px;
  border-bottom: 1px solid #e2e8f0;
  text-decoration: none;
  color: inherit;
}
.row:hover {
  background: #f8fafc;
}
.glyph {
  font-size: 2rem;
  width: 2.5rem;
  text-align: center;
  font-family: "Noto Sans JP", "Source Han Sans JP", sans-serif;
}
.meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.muted {
  opacity: 0.7;
  font-size: 0.9rem;
}
button {
  margin-top: 8px;
  padding: 6px 12px;
  cursor: pointer;
}
</style>
