/** Shared CSS-logical canvas sizing helpers (Prompt 14 fluid canvas). */

import {
  logicalSizeForRecognize,
  resolveCanvasBackingSize,
  type LogicalSize,
} from './coords'

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function layoutDebug(...args: unknown[]) {
  if (isDev) console.debug('[canvas.layout]', ...args)
}

/** Read the live CSS-logical size of a canvas (layout box, not backing pixels). */
export function readLogicalCanvasSize(
  canvas: HTMLCanvasElement | null | undefined,
): LogicalSize | null {
  if (!canvas) return null
  const rect = canvas.getBoundingClientRect()
  if (!Number.isFinite(rect.width) || !Number.isFinite(rect.height)) return null
  if (rect.width < 1 || rect.height < 1) return null
  const size = resolveCanvasBackingSize({
    cssWidth: rect.width,
    cssHeight: rect.height,
    dpr: 1,
  })
  const logical = { width: size.cssWidth, height: size.cssHeight }
  layoutDebug('read', { cssW: logical.width, cssH: logical.height })
  return logical
}

/**
 * Prefer an already-known logical size (e.g. from usePracticeCanvas), else read the element.
 * Logs `[canvas.layout]` DEBUG on submit/resync callers when `reason` is set.
 */
export function resolveSubmitLogicalSize(opts: {
  fromComposable?: LogicalSize | null
  canvas?: HTMLCanvasElement | null
  reason?: string
}): LogicalSize {
  const dpr =
    typeof window !== 'undefined' && Number.isFinite(window.devicePixelRatio)
      ? window.devicePixelRatio || 1
      : 1

  if (opts.fromComposable && opts.fromComposable.width > 0 && opts.fromComposable.height > 0) {
    const logical = logicalSizeForRecognize(opts.fromComposable)
    if (opts.reason) {
      layoutDebug(opts.reason, { cssW: logical.width, cssH: logical.height, dpr })
    }
    return logical
  }

  const fromEl = readLogicalCanvasSize(opts.canvas)
  if (fromEl) {
    if (opts.reason) {
      layoutDebug(opts.reason, { cssW: fromEl.width, cssH: fromEl.height, dpr })
    }
    return fromEl
  }

  // Last-resort fallback when layout is unavailable (jsdom / detached).
  const fallback = { width: 300, height: 300 }
  if (opts.reason) {
    layoutDebug(opts.reason, { cssW: fallback.width, cssH: fallback.height, dpr, fallback: true })
  }
  return fallback
}

/** CSS custom property name for the shared square canvas token. */
export const CANVAS_SIZE_TOKEN = '--canvas-size'
