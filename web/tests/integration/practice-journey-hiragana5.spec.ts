import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, effectScope, nextTick, ref } from 'vue'
import CharacterIntroPanel from '../../src/components/practice/CharacterIntroPanel.vue'
import ComparisonOverlay from '../../src/components/practice/ComparisonOverlay.vue'
import { usePracticeJourney } from '../../src/composables/usePracticeJourney'
import { hiragana5Traces } from '../../src/curriculum/hiragana5'
import { stubCanvasContext } from '../helpers/stubCanvasContext'

vi.mock('../../src/services/curriculumApi', () => ({
  HIRAGANA5_LESSON_ID: 'lesson:hiragana5',
  getLesson: vi.fn(),
}))

vi.mock('../../src/services/progressApi', () => ({
  listProgress: vi.fn(),
}))

vi.mock('../../src/services/attemptsApi', () => ({
  createAttempt: vi.fn(),
  getAttempt: vi.fn(),
  submitAttempt: vi.fn(),
  assessAttempt: vi.fn(),
  getAttemptAssessment: vi.fn(),
  abandonAttempt: vi.fn(),
}))

import { getLesson } from '../../src/services/curriculumApi'
import { listProgress } from '../../src/services/progressApi'
import { assessAttempt, createAttempt, submitAttempt } from '../../src/services/attemptsApi'

const glyphs = [
  { id: 'hira:あ', glyph: 'あ', romanization: 'a', strokeCount: 3, word: 'あさ', meaning: 'morning' },
  { id: 'hira:い', glyph: 'い', romanization: 'i', strokeCount: 2, word: 'いぬ', meaning: 'dog' },
  { id: 'hira:う', glyph: 'う', romanization: 'u', strokeCount: 2, word: 'うみ', meaning: 'sea' },
  { id: 'hira:え', glyph: 'え', romanization: 'e', strokeCount: 2, word: 'えき', meaning: 'station' },
  { id: 'hira:お', glyph: 'お', romanization: 'o', strokeCount: 3, word: 'おと', meaning: 'sound' },
] as const

function lessonWithAll() {
  return {
    id: 'lesson:hiragana5',
    code: 'hiragana5',
    title: 'Hiragana vowels (あいうえお)',
    setId: 'hiragana5',
    contentVersion: hiragana5Traces.contentVersion,
    characters: glyphs.map((g, i) => ({
      id: g.id,
      glyph: g.glyph,
      romanization: g.romanization,
      strokeCount: g.strokeCount,
      pronunciation: { ipa: `/${g.romanization}/`, audioRef: null },
      descriptionEn: `Description for ${g.glyph}`,
      example: {
        word: g.word,
        romanization: g.romanization === 'a' ? 'asa' : g.romanization,
        meaningEn: g.meaning,
      },
      sortKey: i + 1,
      position: i + 1,
    })),
  }
}

async function mount(Comp: object, props: Record<string, unknown>) {
  const root = document.createElement('div')
  document.body.appendChild(root)
  const app = createApp(Comp, props)
  app.mount(root)
  await nextTick()
  return {
    root,
    text: () => root.textContent || '',
    unmount: () => {
      app.unmount()
      root.remove()
    },
  }
}

