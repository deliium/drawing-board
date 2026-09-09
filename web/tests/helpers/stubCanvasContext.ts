import { vi } from 'vitest'

/** Minimal 2d context so jsdom overlay/canvas mounts do not emit not-implemented noise. */
export function stubCanvasContext(): { mockRestore: () => void } {
  const ctx = {
    clearRect: vi.fn(),
    beginPath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    stroke: vi.fn(),
    fill: vi.fn(),
    arc: vi.fn(),
    save: vi.fn(),
    restore: vi.fn(),
    setTransform: vi.fn(),
    strokeStyle: '',
    fillStyle: '',
    lineWidth: 0,
    lineCap: '',
    lineJoin: '',
  }
  // [FIX] mute jsdom HTMLCanvasElement.getContext for practice overlay mounts
  console.debug('[FIX] stubCanvasContext: mocking getContext for jsdom')
  return vi
    .spyOn(HTMLCanvasElement.prototype, 'getContext')
    .mockReturnValue(ctx as unknown as CanvasRenderingContext2D)
}
