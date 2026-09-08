import type { Point } from '../services/strokeSync'

export type DrawableStroke = {
  points: Point[]
  color: string
  width: number
}

export type DrawStrokesOpts = {
  /** Clear before paint. Default true. */
  clear?: boolean
  /** CSS logical clear size when a DPR transform is active. Required for correct clear. */
  cssWidth: number
  cssHeight: number
}

/**
 * Clear the canvas (in CSS-logical space) and paint strokes, including one-point dots.
 * Caller must have set `ctx.setTransform(dpr, 0, 0, dpr, 0, 0)` when using a DPR backing store.
 */
export function drawStrokes(
  ctx: CanvasRenderingContext2D,
  strokes: DrawableStroke[],
  opts: DrawStrokesOpts,
): void {
  const clear = opts.clear !== false
  if (clear) {
    ctx.clearRect(0, 0, opts.cssWidth, opts.cssHeight)
  }

  for (const s of strokes) {
    if (!s.points.length) continue
    ctx.strokeStyle = s.color
    ctx.fillStyle = s.color
    ctx.lineWidth = s.width
    ctx.lineCap = 'round'
    ctx.lineJoin = 'round'

    if (s.points.length === 1) {
      const p = s.points[0]
      const r = Math.max(0.5, s.width / 2)
      ctx.beginPath()
      ctx.arc(p.x, p.y, r, 0, Math.PI * 2)
      ctx.fill()
      continue
    }

    ctx.beginPath()
    ctx.moveTo(s.points[0].x, s.points[0].y)
    for (let i = 1; i < s.points.length; i += 1) {
      ctx.lineTo(s.points[i].x, s.points[i].y)
    }
    ctx.stroke()
  }
}
