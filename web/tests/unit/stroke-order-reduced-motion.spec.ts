import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, nextTick } from 'vue'
import StrokeOrderPlayer from '../../src/components/practice/StrokeOrderPlayer.vue'
import { stubCanvasContext } from '../helpers/stubCanvasContext'

describe('StrokeOrderPlayer reduced motion', () => {
  let canvasSpy: ReturnType<typeof stubCanvasContext>
  let matchMediaSpy: { mockRestore: () => void } | undefined

  beforeEach(() => {
    canvasSpy = stubCanvasContext()
    document.body.innerHTML = ''
  })

  afterEach(() => {
    canvasSpy.mockRestore()
    matchMediaSpy?.mockRestore()
  })

  it('skips animation to final frame when prefers-reduced-motion', async () => {
    matchMediaSpy = vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => {
      return {
        matches: query.includes('prefers-reduced-motion'),
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
      } as MediaQueryList
    }) as unknown as { mockRestore: () => void }

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
    await new Promise((r) => setTimeout(r, 30))
    expect(done).toHaveBeenCalled()
    expect(root.textContent).toMatch(/Motion reduced|動きを減ら/)
    wrapper.unmount()
  })
})
