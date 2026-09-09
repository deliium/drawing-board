<script setup lang="ts">
import { computed } from 'vue'
import { useLocale } from '../../composables/useLocale'
import type { LessonCharacter } from '../../services/curriculumApi'

const props = defineProps<{
  character: LessonCharacter
}>()

const { t, locale } = useLocale()

const description = computed(() => {
  void locale.value
  if (locale.value === 'ja' && props.character.descriptionJa) {
    return props.character.descriptionJa
  }
  return props.character.descriptionEn
})

const meaning = computed(() => {
  void locale.value
  if (locale.value === 'ja' && props.character.example.meaningJa) {
    return props.character.example.meaningJa
  }
  return props.character.example.meaningEn
})

const strokesLabel = computed(() => t('intro.strokes', { count: props.character.strokeCount }))
</script>

<template>
  <section class="intro" :aria-label="t('intro.aria')">
    <p class="glyph" lang="ja">{{ character.glyph }}</p>
    <p class="meta">
      <span :lang="locale === 'ja' ? 'en' : undefined">{{ character.romanization }}</span>
      <span v-if="character.pronunciation?.ipa" class="muted" lang="en">
        {{ character.pronunciation.ipa }}
      </span>
      <span v-if="character.pronunciation?.jaHint" class="muted" lang="ja">
        {{ character.pronunciation.jaHint }}
      </span>
    </p>
    <p class="meta">{{ strokesLabel }}</p>
    <p v-if="description" class="desc" :lang="locale === 'ja' && character.descriptionJa ? 'ja' : 'en'">
      {{ description }}
    </p>
    <p class="example">
      <span>{{ t('intro.example') }}</span>
      <span lang="ja"> {{ character.example.word }}</span>
      <span class="muted">
        (<span :lang="locale === 'ja' ? 'en' : undefined">{{ character.example.romanization }}</span>
        —
        <span :lang="locale === 'ja' && character.example.meaningJa ? 'ja' : 'en'">{{ meaning }}</span>)
      </span>
    </p>
  </section>
</template>

<style scoped>
.intro {
  text-align: center;
  padding: var(--space-3) var(--space-2);
}

.glyph {
  font-size: clamp(3rem, 18vw, 4.5rem);
  line-height: 1.1;
  margin: 0 0 var(--space-2);
  font-family: var(--font-ja);
}

.meta {
  margin: var(--space-1) 0;
  display: flex;
  gap: var(--space-3);
  justify-content: center;
  flex-wrap: wrap;
}

.desc {
  max-width: 36rem;
  margin: var(--space-3) auto;
  color: var(--ink);
}

.example {
  margin: var(--space-2) 0 0;
}

.muted {
  color: var(--ink-muted);
}
</style>
