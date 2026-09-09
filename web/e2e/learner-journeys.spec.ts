import { expect, test } from '@playwright/test'
import { assessCharacterViaAPI, registerViaUI, uniqueEmail } from './helpers'

test.describe('learner journeys', () => {
  test('auth + practice assess shows Match (not confidence)', async ({ page, context, request }) => {
    const email = uniqueEmail('practice')
    await registerViaUI(page, email)

    await page.goto('/#/practice/hira:%E3%81%82')
    await expect(page.getByText('あ').first()).toBeVisible({ timeout: 20_000 })
    await expect(page.getByRole('button', { name: /^(Start|始める)$/ })).toBeVisible()

    const cookies = await context.cookies()
    const result = await assessCharacterViaAPI(request, cookies, 'hira:あ', 'あ')
    expect(result.pass).toBe(true)
    expect(result.score).toBeGreaterThan(0.5)

    await page.goto('/#/practice/history')
    await expect(page.getByRole('heading', { name: /Attempt history|履歴/i })).toBeVisible({
      timeout: 20_000,
    })
    await expect(page.getByText(/Match/i).first()).toBeVisible()
    await expect(page.locator('body')).not.toContainText(/confidence/i)
  })

  test('history and hub after assessment stay metadata-only', async ({ page, context, request }) => {
    const email = uniqueEmail('history')
    await registerViaUI(page, email)

    const cookies = await context.cookies()
    await assessCharacterViaAPI(request, cookies, 'hira:い', 'い')

    await page.goto('/#/practice/history')
    await expect(page.getByRole('heading', { name: /Attempt history|履歴/i })).toBeVisible({
      timeout: 20_000,
    })
    const body = await page.locator('body').innerText()
    expect(body).toMatch(/Match/i)
    expect(body).not.toMatch(/"points"\s*:/)
    expect(body).not.toMatch(/x:\s*0\.\d+,\s*y:/)

    await page.goto('/#/practice')
    await expect(page.getByText(/あ|い|う|え|お/).first()).toBeVisible({ timeout: 20_000 })
    await expect(page.locator('body')).not.toContainText(/confidence/i)
  })
})
