import type { Point } from '../services/strokeSync'

export type HitTestStroke = {
  points: Point[]
  width: number
}

/**
 * Hit-test a CSS-logical point against a stroke, including single-point dots
 * and zero-length segments.
 */
export function hitTest(p: Point, s: HitTestStroke): boolean {
  if (!s.points.length) return false
  const threshold = Math.max(6, s.width + 4)

  if (s.points.length === 1) {
    const a = s.points[0]
    return Math.hypot(p.x - a.x, p.y - a.y) <= threshold
  }

  for (let i = 0; i < s.points.length - 1; i += 1) {
    const a = s.points[i]
    const b = s.points[i + 1]
    const dx = b.x - a.x
    const dy = b.y - a.y
    const len2 = dx * dx + dy * dy
    if (len2 === 0) {
      if (Math.hypot(p.x - a.x, p.y - a.y) <= threshold) return true
      continue
    }
    const t = Math.max(0, Math.min(1, ((p.x - a.x) * dx + (p.y - a.y) * dy) / len2))
    const cx = a.x + t * dx
    const cy = a.y + t * dy
    if (Math.hypot(p.x - cx, p.y - cy) <= threshold) return true
  }
  return false
}
