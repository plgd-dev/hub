import { expect, test } from '@playwright/test'
import testId from '../../../src/testId'

const urlBase = 'http://localhost:3000'

test('devices-detail-edit-name', async ({ page }) => {
    await page.goto(urlBase)

    await page.getByTestId(testId.menu.devices).click()

    await page.getByTestId('device-row-0').click()

    await expect(page.getByTestId(testId.devices.detail.editNameButton)).toBeVisible()

    await page.getByTestId(testId.devices.detail.editNameButton).click()

    await expect(page.getByTestId(testId.devices.detail.editNameModal)).toBeVisible()

    await expect(page.getByTestId(`${testId.devices.detail.editNameModal}-input`)).toBeVisible()

    const name = 'tmp-device-name'

    page.getByTestId(`${testId.devices.detail.editNameModal}-input`).fill('tmp-device-name')

    await expect(page.getByTestId(`${testId.devices.detail.editNameModal}-button-confirm`)).toBeVisible()

    await page.getByTestId(`${testId.devices.detail.editNameModal}-button-confirm`).click()

    await expect(page.getByTestId(testId.devices.detail.editNameModal)).not.toBeVisible()

    const newName = await page.getByTestId(`${testId.devices.detail.layout}-title`).textContent()

    expect(newName).toBe(name)
})
