export type PathPoint = { x: number; y: number }

/**
 * Paint a polyline with midpoint quadratic curves so sparse control points
 * do not show as sharp corner joints. Two-point strokes stay as a single line.
 * Caller sets strokeStyle / lineWidth / lineCap / lineJoin and must call stroke().
 */
export function pathSmoothPolyline(
  ctx: CanvasRenderingContext2D,
  points: PathPoint[],
): void {
  if (points.length === 0) return
  ctx.beginPath()
  ctx.moveTo(points[0].x, points[0].y)
  if (points.length === 1) return
  if (points.length === 2) {
    ctx.lineTo(points[1].x, points[1].y)
    return
  }

  // Soften the first segment into the first midpoint, then chain midpoints.
  ctx.lineTo((points[0].x + points[1].x) / 2, (points[0].y + points[1].y) / 2)

  for (let i = 1; i < points.length - 1; i++) {
    const midX = (points[i].x + points[i + 1].x) / 2
    const midY = (points[i].y + points[i + 1].y) / 2
    ctx.quadraticCurveTo(points[i].x, points[i].y, midX, midY)
  }

  const last = points[points.length - 1]
  ctx.lineTo(last.x, last.y)
}

/** Cumulative chord lengths; lengths[i] is distance along path to points[i]. */
function cumulativeLengths(points: PathPoint[]): number[] {
  const lengths = new Array(points.length).fill(0)
  for (let i = 1; i < points.length; i++) {
    const dx = points[i].x - points[i - 1].x
    const dy = points[i].y - points[i - 1].y
    lengths[i] = lengths[i - 1] + Math.hypot(dx, dy)
  }
  return lengths
}

/**
 * Prefix of a polyline covering fraction `t` of total chord length (0..1),
 * including an interpolated end point so animation advances smoothly.
 */
export function slicePolylineByFraction(points: PathPoint[], t: number): PathPoint[] {
  if (points.length <= 1) return points.slice()
  const clamped = Math.min(1, Math.max(0, t))
  if (clamped <= 0) return [points[0]]
  if (clamped >= 1) return points.slice()

  const lengths = cumulativeLengths(points)
  const total = lengths[lengths.length - 1]
  if (total <= 0) return [points[0]]

  const target = total * clamped
  for (let i = 1; i < points.length; i++) {
    if (lengths[i] < target) continue
    const segLen = lengths[i] - lengths[i - 1]
    const local = segLen > 0 ? (target - lengths[i - 1]) / segLen : 1
    const a = points[i - 1]
    const b = points[i]
    const end: PathPoint = {
      x: a.x + (b.x - a.x) * local,
      y: a.y + (b.y - a.y) * local,
    }
    return points.slice(0, i).concat(end)
  }
  return points.slice()
}
