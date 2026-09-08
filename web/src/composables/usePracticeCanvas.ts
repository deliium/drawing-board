import {
  onScopeDispose,
  watch,
  type ComputedRef,
  type Ref,
} from 'vue'
import {
  applyCanvasBackingStore,
  cssPointFromClient,
  logicalSizeForRecognize,
  resolveCanvasBackingSize,
  type CanvasBackingSize,
} from '../canvas/coords'
import { drawStrokes } from '../canvas/drawStrokes'
import type { Point, Stroke } from '../services/strokeSync'

export type PracticeTool = 'pencil' | 'eraser'

export type PracticeCanvasSyncStatus =
  | 'connecting'
  | 'saving'
  | 'saved'
  | 'offline'
  | 'error'

export type UsePracticeCanvasOptions = {
  canvasRef: Ref<HTMLCanvasElement | null>
  enabled: Ref<boolean> | ComputedRef<boolean>
  strokes: Ref<Stroke[]>
  tool: Ref<PracticeTool>
  color: Ref<string>
  width: Ref<number>
  syncStatus: Ref<PracticeCanvasSyncStatus>
  onStrokeComplete: (points: Point[]) => void
  onEraseAt: (point: Point) => void
  onCancelInProgress?: () => void
}

const practiceDebug =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function canvasDebug(...args: unknown[]) {
  if (practiceDebug) console.debug('[practiceCanvas]', ...args)
}

function canvasWarn(...args: unknown[]) {
  if (practiceDebug) console.warn('[practiceCanvas]', ...args)
}

/**
 * Owns practice-canvas pointer lifecycle, DPR backing store, live preview,
 * resize redraw, and Escape / pointercancel abort.
 */
