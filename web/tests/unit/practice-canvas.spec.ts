import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { effectScope, nextTick, ref, type EffectScope, type Ref } from 'vue'
import { hitTest } from '../../src/canvas/hitTest'
import {
  usePracticeCanvas,
  type PracticeTool,
  type PracticeCanvasSyncStatus,
} from '../../src/composables/usePracticeCanvas'
import type { Point, Stroke } from '../../src/services/strokeSync'

const webRoot = join(dirname(fileURLToPath(import.meta.url)), '../..')

type Harness = {
  scope: EffectScope
  canvas: HTMLCanvasElement
  strokes: Ref<Stroke[]>
  tool: Ref<PracticeTool>
  color: Ref<string>
  width: Ref<number>
  syncStatus: Ref<PracticeCanvasSyncStatus>
  enabled: Ref<boolean>
  onStrokeComplete: ReturnType<typeof vi.fn>
  onEraseAt: ReturnType<typeof vi.fn>
  api: ReturnType<typeof usePracticeCanvas>
  dispose: () => void
  paintCalls: { clearRect: ReturnType<typeof vi.fn>; arc: ReturnType<typeof vi.fn> }
}

function stubCanvas(cssW = 300, cssH = 300, dpr = 1): {
  canvas: HTMLCanvasElement
  paintCalls: Harness['paintCalls']
} {
  const canvas = document.createElement('canvas')
  const clearRect = vi.fn()
  const arc = vi.fn()
  const ctx = {
    clearRect,
    beginPath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    stroke: vi.fn(),
    fill: vi.fn(),
    arc,
    setTransform: vi.fn(),
    strokeStyle: '',
    fillStyle: '',
    lineWidth: 0,
    lineCap: '',
    lineJoin: '',
  }
  vi.spyOn(canvas, 'getContext').mockReturnValue(ctx as unknown as CanvasRenderingContext2D)
  vi.spyOn(canvas, 'getBoundingClientRect').mockReturnValue({
    width: cssW,
    height: cssH,
    left: 10,
    top: 20,
    right: 10 + cssW,
    bottom: 20 + cssH,
    x: 10,
    y: 20,
    toJSON: () => ({}),
  })
  canvas.setPointerCapture = vi.fn()
  canvas.releasePointerCapture = vi.fn()
  canvas.hasPointerCapture = vi.fn(() => true)
  Object.defineProperty(window, 'devicePixelRatio', {
    configurable: true,
    get: () => dpr,
  })
  document.body.appendChild(canvas)
  return { canvas, paintCalls: { clearRect, arc } }
}

function mountPractice(opts?: {
  cssW?: number
  cssH?: number
  dpr?: number
  enabled?: boolean
  tool?: PracticeTool
}): Harness {
  const { canvas, paintCalls } = stubCanvas(opts?.cssW, opts?.cssH, opts?.dpr ?? 1)
  const canvasRef = ref<HTMLCanvasElement | null>(canvas)
  const strokes = ref<Stroke[]>([])
  const tool = ref<PracticeTool>(opts?.tool ?? 'pencil')
  const color = ref('#1d4ed8')
  const width = ref(4)
  const syncStatus = ref<PracticeCanvasSyncStatus>('saved')
  const enabled = ref(opts?.enabled ?? true)
  const onStrokeComplete = vi.fn()
  const onEraseAt = vi.fn()

  const scope = effectScope()
  let api!: ReturnType<typeof usePracticeCanvas>
  scope.run(() => {
    api = usePracticeCanvas({
      canvasRef,
      enabled,
      strokes,
      tool,
      color,
      width,
      syncStatus,
      onStrokeComplete,
      onEraseAt,
    })
  })

  return {
    scope,
    canvas,
    strokes,
    tool,
    color,
    width,
    syncStatus,
    enabled,
    onStrokeComplete,
    onEraseAt,
    api,
    paintCalls,
    dispose: () => {
      scope.stop()
      canvas.remove()
    },
  }
}

function dispatchPointer(
  canvas: HTMLCanvasElement,
  type: string,
  clientX: number,
  clientY: number,
  pointerId = 1,
) {
  const event = new Event(type, { bubbles: true, cancelable: true }) as PointerEvent
  Object.assign(event, {
    clientX,
    clientY,
    pointerId,
    pointerType: 'mouse',
    button: 0,
    buttons: type === 'pointerup' || type === 'pointercancel' ? 0 : 1,
  })
  canvas.dispatchEvent(event)
}

