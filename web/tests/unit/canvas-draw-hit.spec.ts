import { describe, expect, it, vi } from 'vitest'
import { drawStrokes } from '../../src/canvas/drawStrokes'
import { hitTest } from '../../src/canvas/hitTest'

function mockCtx() {
  return {
    clearRect: vi.fn(),
    beginPath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    stroke: vi.fn(),
    fill: vi.fn(),
    arc: vi.fn(),
    strokeStyle: '',
    fillStyle: '',
    lineWidth: 0,
    lineCap: '',
    lineJoin: '',
  } as unknown as CanvasRenderingContext2D
}

describe('drawStrokes', () => {
  it('clears in CSS space and paints polylines', () => {
    const ctx = mockCtx()
    drawStrokes(
      ctx,
      [{ points: [{ x: 1, y: 1 }, { x: 5, y: 5 }], color: '#111', width: 3 }],
      { cssWidth: 300, cssHeight: 300 },
    )
    expect(ctx.clearRect).toHaveBeenCalledWith(0, 0, 300, 300)
    expect(ctx.beginPath).toHaveBeenCalled()
    expect(ctx.moveTo).toHaveBeenCalledWith(1, 1)
    expect(ctx.lineTo).toHaveBeenCalledWith(5, 5)
    expect(ctx.stroke).toHaveBeenCalled()
  })

  it('renders a single-point stroke as a filled dot', () => {
    const ctx = mockCtx()
    drawStrokes(
      ctx,
      [{ points: [{ x: 40, y: 50 }], color: '#1d4ed8', width: 4 }],
      { cssWidth: 300, cssHeight: 300 },
    )
    expect(ctx.arc).toHaveBeenCalledWith(40, 50, 2, 0, Math.PI * 2)
    expect(ctx.fill).toHaveBeenCalled()
    expect(ctx.stroke).not.toHaveBeenCalled()
  })
})

describe('hitTest', () => {
  it('hits a single-point stroke within threshold', () => {
    const stroke = { points: [{ x: 100, y: 100 }], width: 4 }
    expect(hitTest({ x: 102, y: 101 }, stroke)).toBe(true)
    expect(hitTest({ x: 200, y: 200 }, stroke)).toBe(false)
  })

  it('hits polyline segments and zero-length segments', () => {
    const stroke = {
      points: [
        { x: 0, y: 0 },
        { x: 100, y: 0 },
        { x: 100, y: 0 },
      ],
      width: 2,
    }
    expect(hitTest({ x: 50, y: 2 }, stroke)).toBe(true)
    expect(hitTest({ x: 100, y: 3 }, stroke)).toBe(true)
    expect(hitTest({ x: 50, y: 40 }, stroke)).toBe(false)
  })
})
