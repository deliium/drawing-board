import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, nextTick } from 'vue'
import StrokeOrderPlayer from '../../src/components/practice/StrokeOrderPlayer.vue'
import { initLocale, setLocale } from '../../src/i18n'
import { stubCanvasContext } from '../helpers/stubCanvasContext'

function mockMatchMedia(matchesReducedMotion: boolean) {
  const impl = (query: string): MediaQueryList =>
    ({
      matches: matchesReducedMotion && query.includes('prefers-reduced-motion'),
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }) as MediaQueryList

  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn(impl),
  })
}

function stubLayoutBox() {
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    x: 0,
    y: 0,
    top: 0,
    left: 0,
    bottom: 300,
    right: 300,
    width: 300,
    height: 300,
    toJSON: () => ({}),
  } as DOMRect)
}

describe('StrokeOrderPlayer', () => {
  let canvasStub: ReturnType<typeof stubCanvasContext>
  let roCallback: ResizeObserverCallback | null = null

  beforeEach(() => {
    initLocale()
    setLocale('en')
    canvasStub = stubCanvasContext()
    stubLayoutBox()
    document.body.innerHTML = ''
    roCallback = null
    class MockRO {
      constructor(cb: ResizeObserverCallback) {
        roCallback = cb
      }
      observe() {}
      unobserve() {}
      disconnect() {}
    }
    vi.stubGlobal('ResizeObserver', MockRO)
  })

  afterEach(() => {
    canvasStub.mockRestore()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('skips animation to final frame when prefers-reduced-motion', async () => {
    mockMatchMedia(true)

    const done = vi.fn()
    const root = document.createElement('div')
    document.body.appendChild(root)
    const wrapper = createApp({
      components: { StrokeOrderPlayer },
      template: '<StrokeOrderPlayer glyph="あ" @done="onDone" />',
      methods: { onDone: done },
    })
    wrapper.mount(root)
    await nextTick()
    await new Promise((r) => setTimeout(r, 40))
    expect(done).toHaveBeenCalled()
    expect(root.textContent).toMatch(/Motion reduced|動きを減ら/)
    expect(canvasStub.ctx.stroke).toHaveBeenCalled()
    expect(canvasStub.ctx.fillText).toHaveBeenCalled()
    const labels = canvasStub.ctx.fillText.mock.calls.map((c) => c[0])
    expect(labels).toEqual(expect.arrayContaining(['1', '2', '3']))
    wrapper.unmount()
  })

  it('paints strokes and order indices for a known glyph', async () => {
    mockMatchMedia(true)
    const done = vi.fn()
    const root = document.createElement('div')
    document.body.appendChild(root)
    const wrapper = createApp({
      components: { StrokeOrderPlayer },
      template: '<StrokeOrderPlayer glyph="い" @done="onDone" />',
      methods: { onDone: done },
    })
    wrapper.mount(root)
    await nextTick()
    await new Promise((r) => setTimeout(r, 40))
    expect(done).toHaveBeenCalled()
    expect(canvasStub.ctx.beginPath).toHaveBeenCalled()
    expect(canvasStub.ctx.stroke).toHaveBeenCalled()
    expect(canvasStub.ctx.fillText).toHaveBeenCalledWith('1', expect.any(Number), expect.any(Number))
    expect(canvasStub.ctx.fillText).toHaveBeenCalledWith('2', expect.any(Number), expect.any(Number))
    wrapper.unmount()
  })

  it('shows missing hint and [FIX] log when glyph has no trace', async () => {
    mockMatchMedia(true)
    const info = vi.spyOn(console, 'info').mockImplementation(() => {})
    const done = vi.fn()
    const root = document.createElement('div')
    document.body.appendChild(root)
    const wrapper = createApp({
      components: { StrokeOrderPlayer },
      template: '<StrokeOrderPlayer glyph="ん" @done="onDone" />',
      methods: { onDone: done },
    })
    wrapper.mount(root)
    await nextTick()
    await new Promise((r) => setTimeout(r, 40))
    expect(done).toHaveBeenCalled()
    expect(root.textContent).toMatch(/Stroke order for 「ん」 is unavailable/)
    expect(info.mock.calls.some((c) => String(c[0]).includes('[FIX]') && String(c[1]).includes('[strokeOrder]'))).toBe(
      true,
    )
    wrapper.unmount()
    info.mockRestore()
  })

  it('repaints current frame on ResizeObserver callback', async () => {
    mockMatchMedia(true)
    const done = vi.fn()
    const root = document.createElement('div')
    document.body.appendChild(root)
    const wrapper = createApp({
      components: { StrokeOrderPlayer },
      template: '<StrokeOrderPlayer glyph="あ" @done="onDone" />',
      methods: { onDone: done },
    })
    wrapper.mount(root)
    await nextTick()
    await new Promise((r) => setTimeout(r, 40))
    expect(done).toHaveBeenCalled()
    const strokesBefore = canvasStub.ctx.stroke.mock.calls.length
    expect(roCallback).toBeTypeOf('function')
    roCallback?.(
      [{ contentRect: { width: 400, height: 400 } } as unknown as ResizeObserverEntry],
      {} as ResizeObserver,
    )
    expect(canvasStub.ctx.stroke.mock.calls.length).toBeGreaterThan(strokesBefore)
    wrapper.unmount()
  })
})