describe('usePracticeCanvas', () => {
  let harness: Harness | null = null

  afterEach(() => {
    harness?.dispose()
    harness = null
    vi.restoreAllMocks()
  })

  beforeEach(() => {
    vi.stubGlobal(
      'ResizeObserver',
      class {
        observe() {}
        disconnect() {}
        unobserve() {}
      },
    )
  })

  it('attaches once and detaches listeners so a second attach does not double-fire', async () => {
    harness = mountPractice()
    const { canvas, onStrokeComplete, enabled, dispose } = harness

    dispatchPointer(canvas, 'pointerdown', 60, 70)
    dispatchPointer(canvas, 'pointerup', 60, 70)
    expect(onStrokeComplete).toHaveBeenCalledTimes(1)

    enabled.value = false
    await nextTick()
    enabled.value = true
    await nextTick()

    onStrokeComplete.mockClear()
    dispatchPointer(canvas, 'pointerdown', 60, 70)
    dispatchPointer(canvas, 'pointerup', 60, 70)
    expect(onStrokeComplete).toHaveBeenCalledTimes(1)

    dispose()
    onStrokeComplete.mockClear()
    dispatchPointer(canvas, 'pointerdown', 60, 70)
    dispatchPointer(canvas, 'pointerup', 60, 70)
    expect(onStrokeComplete).not.toHaveBeenCalled()
    harness = null
  })

  it('resizes backing store for DPR and redraws existing strokes', async () => {
    harness = mountPractice({ dpr: 2 })
    const { canvas, strokes, api, paintCalls } = harness
    expect(canvas.width).toBe(600)
    expect(canvas.height).toBe(600)

    strokes.value = [
      {
        points: [{ x: 40, y: 50 }],
        color: '#111',
        width: 4,
        clientId: 'c',
        startedAtUnixMs: 1,
      },
    ]
    await nextTick()
    expect(paintCalls.arc).toHaveBeenCalled()

    paintCalls.clearRect.mockClear()
    vi.spyOn(canvas, 'getBoundingClientRect').mockReturnValue({
      width: 200,
      height: 200,
      left: 10,
      top: 20,
      right: 210,
      bottom: 220,
      x: 10,
      y: 20,
      toJSON: () => ({}),
    })
    api.resync()
    expect(canvas.width).toBe(400)
    expect(canvas.height).toBe(400)
    expect(paintCalls.clearRect).toHaveBeenCalledWith(0, 0, 200, 200)
  })

  it('maps client coords to CSS points; recognize logical ≠ backing when dpr=2', () => {
    harness = mountPractice({ dpr: 2 })
    const { canvas, onStrokeComplete, api } = harness

    dispatchPointer(canvas, 'pointerdown', 10 + 100, 20 + 80)
    dispatchPointer(canvas, 'pointerup', 10 + 100, 20 + 80)

    expect(onStrokeComplete).toHaveBeenCalledWith([{ x: 100, y: 80 }])
    expect(canvas.width).toBe(600)
    expect(api.getLogicalSize()).toEqual({ width: 300, height: 300 })
  })

  it('commits one-point taps as strokes', () => {
    harness = mountPractice()
    const { canvas, onStrokeComplete } = harness
    dispatchPointer(canvas, 'pointerdown', 50, 60)
    dispatchPointer(canvas, 'pointerup', 50, 60)
    expect(onStrokeComplete).toHaveBeenCalledTimes(1)
    const points = onStrokeComplete.mock.calls[0][0] as Point[]
    expect(points).toHaveLength(1)
  })

  it('Escape and pointercancel do not call onStrokeComplete', () => {
    harness = mountPractice()
    const { canvas, onStrokeComplete } = harness

    dispatchPointer(canvas, 'pointerdown', 50, 60)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(onStrokeComplete).not.toHaveBeenCalled()

    dispatchPointer(canvas, 'pointerdown', 50, 60)
    dispatchPointer(canvas, 'pointercancel', 50, 60)
    expect(onStrokeComplete).not.toHaveBeenCalled()
  })

  it('commits multi-point pencil strokes', () => {
    harness = mountPractice()
    const { canvas, onStrokeComplete } = harness
    dispatchPointer(canvas, 'pointerdown', 20, 30)
    dispatchPointer(canvas, 'pointermove', 40, 50)
    dispatchPointer(canvas, 'pointermove', 60, 70)
    dispatchPointer(canvas, 'pointerup', 60, 70)
    const points = onStrokeComplete.mock.calls[0][0] as Point[]
    expect(points.length).toBeGreaterThanOrEqual(3)
  })

  it('eraser calls onEraseAt; hit-test succeeds for dots', () => {
    harness = mountPractice({ tool: 'eraser' })
    const { canvas, onEraseAt } = harness
    dispatchPointer(canvas, 'pointerdown', 10 + 100, 20 + 100)
    expect(onEraseAt).toHaveBeenCalledWith({ x: 100, y: 100 })
    expect(hitTest({ x: 100, y: 100 }, { points: [{ x: 100, y: 100 }], width: 4 })).toBe(true)
  })
})

describe('practice canvas erase ownership', () => {
  it('keeps hitTest on BoardPage and out of usePracticeCanvas imports', () => {
    const composableSrc = readFileSync(
      join(webRoot, 'src/composables/usePracticeCanvas.ts'),
      'utf8',
    )
    const boardSrc = readFileSync(join(webRoot, 'src/pages/BoardPage.vue'), 'utf8')

    expect(composableSrc).not.toMatch(/from ['"]\.\.\/canvas\/hitTest['"]/)
    expect(composableSrc).toMatch(/onEraseAt/)
    expect(boardSrc).toMatch(/from ['"]\.\.\/canvas\/hitTest['"]/)
    expect(boardSrc).toMatch(/onEraseAt/)
    expect(boardSrc).toMatch(/hitTest\(/)
  })
})
