import { expect, Page, test } from '@playwright/test'
import testId from '../../../src/testId'

const openDevice = async (page: Page) => {
    await page.goto('')
    await page.request.get('http://localhost:8181/api/v1/devices/api-reset')
    await page.setViewportSize({ width: 1600, height: 800 })

    await page.getByTestId(testId.menu.devices).click()

    await expect(page.getByTestId('device-row-0')).toBeVisible()
    await page.getByTestId('device-row-0').click()
}

test('device-detail-tab-1', async ({ page }) => {
    await openDevice(page)

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(testId.devices.detail.tabInformation)).toBeVisible()
    await expect(page.getByTestId(testId.devices.detail.tabResources)).toBeVisible()
    await expect(page.getByTestId(testId.devices.detail.tabCertificates)).toBeVisible()
    await expect(page.getByTestId(testId.devices.detail.tabProvisioningRecords)).toBeVisible()

    await expect(page.getByTestId(testId.devices.detail.tabInformation)).toHaveClass(/active/)
})

test('device-detail-tab-1-toggles', async ({ page }) => {
    await openDevice(page)

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.devices.detail.information.twinToggle}`)).toBeVisible()
    await expect(page.getByTestId(`${testId.devices.detail.information.notificationsToggle}`)).toBeVisible()

    await expect(page.getByTestId(`${testId.devices.detail.information.twinToggle}-switch`)).toBeChecked()
    await expect(page.getByTestId(`${testId.devices.detail.information.notificationsToggle}-switch`)).not.toBeChecked()

    // change state
    await page.getByTestId(`${testId.devices.detail.information.twinToggle}-switch-label`).click()
    await page.getByTestId(`${testId.devices.detail.information.notificationsToggle}-switch-label`).click()

    await expect(page.getByTestId(`${testId.devices.detail.information.twinToggle}-switch`)).not.toBeChecked()
    await expect(page.getByTestId(`${testId.devices.detail.information.notificationsToggle}-switch`)).toBeChecked()
})

test('device-detail-tab-1-table', async ({ page }) => {
    await openDevice(page)

    await expect(page.getByTestId(`${testId.devices.detail.information.types}-modal-btn`)).not.toBeVisible()
    await expect(page.getByTestId(`${testId.devices.detail.information.endpoints}-modal-btn`)).toBeVisible()

    // more tags modal open
    await page.getByTestId(`${testId.devices.detail.information.endpoints}-modal-btn`).click()
    await expect(page.getByTestId(`${testId.devices.detail.information.endpoints}-modal`)).toBeVisible()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await page.getByTestId(`${testId.devices.detail.information.endpoints}-modal-close`).click()
    await expect(page.getByTestId(`${testId.devices.detail.information.endpoints}-modal`)).not.toBeVisible()
})

test('devices-detail-edit-name', async ({ page }) => {
    await openDevice(page)

    await expect(page.getByTestId(testId.devices.detail.editNameButton)).toBeVisible()

    await page.getByTestId(testId.devices.detail.editNameButton).click()

    await expect(page.getByTestId(testId.devices.detail.editNameModal)).toBeVisible()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.devices.detail.editNameModal}-input`)).toBeVisible()

    const name = 'New Device Name'

    await page.getByTestId(`${testId.devices.detail.editNameModal}-input`).fill(name)

    await expect(page.getByTestId(`${testId.devices.detail.editNameModal}-button-confirm`)).toBeVisible()

    await page.getByTestId(`${testId.devices.detail.editNameModal}-button-confirm`).click()

    await expect(page.getByTestId(testId.devices.detail.editNameModal)).not.toBeVisible()

    const newName = await page.getByTestId(`${testId.devices.detail.layout}-title`).textContent()

    expect(newName).toBe(name)
})

test('devices-detail-delete-device-close', async ({ page }) => {
    await openDevice(page)

    await expect(page.getByTestId(testId.devices.detail.deleteDeviceButton)).toBeVisible()

    await page.getByTestId(testId.devices.detail.deleteDeviceButton).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

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

    await expect(page).toHaveURL(/devices/)
})

test('device-detail-tab-2', async ({ page }) => {
    await openDevice(page)

    await expect(page.getByTestId(testId.devices.detail.tabResources)).toBeVisible()
    await page.getByTestId(testId.devices.detail.tabResources).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-0`)).toBeVisible()
    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-1`)).toBeVisible()
    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-4`)).toBeVisible()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(testId.devices.detail.tabResources)).toHaveClass(/active/)
})

test('device-detail-tab-2-table-filter', async ({ page }) => {
    await openDevice(page)

    await page.getByTestId(testId.devices.detail.tabResources).click()
    await expect(page.getByTestId(testId.devices.detail.resources.table)).toBeVisible()
    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-filter-input`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-filter-input`).fill('/light')

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-4`)).toBeVisible()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})

