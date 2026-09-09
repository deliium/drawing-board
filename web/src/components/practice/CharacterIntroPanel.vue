<script setup lang="ts">
import type { LessonCharacter } from '../services/curriculumApi'

defineProps<{
  character: LessonCharacter
}>()
</script>

<template>
  <section class="intro" aria-label="Character introduction">
    <p class="glyph" lang="ja">{{ character.glyph }}</p>
    <p class="meta">
      <span>{{ character.romanization }}</span>
      <span v-if="character.pronunciation?.ipa" class="muted">
        {{ character.pronunciation.ipa }}
      </span>
      <span v-if="character.pronunciation?.jaHint" class="muted" lang="ja">
        {{ character.pronunciation.jaHint }}
      </span>
    </p>
    <p class="meta">{{ character.strokeCount }} strokes</p>
    <p v-if="character.descriptionEn" class="desc">{{ character.descriptionEn }}</p>
    <p class="example" lang="ja">
      Example: {{ character.example.word }}
      <span class="muted">({{ character.example.romanization }} — {{ character.example.meaningEn }})</span>
    </p>
  </section>
</template>

<style scoped>
.intro {
  text-align: center;
  padding: 12px 8px;
}
.glyph {
  font-size: 4.5rem;
  line-height: 1.1;
  margin: 0 0 8px;
  font-family: "Noto Sans JP", "Source Han Sans JP", sans-serif;
}
.meta {
  margin: 4px 0;
  display: flex;
  gap: 10px;
  justify-content: center;
  flex-wrap: wrap;
}
.desc {
  max-width: 36rem;
  margin: 10px auto;
  opacity: 0.9;
}
.example {
  margin: 8px 0 0;
}
.muted {
  opacity: 0.7;
}
</style>
