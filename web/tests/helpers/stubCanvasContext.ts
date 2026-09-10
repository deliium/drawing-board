import { vi } from 'vitest'

export type StubCanvasContext = {
  clearRect: ReturnType<typeof vi.fn>
  beginPath: ReturnType<typeof vi.fn>
  moveTo: ReturnType<typeof vi.fn>
  lineTo: ReturnType<typeof vi.fn>
  stroke: ReturnType<typeof vi.fn>
  fill: ReturnType<typeof vi.fn>
  fillText: ReturnType<typeof vi.fn>
  arc: ReturnType<typeof vi.fn>
  save: ReturnType<typeof vi.fn>
  restore: ReturnType<typeof vi.fn>
  setTransform: ReturnType<typeof vi.fn>
  strokeStyle: string
  fillStyle: string
  lineWidth: number
  lineCap: string
  lineJoin: string
  font: string
  textAlign: string
  textBaseline: string
}

/** Minimal 2d context so jsdom overlay/canvas mounts do not emit not-implemented noise. */
export function stubCanvasContext(): {
  mockRestore: () => void
  ctx: StubCanvasContext
} {
  const ctx: StubCanvasContext = {
    clearRect: vi.fn(),
    beginPath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    stroke: vi.fn(),
    fill: vi.fn(),
    fillText: vi.fn(),
    arc: vi.fn(),
    save: vi.fn(),
    restore: vi.fn(),
    setTransform: vi.fn(),
    strokeStyle: '',
    fillStyle: '',
    lineWidth: 0,
    lineCap: '',
    lineJoin: '',
    font: '',
    textAlign: '',
    textBaseline: '',
  }
  // [FIX] mute jsdom HTMLCanvasElement.getContext for practice overlay mounts
  console.debug('[FIX] stubCanvasContext: mocking getContext for jsdom')
  const spy = vi
    .spyOn(HTMLCanvasElement.prototype, 'getContext')
    .mockReturnValue(ctx as unknown as CanvasRenderingContext2D)
  return {
    mockRestore: () => spy.mockRestore(),
    ctx,
  }
}
