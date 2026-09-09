<script setup lang="ts">
import { computed } from 'vue'
import { useLocale } from '../../composables/useLocale'
import { useRomanizationPreference } from '../../composables/useRomanizationPreference'
import type { LessonCharacter } from '../../services/curriculumApi'
import PronunciationAudioButton from './PronunciationAudioButton.vue'

const props = defineProps<{
  character: LessonCharacter
}>()

const { t, locale } = useLocale()
const { romanizationVisible } = useRomanizationPreference()

const description = computed(() => {
  void locale.value
  if (locale.value === 'ja' && props.character.descriptionJa) {
    return props.character.descriptionJa
  }
  return props.character.descriptionEn
})

const guidance = computed(() => {
  void locale.value
  if (locale.value === 'ja' && props.character.guidanceJa) {
    return props.character.guidanceJa
  }
  return props.character.guidanceEn || ''
})

const meaning = computed(() => {
  void locale.value
  if (locale.value === 'ja' && props.character.example.meaningJa) {
    return props.character.example.meaningJa
  }
  return props.character.example.meaningEn
})

const strokesLabel = computed(() => t('intro.strokes', { count: props.character.strokeCount }))

const audioSrc = computed(() => {
  const ref = props.character.pronunciation?.audioRef
  if (!ref || typeof ref !== 'string' || !ref.trim()) return null
  return ref.trim()
})
</script>

<template>
  <section class="intro" :aria-label="t('intro.aria')">
    <p class="glyph" lang="ja">{{ character.glyph }}</p>
    <p class="meta">
      <span
        v-if="romanizationVisible"
        :lang="locale === 'ja' ? 'en' : undefined"
      >{{ character.romanization }}</span>
      <span v-if="character.pronunciation?.ipa" class="muted" lang="en">
        {{ character.pronunciation.ipa }}
      </span>
      <span v-if="character.pronunciation?.jaHint" class="muted" lang="ja">
        {{ character.pronunciation.jaHint }}
      </span>
    </p>
    <p class="meta">{{ strokesLabel }}</p>
    <PronunciationAudioButton
      v-if="audioSrc"
      :audio-ref="audioSrc"
      :glyph="character.glyph"
    />
    <p
      v-if="guidance"
      class="guidance"
      :lang="locale === 'ja' && character.guidanceJa ? 'ja' : 'en'"
    >
      {{ guidance }}
    </p>
    <p v-if="description" class="desc" :lang="locale === 'ja' && character.descriptionJa ? 'ja' : 'en'">
      {{ description }}
    </p>
    <p class="example">
      <span>{{ t('intro.example') }}</span>
      <span lang="ja"> {{ character.example.word }}</span>
      <span class="muted">
        (
        <template v-if="romanizationVisible">
          <span :lang="locale === 'ja' ? 'en' : undefined">{{ character.example.romanization }}</span>
          —
        </template>
        <span :lang="locale === 'ja' && character.example.meaningJa ? 'ja' : 'en'">{{ meaning }}</span>
        )
      </span>
    </p>
  </section>
</template>

<style scoped>
.intro {
  text-align: center;
  padding: var(--space-3) var(--space-2);
  position: relative;
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

.guidance {
  max-width: 36rem;
  margin: var(--space-2) auto;
  color: var(--ink);
  font-weight: 500;
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