test('device-detail-tab-2-table-update-modal-open-close', async ({ page }) => {
    await openDevice(page)

    await page.getByTestId(testId.devices.detail.tabResources).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-4-href`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-4-href`).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-modal`)).toBeVisible()
    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-retrieve-button`)).toBeVisible()

    await page.getByTestId(`${testId.devices.detail.resources.updateModal}-retrieve-button`).click()
    await page.getByTestId(`${testId.devices.detail.resources.updateModal}-modal-close`).click()
    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-modal`)).not.toBeVisible()
})

test('device-detail-tab-2-table-update-modal-full-view', async ({ page }) => {
    await openDevice(page)

    await page.getByTestId(testId.devices.detail.tabResources).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-4-href`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-4-href`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-modal`)).toBeVisible()
    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-editor`)).toBeVisible()
    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-editor-view-button`)).toBeVisible()

    await page.getByTestId(`${testId.devices.detail.resources.updateModal}-editor-view-button`).click()
    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-modal`)).toHaveClass(/fullSize/)

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await page.getByTestId(`${testId.devices.detail.resources.updateModal}-modal-close`).click()
    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-modal`)).not.toBeVisible()
})

test('device-detail-tab-2-table-open-update-modal-update', async ({ page }) => {
    await openDevice(page)

    await page.getByTestId(testId.devices.detail.tabResources).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-4-href`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-4-href`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-modal`)).toBeVisible()

    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-editor`)).toBeVisible()

    await page.getByTestId(`${testId.devices.detail.resources.updateModal}-editor-input`).fill(
        JSON.stringify({
            state: false,
            power: 1,
            name: 'Light',
        })
    )

    await page.getByTestId(`${testId.devices.detail.resources.updateModal}-confirm-button`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-modal`)).not.toBeVisible()
})

test('device-detail-tab-2-table-update-modal-open-action-button', async ({ page }) => {
    await openDevice(page)

    await page.setViewportSize({ width: 1600, height: 800 })

    await page.getByTestId(testId.devices.detail.tabResources).click()
    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-4-action-update`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-4-action-update`).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})

test('device-detail-tab-2-table-update-modal-open-action-button-toggle-open', async ({ page }) => {
    await openDevice(page)

    await page.getByTestId(testId.devices.detail.tabResources).click()

    await page.setViewportSize({ width: 1200, height: 800 })

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-4-actions-toggle`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-4-actions-toggle`).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})

test('device-detail-tab-2-table-update-modal-open-action-button-toggle', async ({ page }) => {
    await openDevice(page)

    await page.getByTestId(testId.devices.detail.tabResources).click()

    await page.setViewportSize({ width: 1200, height: 800 })

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-4-actions-toggle`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-4-actions-toggle`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-4-action-update`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-4-action-update`).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})

const openDeviceWotResource = async (page: Page) => {
    await page.goto('/')
    await page.getByTestId(testId.menu.devices).click()
    await page.getByTestId('device-row-3').click()

    await page.getByTestId(testId.devices.detail.tabResources).click()
}

test('device-detail-tab-2-wot-resource-form', async ({ page }) => {
    await openDeviceWotResource(page)

    await page.setViewportSize({ width: 1600, height: 1600 })
    await expect(page.getByTestId(testId.devices.detail.tabCertificates)).not.toHaveClass(/disabled/)
    await expect(page.getByTestId(testId.devices.detail.tabProvisioningRecords)).not.toHaveClass(/disabled/)

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-1-href`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-1-href`).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})

test('device-detail-tab-2-wot-resource-simple-form-update', async ({ page }) => {
    await openDeviceWotResource(page)

    await page.setViewportSize({ width: 1600, height: 1600 })

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-7-href`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-7-href`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-generated-form-form-/color`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.updateModal}-generated-form-form-/color`).fill('#000000')

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-confirm-button`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.updateModal}-confirm-button`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}`)).not.toBeVisible()

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-7-href`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-7-href`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-generated-form-form-/color`)).toHaveValue(/#000000/)
})

test('device-detail-tab-2-view-switch', async ({ page }) => {
    await openDevice(page)

    await page.getByTestId(testId.devices.detail.tabResources).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.viewSwitch}-label`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.viewSwitch}-label`).click()
    await expect(page.getByTestId(`${testId.devices.detail.resources.viewSwitch}`)).toBeChecked()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})

test('device-detail-tab-2-tree-open-update-modal-href', async ({ page }) => {
    await openDevice(page)

    await page.getByTestId(testId.devices.detail.tabResources).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.viewSwitch}-label`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.viewSwitch}-label`).click()
    await expect(page.getByTestId(`${testId.devices.detail.resources.viewSwitch}`)).toBeChecked()

    await expect(page.getByTestId(`${testId.devices.detail.resources.tree}-row-0-expander-expander`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.tree}-row-0-expander-expander`).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.devices.detail.resources.tree}-row-/light/1/`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.tree}-row-/light/1/`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-modal`)).toBeVisible()
})

test('device-detail-tab-2-tree-open-update-modal-toggle', async ({ page }) => {
    await openDevice(page)

    await page.getByTestId(testId.devices.detail.tabResources).click()

    await page.setViewportSize({ width: 1200, height: 800 })

    await expect(page.getByTestId(`${testId.devices.detail.resources.viewSwitch}-label`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.viewSwitch}-label`).click()
    await expect(page.getByTestId(`${testId.devices.detail.resources.viewSwitch}`)).toBeChecked()

    await expect(page.getByTestId(`${testId.devices.detail.resources.tree}-row-0-expander-expander`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.tree}-row-0-expander-expander`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-0.0-actions-toggle`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-0.0-actions-toggle`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-0.0-action-update`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-0.0-action-update`).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-modal`)).toBeVisible()
})