export function usePracticeCanvas(options: UsePracticeCanvasOptions) {
  let attachGen = 0
  let attached: {
    gen: number
    canvas: HTMLCanvasElement
    cleanup: () => void
  } | null = null

  let drawing = false
  let activePointerId: number | null = null
  let inProgress: Point[] = []
  let lastSize: CanvasBackingSize | null = null

  function getCtx(canvas: HTMLCanvasElement): CanvasRenderingContext2D | null {
    const ctx = canvas.getContext('2d', { willReadFrequently: true })
    if (!ctx) {
      canvasWarn('missing 2d context')
      return null
    }
    return ctx
  }

  function applyTransform(ctx: CanvasRenderingContext2D, dpr: number) {
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    ctx.lineCap = 'round'
    ctx.lineJoin = 'round'
  }

  function paintAll(canvas: HTMLCanvasElement, size: CanvasBackingSize, preview?: Point[]) {
    const ctx = getCtx(canvas)
    if (!ctx) return
    applyTransform(ctx, size.dpr)
    const layers: Stroke[] = [...options.strokes.value]
    if (preview && preview.length) {
      layers.push({
        points: preview,
        color: options.color.value,
        width: options.width.value,
        clientId: '',
        startedAtUnixMs: 0,
      })
    }
    drawStrokes(ctx, layers, {
      cssWidth: size.cssWidth,
      cssHeight: size.cssHeight,
    })
  }

  function cancelInProgress(reason: string, canvas?: HTMLCanvasElement | null) {
    if (!drawing && inProgress.length === 0) return
    const points = inProgress.length
    drawing = false
    inProgress = []
    const pid = activePointerId
    activePointerId = null
    if (canvas && pid !== null) {
      try {
        if (canvas.hasPointerCapture?.(pid)) {
          canvas.releasePointerCapture(pid)
        }
      } catch {
        canvasWarn('capture release failed', 'pointerId=', pid)
      }
    }
    canvasDebug('stroke_cancel', 'reason=', reason, 'points=', points, 'pointerId=', pid)
    options.onCancelInProgress?.()
    if (canvas && lastSize) {
      paintAll(canvas, lastSize)
    }
  }

  function syncSize(canvas: HTMLCanvasElement, reason: string): CanvasBackingSize | null {
    const rect = canvas.getBoundingClientRect()
    const dpr =
      typeof window !== 'undefined' && Number.isFinite(window.devicePixelRatio)
        ? window.devicePixelRatio || 1
        : 1
    const size = resolveCanvasBackingSize({
      cssWidth: rect.width,
      cssHeight: rect.height,
      dpr,
    })
    if (size.clamped) {
      canvasDebug('resize clamp', 'css=', `${size.cssWidth}x${size.cssHeight}`, 'dpr=', size.dpr)
    }
    if (drawing || inProgress.length) {
      cancelInProgress('resize', canvas)
    }
    const wiped = applyCanvasBackingStore(canvas, size)
    lastSize = size
    canvasDebug(
      'resize',
      'reason=',
      reason,
      'css=',
      `${size.cssWidth}x${size.cssHeight}`,
      'dpr=',
      size.dpr,
      'backing=',
      `${size.backingWidth}x${size.backingHeight}`,
      'wiped=',
      wiped,
    )
    paintAll(canvas, size)
    return size
  }

  function detach(reason: string) {
    if (!attached) return
    const { canvas, cleanup, gen } = attached
    attached = null
    cancelInProgress('detach', canvas)
    cleanup()
    canvasDebug('detach', 'reason=', reason, 'gen=', gen)
  }

  function attach(canvas: HTMLCanvasElement) {
    if (attached?.canvas === canvas) {
      canvasDebug('duplicate_attach_prevented', 'gen=', attached.gen)
      return
    }
    detach('reattach')
    const ctx = getCtx(canvas)
    if (!ctx) return

    const gen = ++attachGen
    canvasDebug('attach', 'gen=', gen)

    const size = syncSize(canvas, 'attach')
    if (!size) return

    const toPoint = (e: PointerEvent): Point => {
      const rect = canvas.getBoundingClientRect()
      return cssPointFromClient(e.clientX, e.clientY, rect)
    }

    const onDown = (e: PointerEvent) => {
      if (options.syncStatus.value === 'error') return
      const p = toPoint(e)
      if (options.tool.value === 'eraser') {
        options.onEraseAt(p)
        return
      }
      drawing = true
      activePointerId = e.pointerId
      inProgress = [p]
      canvasDebug('stroke_start', 'pointerId=', e.pointerId, 'points=', 1)
      try {
        canvas.setPointerCapture(e.pointerId)
      } catch {
        canvasWarn('capture failed', 'pointerId=', e.pointerId)
      }
      if (lastSize) paintAll(canvas, lastSize, inProgress)
    }

    const onMove = (e: PointerEvent) => {
      if (!drawing || options.tool.value !== 'pencil') return
      if (activePointerId !== null && e.pointerId !== activePointerId) return
      inProgress.push(toPoint(e))
      if (lastSize) paintAll(canvas, lastSize, inProgress)
    }

    const onUp = (e: PointerEvent) => {
      if (!drawing || options.tool.value !== 'pencil') {
        drawing = false
        inProgress = []
        activePointerId = null
        return
      }
      if (activePointerId !== null && e.pointerId !== activePointerId) return
      const points = [...inProgress]
      drawing = false
      inProgress = []
      const pid = activePointerId
      activePointerId = null
      try {
        if (pid !== null && canvas.hasPointerCapture?.(pid)) {
          canvas.releasePointerCapture(pid)
        }
      } catch {
        canvasWarn('capture release failed', 'pointerId=', pid)
      }
      if (points.length >= 1) {
        canvasDebug('stroke_commit', 'pointerId=', pid, 'points=', points.length)
        options.onStrokeComplete(points)
      }
      if (lastSize) paintAll(canvas, lastSize)
    }

    const onCancel = (e: PointerEvent) => {
      if (activePointerId !== null && e.pointerId !== activePointerId) return
      cancelInProgress('pointercancel', canvas)
    }

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (!drawing && !inProgress.length) return
        e.preventDefault()
        cancelInProgress('user_escape', canvas)
      }
    }

    const onWindowResize = () => {
      syncSize(canvas, 'window')
    }

    let ro: ResizeObserver | null = null
    if (typeof ResizeObserver !== 'undefined') {
      ro = new ResizeObserver(() => {
        if (attached?.gen !== gen) return
        syncSize(canvas, 'observer')
      })
      ro.observe(canvas)
    }

    canvas.addEventListener('pointerdown', onDown)
    canvas.addEventListener('pointermove', onMove)
    canvas.addEventListener('pointerup', onUp)
    canvas.addEventListener('pointercancel', onCancel)
    window.addEventListener('keydown', onKeyDown)
    window.addEventListener('resize', onWindowResize)

    const cleanup = () => {
      canvas.removeEventListener('pointerdown', onDown)
      canvas.removeEventListener('pointermove', onMove)
      canvas.removeEventListener('pointerup', onUp)
      canvas.removeEventListener('pointercancel', onCancel)
      window.removeEventListener('keydown', onKeyDown)
      window.removeEventListener('resize', onWindowResize)
      ro?.disconnect()
    }

    attached = { gen, canvas, cleanup }
  }

  const stopEnabledWatch = watch(
    [options.canvasRef, options.enabled],
    ([canvas, enabled]) => {
      if (!enabled || !canvas) {
        detach(enabled ? 'no_canvas' : 'disabled')
        return
      }
      attach(canvas)
    },
    { immediate: true },
  )

  const stopStrokesWatch = watch(
    options.strokes,
    () => {
      if (!attached || !lastSize) return
      if (drawing && inProgress.length) {
        paintAll(attached.canvas, lastSize, inProgress)
      } else {
        paintAll(attached.canvas, lastSize)
      }
    },
    { deep: true },
  )

  onScopeDispose(() => {
    stopEnabledWatch()
    stopStrokesWatch()
    detach('scope_dispose')
  })

  return {
    /** Logical CSS size for recognize body (matches stored stroke coords). */
    getLogicalSize(): { width: number; height: number } | null {
      if (lastSize) {
        return logicalSizeForRecognize(lastSize)
      }
      const canvas = options.canvasRef.value
      if (!canvas) return null
      return logicalSizeForRecognize(canvas)
    },
    /** Force resize + redraw (tests / external layout changes). */
    resync() {
      const canvas = attached?.canvas ?? options.canvasRef.value
      if (!canvas) return
      syncSize(canvas, 'manual')
    },
  }
}
