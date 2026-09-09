import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, nextTick } from 'vue'
import ComparisonOverlay from '../../src/components/practice/ComparisonOverlay.vue'
import JourneyStatusBanner from '../../src/components/practice/JourneyStatusBanner.vue'
import CharacterIntroPanel from '../../src/components/practice/CharacterIntroPanel.vue'
import { initLocale, setLocale } from '../../src/i18n'
import { useRomanizationPreference } from '../../src/composables/useRomanizationPreference'
import { stubCanvasContext } from '../helpers/stubCanvasContext'

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

const baseCharacter = {
  id: 'hira:あ',
  glyph: 'あ',
  romanization: 'a',
  strokeCount: 3,
  pronunciation: { ipa: '/a/', jaHint: '「あ」の音', audioRef: '/audio/hiragana5/a.mp3' },
  descriptionEn: 'Open vowel',
  descriptionJa: '開いた母音',
  guidanceEn: 'Listen then write three strokes.',
  guidanceJa: '聞いてから三画。',
  example: { word: 'あさ', romanization: 'asa', meaningEn: 'morning', meaningJa: '朝' },
  sortKey: 1,
  position: 1,
}

describe('ComparisonOverlay', () => {
  let canvasSpy: ReturnType<typeof stubCanvasContext>

  beforeEach(() => {
    canvasSpy = stubCanvasContext()
    initLocale()
    setLocale('en')
  })
  afterEach(() => {
    canvasSpy.mockRestore()
  })

  it('shows at most two feedback messages and Match label', async () => {
    const m = await mount(ComparisonOverlay, {
      glyph: 'あ',
      learnerStrokes: [],
      pass: false,
      score: 0.4,
      feedback: [
        { rank: 1, code: 'a', message: 'First tip' },
        { rank: 2, code: 'b', message: 'Second tip' },
        { rank: 3, code: 'c', message: 'Third tip' },
      ],
    })
    expect(m.text()).toContain('Match')
    expect(m.text()).toContain('First tip')
    expect(m.text()).toContain('Second tip')
    expect(m.text()).not.toContain('Third tip')
    expect(m.text()).not.toMatch(/confidence/i)
    m.unmount()
  })
})

describe('JourneyStatusBanner', () => {
  it('renders loading and error banners', async () => {
    const loading = await mount(JourneyStatusBanner, { stage: 'loading' })
    expect(loading.text()).toContain('Loading')
    loading.unmount()

    const err = await mount(JourneyStatusBanner, {
      stage: 'intro',
      banner: { kind: 'error', message: 'Network failed' },
    })
    expect(err.text()).toContain('Network failed')
    err.unmount()
  })
})

describe('CharacterIntroPanel', () => {
  beforeEach(() => {
    initLocale()
    setLocale('en')
    localStorage.setItem('romanizationVisible:v1', 'true')
    useRomanizationPreference().setRomanizationVisible(true)
  })

  it('renders pedagogy fields including guidance and play control', async () => {
    const m = await mount(CharacterIntroPanel, { character: baseCharacter })
    expect(m.text()).toContain('あ')
    expect(m.text()).toMatch(/3 strokes|3画/)
    expect(m.text()).toContain('あさ')
    expect(m.text()).toContain('morning')
    expect(m.text()).toContain('Listen then write three strokes.')
    expect(m.root.querySelector('button')).toBeTruthy()
    expect(m.text()).toMatch(/Play pronunciation|発音/)
    m.unmount()
  })

  it('omits play control when audioRef is null', async () => {
    const m = await mount(CharacterIntroPanel, {
      character: {
        ...baseCharacter,
        pronunciation: { ipa: '/a/', jaHint: '「あ」の音', audioRef: null },
      },
    })
    expect(m.root.querySelector('audio')).toBeNull()
    expect(m.text()).not.toMatch(/Play pronunciation|発音を再生/)
    m.unmount()
  })

  it('hides romanization when preference is off', async () => {
    useRomanizationPreference().setRomanizationVisible(false)
    const m = await mount(CharacterIntroPanel, { character: baseCharacter })
    expect(m.text()).toContain('あ')
    expect(m.text()).toContain('あさ')
    expect(m.text()).toContain('morning')
    expect(m.text()).not.toContain('asa')
    // Character Hepburn is omitted from the tree (IPA /a/ may still appear).
    const meta = m.root.querySelector('.meta')
    expect(meta?.textContent || '').not.toMatch(/^\s*a\s/)
    m.unmount()
  })

  it('switches guidance with locale', async () => {
    const m = await mount(CharacterIntroPanel, { character: baseCharacter })
    expect(m.text()).toContain('Listen then write three strokes.')
    setLocale('ja')
    await nextTick()
    expect(m.text()).toContain('聞いてから三画。')
    m.unmount()
  })

  it('soft-fails when audio src errors', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const m = await mount(CharacterIntroPanel, {
      character: {
        ...baseCharacter,
        pronunciation: {
          ...baseCharacter.pronunciation,
          audioRef: '/audio/hiragana5/missing-broken.mp3',
        },
      },
    })
    const audio = m.root.querySelector('audio') as HTMLAudioElement
    expect(audio).toBeTruthy()
    audio.dispatchEvent(new Event('error'))
    await nextTick()
    expect(m.text()).toMatch(/Audio unavailable|音声を再生できません/)
    warn.mockRestore()
    m.unmount()
  })
})
