import { expect, test } from '@playwright/test'
import testId from '../../../src/testId'

const urlBase = 'http://localhost:3000'

const openDevice = async (page) => {
    await page.goto(urlBase)
    await page.getByTestId(testId.menu.devices).click()
    await page.getByTestId('device-row-0').click()
}

test('devices-detail-edit-name', async ({ page }) => {
    await openDevice(page)

    await expect(page.getByTestId(testId.devices.detail.editNameButton)).toBeVisible()

    await page.getByTestId(testId.devices.detail.editNameButton).click()

    await expect(page.getByTestId(testId.devices.detail.editNameModal)).toBeVisible()

    await expect(page.getByTestId(`${testId.devices.detail.editNameModal}-input`)).toBeVisible()

    const name = 'tmp-device-name'

    await page.getByTestId(`${testId.devices.detail.editNameModal}-input`).fill(name)

    await expect(page.getByTestId(`${testId.devices.detail.editNameModal}-button-confirm`)).toBeVisible()

    await page.getByTestId(`${testId.devices.detail.editNameModal}-button-confirm`).click()

    await expect(page.getByTestId(testId.devices.detail.editNameModal)).not.toBeVisible()

    const newName = await page.getByTestId(`${testId.devices.detail.layout}-title`).textContent()

    expect(newName).toBe(name)
})

test('devices-detail-edit-name-reset-close', async ({ page }) => {
    await openDevice(page)

    await expect(page.getByTestId(testId.devices.detail.editNameButton)).toBeVisible()

    await page.getByTestId(testId.devices.detail.editNameButton).click()

    await expect(page.getByTestId(testId.devices.detail.editNameModal)).toBeVisible()

    await expect(page.getByTestId(`${testId.devices.detail.editNameModal}-input`)).toBeVisible()

    await expect(page.getByTestId(`${testId.devices.detail.editNameModal}-button-reset`)).toBeVisible()

    const originName = await page.getByTestId(`${testId.devices.detail.editNameModal}-input`).textContent()

    await page.getByTestId(`${testId.devices.detail.editNameModal}-input`).fill('random-device-name')

    await page.getByTestId(`${testId.devices.detail.editNameModal}-button-reset`).click()

    const currentVal = await page.getByTestId(`${testId.devices.detail.editNameModal}-input`).textContent()

    expect(currentVal).toBe(originName)
})

test('devices-detail-delete-device-close', async ({ page }) => {
    await openDevice(page)

    await expect(page.getByTestId(testId.devices.detail.deleteDeviceButton)).toBeVisible()

    await page.getByTestId(testId.devices.detail.deleteDeviceButton).click()

    await expect(page.getByTestId(testId.devices.detail.deleteDeviceModal)).toBeVisible()

    await expect(page.getByTestId(testId.devices.detail.deleteDeviceButtonCancel)).toBeVisible()

    await page.getByTestId(testId.devices.detail.deleteDeviceButtonCancel).click()

    await expect(page.getByTestId(testId.devices.detail.deleteDeviceModal)).not.toBeVisible()
})

test('devices-detail-delete-device', async ({ page }) => {
    await openDevice(page)

    await expect(page.getByTestId(testId.devices.detail.deleteDeviceButton)).toBeVisible()

    await page.getByTestId(testId.devices.detail.deleteDeviceButton).click()

    await expect(page.getByTestId(testId.devices.detail.deleteDeviceModal)).toBeVisible()

    await expect(page.getByTestId(testId.devices.detail.deleteDeviceButtonCancel)).toBeVisible()

    await page.getByTestId(testId.devices.detail.deleteDeviceButtonDelete).click()

    await expect(page.getByTestId(testId.devices.detail.deleteDeviceModal)).not.toBeVisible()

    await expect(page).toHaveURL(urlBase)
})