describe('hiragana5 practice journey integration', () => {
  let canvasSpy: ReturnType<typeof stubCanvasContext>

  beforeEach(() => {
    canvasSpy = stubCanvasContext()
  })

  afterEach(() => {
    canvasSpy.mockRestore()
    vi.clearAllMocks()
    sessionStorage.clear()
  })

  it.each(glyphs)(
    'intro fields for $glyph (strokeCount=$strokeCount example=$word)',
    async (g) => {
      vi.mocked(getLesson).mockResolvedValue(lessonWithAll() as never)
      vi.mocked(listProgress).mockResolvedValue({ items: [] })

      const scope = effectScope()
      const characterId = ref(g.id)
      const api = scope.run(() => usePracticeJourney({ characterId }))!
      await api.load()
      expect(api.stage.value).toBe('intro')
      expect(api.character.value?.glyph).toBe(g.glyph)
      expect(api.character.value?.strokeCount).toBe(g.strokeCount)
      expect(api.character.value?.example.word).toBe(g.word)
      expect(api.character.value?.example.meaningEn).toBe(g.meaning)

      const panel = await mount(CharacterIntroPanel, { character: api.character.value! })
      expect(panel.text()).toContain(g.glyph)
      expect(panel.text()).toContain(String(g.strokeCount))
      expect(panel.text()).toContain(g.word)
      expect(panel.text()).toContain(g.meaning)
      panel.unmount()
      scope.stop()
    },
  )

  it.each(glyphs)('pass and fail paths for $glyph', async (g) => {
    vi.mocked(getLesson).mockResolvedValue(lessonWithAll() as never)
    vi.mocked(listProgress).mockResolvedValue({ items: [] })
    vi.mocked(createAttempt).mockResolvedValue({
      id: '10000000-1000-4000-8000-100000000000',
      characterId: g.id,
      status: 'draft',
      startedAt: 't',
      clientAttemptId: 'c-pass',
    })
    vi.mocked(submitAttempt).mockResolvedValue({
      id: '10000000-1000-4000-8000-100000000000',
      status: 'submitted',
      submittedAt: 't',
      strokeCount: 1,
      width: 300,
      height: 300,
    })
    vi.mocked(assessAttempt).mockResolvedValueOnce({
      attemptId: '10000000-1000-4000-8000-100000000000',
      characterId: g.id,
      status: 'assessed',
      pass: false,
      score: 0.25,
      scoreKind: 'match',
      assessor: 'target_compare',
      setId: 'hiragana5',
      reasons: ['shape'],
      feedback: [
        { rank: 1, code: 'shape_mismatch', message: 'Check stroke order' },
        { rank: 2, code: 'start_end', message: 'Watch start and end points' },
        { rank: 3, code: 'extra', message: 'Should be hidden' },
      ],
    })

    const scope = effectScope()
    const characterId = ref(g.id)
    const api = scope.run(() => usePracticeJourney({ characterId }))!
    await api.load()
    await api.start()
    await api.continueFromAnimate()
    api.onStrokesChanged([
      { points: [{ x: 1, y: 1 }], color: '#000', width: 2, clientId: 't1', startedAtUnixMs: 1 },
    ])
    api.nextFromTrace()
    api.onStrokesChanged([
      { points: [{ x: 2, y: 2 }], color: '#000', width: 2, clientId: 'f1', startedAtUnixMs: 2 },
    ])
    await api.submit()
    expect(api.stage.value).toBe('result')
    expect(api.assessment.value?.pass).toBe(false)
    expect(api.feedback.value).toHaveLength(2)
    expect(api.feedback.value.map((f) => f.message)).not.toContain('Should be hidden')

    const overlay = await mount(ComparisonOverlay, {
      glyph: g.glyph,
      learnerStrokes: api.strokes.value,
      pass: false,
      score: 0.25,
      feedback: api.feedback.value,
    })
    expect(overlay.text()).toContain('Match')
    expect(overlay.text()).toContain('Check stroke order')
    expect(overlay.text()).not.toContain('Should be hidden')
    expect(overlay.text()).not.toContain('confidence')
    expect(overlay.text()).not.toContain('candidates')
    overlay.unmount()

    vi.mocked(createAttempt).mockResolvedValue({
      id: '10100000-1010-4010-8010-101000000000',
      characterId: g.id,
      status: 'draft',
      startedAt: 't',
      clientAttemptId: 'c-retry',
    })
    vi.mocked(assessAttempt).mockResolvedValueOnce({
      attemptId: '10100000-1010-4010-8010-101000000000',
      characterId: g.id,
      status: 'assessed',
      pass: true,
      score: 0.91,
      scoreKind: 'match',
      assessor: 'target_compare',
      setId: 'hiragana5',
      reasons: [],
      feedback: [],
    })
    const prevClient = api.clientAttemptId.value
    await api.retry()
    expect(api.clientAttemptId.value).not.toBe(prevClient)
    api.onStrokesChanged([
      { points: [{ x: 1, y: 1 }], color: '#000', width: 2, clientId: 't2', startedAtUnixMs: 3 },
    ])
    api.nextFromTrace()
    api.onStrokesChanged([
      { points: [{ x: 3, y: 3 }], color: '#000', width: 2, clientId: 'f2', startedAtUnixMs: 4 },
    ])
    await api.submit()
    expect(api.assessment.value?.pass).toBe(true)
    api.complete()
    expect(api.stage.value).toBe('complete')
    scope.stop()
  })

  it('surfaces backend error without claiming pass', async () => {
    vi.mocked(getLesson).mockRejectedValue({ status: 500, message: 'server error' })
    vi.mocked(listProgress).mockResolvedValue({ items: [] })
    const scope = effectScope()
    const characterId = ref('hira:あ')
    const api = scope.run(() => usePracticeJourney({ characterId }))!
    await api.load()
    expect(api.stage.value).toBe('error')
    expect(api.banner.value?.message).toMatch(/server error|Could not load/i)
    expect(api.assessment.value).toBeNull()
    scope.stop()
  })

  it('submit failure does not claim pass', async () => {
    vi.mocked(getLesson).mockResolvedValue(lessonWithAll() as never)
    vi.mocked(listProgress).mockResolvedValue({ items: [] })
    vi.mocked(createAttempt).mockResolvedValue({
      id: '20000000-2000-4000-8000-200000000000',
      characterId: 'hira:あ',
      status: 'draft',
      startedAt: 't',
      clientAttemptId: 'c-err',
    })
    vi.mocked(submitAttempt).mockRejectedValue({ status: 503, message: 'unavailable' })

    const scope = effectScope()
    const characterId = ref('hira:あ')
    const api = scope.run(() => usePracticeJourney({ characterId }))!
    await api.load()
    await api.start()
    await api.continueFromAnimate()
    api.onStrokesChanged([
      { points: [{ x: 1, y: 1 }], color: '#000', width: 2, clientId: 't1', startedAtUnixMs: 1 },
    ])
    api.nextFromTrace()
    api.onStrokesChanged([
      { points: [{ x: 2, y: 2 }], color: '#000', width: 2, clientId: 'f1', startedAtUnixMs: 2 },
    ])
    await api.submit()
    expect(api.stage.value).toBe('freewrite')
    expect(api.assessment.value).toBeNull()
    expect(api.banner.value?.kind).toBe('error')
    scope.stop()
  })
})
