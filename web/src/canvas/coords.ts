/** CSS-logical canvas coordinates and DPR-aware backing-store sizing. */

export type CssPoint = { x: number; y: number }

export type CanvasBackingSize = {
  cssWidth: number
  cssHeight: number
  dpr: number
  backingWidth: number
  backingHeight: number
  clamped: boolean
}

/** Server recognize/canvas bound (logical CSS px), mirrored from internal/limits.MaxCanvasDim. */
export const MAX_LOGICAL_CANVAS_DIM = 2048
export const MIN_LOGICAL_CANVAS_DIM = 1

/**
 * Map a client pointer position into CSS pixels relative to the canvas layout box.
 * Stored stroke points use this space — never multiply by DPR.
 */
export function cssPointFromClient(
  clientX: number,
  clientY: number,
  rect: Pick<DOMRect, 'left' | 'top'>,
): CssPoint {
  return {
    x: clientX - rect.left,
    y: clientY - rect.top,
  }
}

/**
 * Resolve CSS logical size + devicePixelRatio into a backing-store size.
 * Clamps logical dims to practice/recognize limits; does not multiply persisted coords.
 */
export function resolveCanvasBackingSize(opts: {
  cssWidth: number
  cssHeight: number
  dpr?: number
  maxLogicalDim?: number
}): CanvasBackingSize {
  const maxDim = opts.maxLogicalDim ?? MAX_LOGICAL_CANVAS_DIM
  const rawDpr =
    typeof opts.dpr === 'number' && Number.isFinite(opts.dpr) && opts.dpr > 0
      ? opts.dpr
      : 1
  let cssWidth = Number.isFinite(opts.cssWidth) ? opts.cssWidth : 0
  let cssHeight = Number.isFinite(opts.cssHeight) ? opts.cssHeight : 0
  let clamped = false

  if (cssWidth < MIN_LOGICAL_CANVAS_DIM) {
    cssWidth = MIN_LOGICAL_CANVAS_DIM
    clamped = true
  }
  if (cssHeight < MIN_LOGICAL_CANVAS_DIM) {
    cssHeight = MIN_LOGICAL_CANVAS_DIM
    clamped = true
  }
  if (cssWidth > maxDim) {
    cssWidth = maxDim
    clamped = true
  }
  if (cssHeight > maxDim) {
    cssHeight = maxDim
    clamped = true
  }

  const cssW = Math.round(cssWidth)
  const cssH = Math.round(cssHeight)
  return {
    cssWidth: cssW,
    cssHeight: cssH,
    dpr: rawDpr,
    backingWidth: Math.max(1, Math.round(cssW * rawDpr)),
    backingHeight: Math.max(1, Math.round(cssH * rawDpr)),
    clamped,
  }
}

/**
 * Apply backing-store pixel size only when it changes (assignment wipes the bitmap).
 * Does not set the 2d transform — callers apply DPR transform before drawing in CSS space.
 */
export function applyCanvasBackingStore(
  canvas: HTMLCanvasElement,
  size: Pick<CanvasBackingSize, 'backingWidth' | 'backingHeight'>,
): boolean {
  const wiped =
    canvas.width !== size.backingWidth || canvas.height !== size.backingHeight
  if (wiped) {
    canvas.width = size.backingWidth
    canvas.height = size.backingHeight
  }
  return wiped
}

export type LogicalSize = { width: number; height: number }

/**
 * Logical CSS size for recognize POST body (must match stored stroke coordinate space).
 * Accepts a size object or an HTMLCanvasElement (uses layout box, not backing pixels).
 */
export function logicalSizeForRecognize(
  canvasOrSize:
    | HTMLCanvasElement
    | Pick<CanvasBackingSize, 'cssWidth' | 'cssHeight'>
    | LogicalSize,
): LogicalSize {
  if (typeof HTMLCanvasElement !== 'undefined' && canvasOrSize instanceof HTMLCanvasElement) {
    const rect = canvasOrSize.getBoundingClientRect()
    const size = resolveCanvasBackingSize({
      cssWidth: rect.width,
      cssHeight: rect.height,
      dpr: 1,
    })
    return { width: size.cssWidth, height: size.cssHeight }
  }
  const asLogical = canvasOrSize as LogicalSize
  if (typeof asLogical.width === 'number' && typeof asLogical.height === 'number') {
    return {
      width: Math.round(asLogical.width),
      height: Math.round(asLogical.height),
    }
  }
  const asCss = canvasOrSize as Pick<CanvasBackingSize, 'cssWidth' | 'cssHeight'>
  return {
    width: Math.round(asCss.cssWidth),
    height: Math.round(asCss.cssHeight),
  }
}
