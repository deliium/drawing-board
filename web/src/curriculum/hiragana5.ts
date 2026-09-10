import { pathSmoothPolyline } from '../canvas/smoothPath'
import traces from './hiragana5-traces.json'

export type TracePoint = { x: number; y: number }

export type TraceCharacter = {
  glyph: string
  strokes: TracePoint[][]
}

export type Hiragana5TracePack = {
  contentVersion: string
  setId: string
  characters: TraceCharacter[]
}

export const hiragana5Traces = traces as Hiragana5TracePack

/** Scale normalized [0,1] pack polylines into CSS canvas space and paint as faint guides. */
export function drawTraceTemplates(
  ctx: CanvasRenderingContext2D,
  pack: Hiragana5TracePack,
  glyph: string,
  cssWidth: number,
  cssHeight: number,
): void {
  const entry = pack.characters.find((c) => c.glyph === glyph)
  if (!entry) return
  ctx.save()
  ctx.strokeStyle = 'rgba(100, 116, 139, 0.45)'
  ctx.lineWidth = 2
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  for (const stroke of entry.strokes) {
    if (stroke.length === 0) continue
    const scaled = stroke.map((p) => ({ x: p.x * cssWidth, y: p.y * cssHeight }))
    pathSmoothPolyline(ctx, scaled)
    ctx.stroke()
  }
  ctx.restore()
}
