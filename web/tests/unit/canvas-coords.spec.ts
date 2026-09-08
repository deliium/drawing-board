import { describe, expect, it, vi } from 'vitest'
import {
  MAX_LOGICAL_CANVAS_DIM,
  applyCanvasBackingStore,
  cssPointFromClient,
  logicalSizeForRecognize,
  resolveCanvasBackingSize,
} from '../../src/canvas/coords'

describe('canvas coords', () => {
  it('maps client coordinates into CSS-logical canvas space', () => {
    expect(cssPointFromClient(150, 220, { left: 50, top: 20 })).toEqual({ x: 100, y: 200 })
  })

  it('resolves backing store as css * dpr without rewriting logical size', () => {
    const size = resolveCanvasBackingSize({ cssWidth: 300, cssHeight: 300, dpr: 2 })
    expect(size.cssWidth).toBe(300)
    expect(size.cssHeight).toBe(300)
    expect(size.dpr).toBe(2)
    expect(size.backingWidth).toBe(600)
    expect(size.backingHeight).toBe(600)
    expect(size.clamped).toBe(false)
  })

  it('clamps logical dims to recognize/practice max', () => {
    const size = resolveCanvasBackingSize({
      cssWidth: MAX_LOGICAL_CANVAS_DIM + 100,
      cssHeight: 10,
      dpr: 2,
    })
    expect(size.cssWidth).toBe(MAX_LOGICAL_CANVAS_DIM)
    expect(size.backingWidth).toBe(MAX_LOGICAL_CANVAS_DIM * 2)
    expect(size.clamped).toBe(true)
  })

  it('applyCanvasBackingStore wipes only when size changes', () => {
    const canvas = {
      width: 300,
      height: 300,
    } as HTMLCanvasElement
    expect(
      applyCanvasBackingStore(canvas, { backingWidth: 300, backingHeight: 300 }),
    ).toBe(false)
    expect(
      applyCanvasBackingStore(canvas, { backingWidth: 600, backingHeight: 600 }),
    ).toBe(true)
    expect(canvas.width).toBe(600)
    expect(canvas.height).toBe(600)
  })

  it('logicalSizeForRecognize uses CSS size, not backing pixels', () => {
    const fromSize = logicalSizeForRecognize({ cssWidth: 300, cssHeight: 300 })
    expect(fromSize).toEqual({ width: 300, height: 300 })

    const canvas = {
      getBoundingClientRect: () => ({
        width: 300,
        height: 300,
        left: 0,
        top: 0,
        right: 300,
        bottom: 300,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      }),
      width: 600,
      height: 600,
    } as HTMLCanvasElement
    // instanceof check needs a real element in jsdom — stub via width/height object path
    expect(logicalSizeForRecognize({ width: 300, height: 300 })).toEqual({
      width: 300,
      height: 300,
    })
    void canvas
  })

  it('logicalSizeForRecognize reads layout box from a real canvas element', () => {
    const el = document.createElement('canvas')
    vi.spyOn(el, 'getBoundingClientRect').mockReturnValue({
      width: 300,
      height: 280,
      left: 10,
      top: 20,
      right: 310,
      bottom: 300,
      x: 10,
      y: 20,
      toJSON: () => ({}),
    })
    el.width = 600
    el.height = 560
    expect(logicalSizeForRecognize(el)).toEqual({ width: 300, height: 280 })
  })
})
