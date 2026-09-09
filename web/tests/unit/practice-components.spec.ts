import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { createApp, nextTick } from 'vue'
import ComparisonOverlay from '../../src/components/practice/ComparisonOverlay.vue'
import JourneyStatusBanner from '../../src/components/practice/JourneyStatusBanner.vue'
import CharacterIntroPanel from '../../src/components/practice/CharacterIntroPanel.vue'
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

describe('ComparisonOverlay', () => {
  let canvasSpy: ReturnType<typeof stubCanvasContext>

  beforeEach(() => {
    canvasSpy = stubCanvasContext()
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
  it('renders pedagogy fields', async () => {
    const m = await mount(CharacterIntroPanel, {
      character: {
        id: 'hira:あ',
        glyph: 'あ',
        romanization: 'a',
        strokeCount: 3,
        pronunciation: { ipa: '/a/', jaHint: '「あ」の音' },
        descriptionEn: 'Open vowel',
        example: { word: 'あさ', romanization: 'asa', meaningEn: 'morning' },
        sortKey: 1,
        position: 1,
      },
    })
    expect(m.text()).toContain('あ')
    expect(m.text()).toContain('3 strokes')
    expect(m.text()).toContain('あさ')
    expect(m.text()).toContain('morning')
    m.unmount()
  })
})
