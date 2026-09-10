import { describe, expect, it, vi } from 'vitest'
import { drawTraceTemplates, hiragana5Traces } from '../../src/curriculum/hiragana5'

function mockCtx() {
  return {
    save: vi.fn(),
    restore: vi.fn(),
    beginPath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    quadraticCurveTo: vi.fn(),
    stroke: vi.fn(),
    strokeStyle: '',
    lineWidth: 0,
    lineCap: '',
    lineJoin: '',
  } as unknown as CanvasRenderingContext2D
}

describe('hiragana5 trace fixtures', () => {
  it('loads five glyphs with matching stroke counts', () => {
    expect(hiragana5Traces.setId).toBe('hiragana5')
    expect(hiragana5Traces.characters).toHaveLength(5)
    const glyphs = hiragana5Traces.characters.map((c) => c.glyph).join('')
    expect(glyphs).toBe('あいうえお')
    const expectedCounts: Record<string, number> = { あ: 3, い: 2, う: 2, え: 2, お: 3 }
    for (const c of hiragana5Traces.characters) {
      expect(c.strokes.length).toBe(expectedCounts[c.glyph])
      for (const s of c.strokes) {
        expect(s.length).toBeGreaterThanOrEqual(5)
        for (const p of s) {
          expect(p.x).toBeGreaterThanOrEqual(0)
          expect(p.x).toBeLessThanOrEqual(1)
          expect(p.y).toBeGreaterThanOrEqual(0)
          expect(p.y).toBeLessThanOrEqual(1)
        }
      }
    }
  })

  it('draws scaled smooth polylines for a glyph', () => {
    const ctx = mockCtx()
    drawTraceTemplates(ctx, hiragana5Traces, 'あ', 300, 300)
    expect(ctx.beginPath).toHaveBeenCalled()
    expect(ctx.moveTo).toHaveBeenCalled()
    expect(ctx.quadraticCurveTo).toHaveBeenCalled()
    expect(ctx.stroke).toHaveBeenCalledTimes(3)
  })
})
