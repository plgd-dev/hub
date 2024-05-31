import { test, expect } from '@playwright/test'
import testId from '../../../src/testId'

const urlBase = 'http://localhost:3000'

test('devices-list-open', async ({ page }) => {
    await page.goto(urlBase)
    await page.getByTestId(testId.menu.devices).click()
    await expect(page).toHaveTitle(/Devices | plgd Dashboard/)
    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true })
})

test('devices-list-open-detail', async ({ page }) => {
    await page.goto(urlBase)
    await page.getByTestId(testId.menu.devices).click()
    await expect(page).toHaveTitle(/Devices | plgd Dashboard/)
    await expect(page.getByTestId('device-row-0')).toBeVisible()
    const deviceName = await page.getByTestId('device-row-0').textContent()
    await page.getByTestId('device-row-0').click()
    await expect(page).toHaveTitle(`${deviceName} | plgd Dashboard`)
})
