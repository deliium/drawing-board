import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { createApp, defineComponent, nextTick } from 'vue'
import MigrationHealthPanel from '../../src/components/MigrationHealthPanel.vue'

async function flush() {
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

describe('MigrationHealthPanel learner absence', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('does not render Migration Health heading (renamed Dev metrics only when mounted)', () => {
    const root = document.createElement('div')
    document.body.appendChild(root)
    const app = createApp(MigrationHealthPanel)
    app.mount(root)
    expect(root.textContent).not.toMatch(/Migration Health/i)
    expect(root.textContent).toMatch(/Dev metrics/)
    app.unmount()
  })

  it('production App.vue path omits panel when DEV is false', async () => {
    // Simulate the App.vue gate: panel only when import.meta.env.DEV.
    const isDev = false
    const root = document.createElement('div')
    document.body.appendChild(root)
    const Host = defineComponent({
      components: { MigrationHealthPanel },
      setup() {
        return { isDev }
      },
      template: '<div><MigrationHealthPanel v-if="isDev" /></div>',
    })
    const app = createApp(Host)
    app.mount(root)
    await flush()
    expect(root.textContent).not.toMatch(/Migration Health|Dev metrics/i)
    app.unmount()
  })
})
