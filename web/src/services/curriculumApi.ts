import { apiFetch } from './apiClient'
import { hiragana5Traces } from '../curriculum/hiragana5'

export type Pronunciation = {
  ipa?: string
  jaHint?: string
  audioRef?: string | null
  pitch?: string
}

export type LessonExample = {
  word: string
  romanization: string
  meaningEn: string
}

export type LessonCharacter = {
  id: string
  glyph: string
  romanization: string
  strokeCount: number
  pronunciation: Pronunciation
  descriptionEn: string
  example: LessonExample
  sortKey: number
  position: number
}

export type Lesson = {
  id: string
  code: string
  title: string
  setId: string
  contentVersion: string
  characters: LessonCharacter[]
}

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function debug(...args: unknown[]) {
  if (isDev) console.debug('[curriculumApi]', ...args)
}

/** Published lesson pedagogy for the practice hub / journey (geometry stays in client fixtures). */
export async function getLesson(id: string): Promise<Lesson> {
  debug('getLesson', id)
  try {
    const lesson = await apiFetch<Lesson>(`/api/lessons/${encodeURIComponent(id)}`)
    if (
      isDev &&
      lesson.contentVersion &&
      hiragana5Traces.contentVersion &&
      lesson.contentVersion !== hiragana5Traces.contentVersion
    ) {
      console.warn(
        '[curriculumApi] contentVersion mismatch lesson=%s fixture=%s',
        lesson.contentVersion,
        hiragana5Traces.contentVersion,
      )
    }
    debug('getLesson ok', lesson.id, 'characters', lesson.characters?.length ?? 0)
    return lesson
  } catch (err) {
    debug('getLesson error', err)
    throw err
  }
}

export const HIRAGANA5_LESSON_ID = 'lesson:hiragana5'
