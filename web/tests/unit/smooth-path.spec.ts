import { describe, expect, it, vi } from 'vitest'
import { pathSmoothPolyline, slicePolylineByFraction } from '../../src/canvas/smoothPath'

function mockCtx() {
  return {
    beginPath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    quadraticCurveTo: vi.fn(),
  } as unknown as CanvasRenderingContext2D
}

describe('smoothPath', () => {
  it('uses quadratic curves for multi-point strokes', () => {
    const ctx = mockCtx()
    pathSmoothPolyline(ctx, [
      { x: 0, y: 0 },
      { x: 10, y: 10 },
      { x: 20, y: 0 },
      { x: 30, y: 10 },
    ])
    expect(ctx.beginPath).toHaveBeenCalled()
    expect(ctx.moveTo).toHaveBeenCalledWith(0, 0)
    expect(ctx.quadraticCurveTo).toHaveBeenCalled()
    expect(ctx.lineTo).toHaveBeenCalled()
  })

  it('keeps two-point strokes as a straight line', () => {
    const ctx = mockCtx()
    pathSmoothPolyline(ctx, [
      { x: 0, y: 0 },
      { x: 10, y: 10 },
    ])
    expect(ctx.lineTo).toHaveBeenCalledWith(10, 10)
    expect(ctx.quadraticCurveTo).not.toHaveBeenCalled()
  })

  it('slices by arc-length fraction with an interpolated end', () => {
    const pts = [
      { x: 0, y: 0 },
      { x: 10, y: 0 },
      { x: 20, y: 0 },
    ]
    const half = slicePolylineByFraction(pts, 0.5)
    expect(half).toHaveLength(2)
    expect(half[0]).toEqual({ x: 0, y: 0 })
    expect(half[1].x).toBeCloseTo(10)
    expect(half[1].y).toBeCloseTo(0)

    const threeQuarter = slicePolylineByFraction(pts, 0.75)
    expect(threeQuarter).toHaveLength(3)
    expect(threeQuarter[2].x).toBeCloseTo(15)
  })
})
