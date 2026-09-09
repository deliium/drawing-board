import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  getLocale,
  initLocale,
  LOCALE_STORAGE_KEY,
  setLocale,
  t,
} from '../../src/i18n'
import {
  CORRECTION_CODES,
  englishCatalogMessage,
  formatCorrectionDisplay,
} from '../../src/i18n/corrections'
import { resolveSubmitLogicalSize } from '../../src/canvas/layout'

describe('i18n', () => {
  beforeEach(() => {
    localStorage.clear()
    setLocale('en')
  })

  it('persists locale preference and sets document.lang', () => {
    setLocale('ja')
    expect(getLocale()).toBe('ja')
    expect(document.documentElement.lang).toBe('ja')
    expect(localStorage.getItem(LOCALE_STORAGE_KEY)).toBe('ja')
    expect(t('nav.practice')).toContain('ひらがな')
  })

  it('falls back to key and warns on missing key', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    expect(t('totally.missing.key')).toBe('totally.missing.key')
    expect(warn).toHaveBeenCalled()
    warn.mockRestore()
  })

  it('initLocale loads storage', () => {
    localStorage.setItem(LOCALE_STORAGE_KEY, 'ja')
    expect(initLocale()).toBe('ja')
    expect(getLocale()).toBe('ja')
  })
})

describe('corrections display map', () => {
  it('covers every stable correction code in EN', () => {
    for (const code of CORRECTION_CODES) {
      const msg = englishCatalogMessage(code, 'あ', 3, 2, 1)
      expect(msg.length).toBeGreaterThan(5)
    }
  })

  it('EN map matches Go catalogMessage strings (Prompt 14 parity)', () => {
    // Mirrors internal/recognize/corrections.go catalogMessage
    const go = {
      empty_strokes: 'No strokes were submitted. Draw the character, then try again.',
      stroke_count_mismatch:
        'Use 3 strokes for 「あ」 (you used 2). Assessed attempts are immutable — start a new attempt and focus on stroke count.',
      stroke_order:
        'Check stroke order — start with the stroke that begins at the top/left for 「あ」. Start a new attempt after this one.',
      start_direction:
        'Start stroke 1 in the same direction as the model (see the tip of the first movement). Retry on a new attempt.',
      end_direction: 'Finish stroke 1 in the expected direction. Retry on a new attempt.',
      relative_placement:
        'Place the strokes closer to their usual positions relative to each other. Start a new attempt focusing on placement.',
      proportions:
        'Adjust the length or size of the strokes so parts of 「あ」 match usual proportions. Retry on a new attempt.',
      shape:
        'The overall shape of 「あ」 still differs from the model — slow down and retrace on a new attempt.',
    } as const
    expect(englishCatalogMessage('empty_strokes', 'あ', 0, 0, 0)).toBe(go.empty_strokes)
    expect(englishCatalogMessage('stroke_count_mismatch', 'あ', 3, 2, 0)).toBe(go.stroke_count_mismatch)
    expect(englishCatalogMessage('stroke_order', 'あ', 0, 0, 0)).toBe(go.stroke_order)
    expect(englishCatalogMessage('start_direction', 'あ', 0, 0, 1)).toBe(go.start_direction)
    expect(englishCatalogMessage('end_direction', 'あ', 0, 0, 1)).toBe(go.end_direction)
    expect(englishCatalogMessage('relative_placement', 'あ', 0, 0, 0)).toBe(go.relative_placement)
    expect(englishCatalogMessage('proportions', 'あ', 0, 0, 0)).toBe(go.proportions)
    expect(englishCatalogMessage('shape', 'あ', 0, 0, 0)).toBe(go.shape)
    expect(englishCatalogMessage('start_direction', 'あ', 0, 0, 0)).toBe(
      'Start the stroke in the same direction as the model. Retry on a new attempt.',
    )
  })

  it('matches Go catalog empty_strokes and switches JA with glyph', () => {
    expect(englishCatalogMessage('empty_strokes', 'あ', 0, 0, 0)).toContain('No strokes')
    const ja = formatCorrectionDisplay('shape', 'EN fallback', { glyph: 'あ' }, 'ja')
    expect(ja).toContain('あ')
    expect(ja).not.toBe('EN fallback')
  })

  it('falls back to API message for unknown codes', () => {
    expect(formatCorrectionDisplay('weird', 'API text', {}, 'en')).toBe('API text')
  })
})

describe('canvas layout submit size', () => {
  it('prefers composable logical size', () => {
    const size = resolveSubmitLogicalSize({
      fromComposable: { width: 360, height: 360 },
      reason: 'test',
    })
    expect(size).toEqual({ width: 360, height: 360 })
  })

  it('falls back when layout unavailable', () => {
    const size = resolveSubmitLogicalSize({ reason: 'test-fallback' })
    expect(size.width).toBe(300)
    expect(size.height).toBe(300)
  })
})