test('device-detail-tab-2-tree-open-update-modal-toggle-delete', async ({ page }) => {
    await openDevice(page)

    await page.getByTestId(testId.devices.detail.tabResources).click()

    await page.setViewportSize({ width: 1200, height: 800 })

    await expect(page.getByTestId(`${testId.devices.detail.resources.viewSwitch}-label`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.viewSwitch}-label`).click()
    await expect(page.getByTestId(`${testId.devices.detail.resources.viewSwitch}`)).toBeChecked()

    await expect(page.getByTestId(`${testId.devices.detail.resources.tree}-row-0-expander-expander`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.tree}-row-0-expander-expander`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-0.0-actions-toggle`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-0.0-actions-toggle`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-0.0-action-delete`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-0.0-action-delete`).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.devices.detail.resources.deleteModal}`)).toBeVisible()
})

test('device-detail-tab-2-tree-open-update-modal-edit-icon', async ({ page }) => {
    await openDevice(page)

    await page.setViewportSize({ width: 1600, height: 1600 })

    await page.getByTestId(testId.devices.detail.tabResources).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.viewSwitch}-label`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.viewSwitch}-label`).click()
    await expect(page.getByTestId(`${testId.devices.detail.resources.viewSwitch}`)).toBeChecked()

    await expect(page.getByTestId(`${testId.devices.detail.resources.tree}-row-0-expander-expander`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.tree}-row-0-expander-expander`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-0.0-action-update`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-0.0-action-update`).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.devices.detail.resources.updateModal}-modal`)).toBeVisible()
})

test('device-detail-tab-2-tree-open-update-modal-delete-icon', async ({ page }) => {
    await openDevice(page)

    await page.setViewportSize({ width: 1600, height: 1600 })

    await page.getByTestId(testId.devices.detail.tabResources).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.viewSwitch}-label`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.viewSwitch}-label`).click()
    await expect(page.getByTestId(`${testId.devices.detail.resources.viewSwitch}`)).toBeChecked()

    await expect(page.getByTestId(`${testId.devices.detail.resources.tree}-row-0-expander-expander`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.tree}-row-0-expander-expander`).click()

    await expect(page.getByTestId(`${testId.devices.detail.resources.table}-row-1-action-delete`)).toBeVisible()
    await page.getByTestId(`${testId.devices.detail.resources.table}-row-1-action-delete`).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.devices.detail.resources.deleteModal}`)).toBeVisible()
})

test('device-detail-tab-3', async ({ page }) => {
    await openDevice(page)

    await page.setViewportSize({ width: 1600, height: 1600 })

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})

test('devices-detail-rest', async ({ page, request }) => {
    const resetRequest = await request.get(`/`)
    expect(resetRequest.ok()).toBeTruthy()
})
