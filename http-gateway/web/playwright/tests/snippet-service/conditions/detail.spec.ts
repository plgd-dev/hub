import { expect, Page, test } from '@playwright/test'
import testId from '../../../../src/testId'
import { addAndCheckFilter, openConditionFilter, removeAndCheck } from '../../utils'

const openConditionItem = async (page: Page) => {
    await page.goto('')
    await page.request.get('http://localhost:8181/snippet-service/api/v1/conditions/api-reset')

    await page.getByTestId(testId.menu.snippetService.link).click()
    await page.getByTestId(testId.menu.snippetService.conditions).click()
    await page.setViewportSize({ width: 1600, height: 800 })
    await page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0-detail`).click()
}

test('snippet-service-conditions-detail-version', async ({ page }) => {
    await openConditionItem(page)

    await expect(page).toHaveTitle(/jkralik-cond-0 | plgd Dashboard/)
    await expect(page.getByTestId(`${testId.snippetService.conditions.detail.versionSelector}`)).toBeVisible()

    await page.locator('#version').click()
    await expect(page.getByTestId(`${testId.snippetService.conditions.detail.versionSelector}-select-input`)).toBeVisible()

    await page.getByTestId(`${testId.snippetService.conditions.detail.versionSelector}-select-0`).click()

    await expect(page).toHaveTitle(/jkralik-cond-0 | plgd Dashboard/)
})

test('snippet-service-conditions-detail-delete', async ({ page }) => {
    await openConditionItem(page)

    await expect(page.getByTestId(testId.snippetService.conditions.detail.deleteButton)).toBeVisible()
    await page.getByTestId(testId.snippetService.conditions.detail.deleteButton).click()

    await expect(page.getByTestId(testId.snippetService.conditions.detail.deleteModal)).toBeVisible()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(testId.snippetService.conditions.detail.deleteButtonCancel)).toBeVisible()
    await expect(page.getByTestId(testId.snippetService.conditions.detail.deleteButtonConfirm)).toBeVisible()

    await page.getByTestId(testId.snippetService.conditions.detail.deleteButtonConfirm).click()

    await expect(page.getByTestId(testId.snippetService.conditions.detail.deleteModal)).not.toBeVisible()

    await expect(page).toHaveTitle(/Conditions | plgd Dashboard/)

    await expect(page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0`)).not.toBeVisible()
})

test('snippet-service-conditions-detail-tab-1', async ({ page }) => {
    await openConditionItem(page)

    await expect(page).toHaveTitle(/jkralik-cond-0 | plgd Dashboard/)

    await expect(page.getByTestId(testId.snippetService.conditions.detail.tab1.form.name)).toHaveValue('jkralik-cond-0')
    await expect(page.getByTestId(testId.snippetService.conditions.detail.bottomPanel)).not.toBeVisible()

    await page.getByTestId(testId.snippetService.conditions.detail.tab1.form.name).fill('jkralik-cond-01')
    await expect(page.getByTestId(testId.snippetService.conditions.detail.bottomPanel)).toBeVisible()

    await expect(page.getByTestId(testId.snippetService.conditions.detail.bottomPanelReset)).toBeVisible()
    await expect(page.getByTestId(testId.snippetService.conditions.detail.bottomPanelSave)).toBeVisible()
    await expect(page.getByTestId(testId.snippetService.conditions.detail.bottomPanelSave)).not.toBeDisabled()

    await page.getByTestId(testId.snippetService.conditions.detail.bottomPanelReset).click()
    await expect(page.getByTestId(testId.snippetService.conditions.detail.tab1.form.name)).toHaveValue('jkralik-cond-0')
    await expect(page.getByTestId(testId.snippetService.conditions.detail.bottomPanel)).not.toBeVisible()

    await page.getByTestId(testId.snippetService.conditions.detail.tab1.form.name).fill('jkralik-cond-01')
    await page.getByTestId(testId.snippetService.conditions.detail.bottomPanelSave).click()

    await expect(page).toHaveTitle(/Conditions | plgd Dashboard/)
})

test('snippet-service-conditions-detail-tab-2', async ({ page }) => {
    await openConditionItem(page)

    await page.getByTestId(testId.snippetService.conditions.detail.tabFilters).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await openConditionFilter(page, testId.snippetService.conditions.addPage.step2.filterDeviceId)
    await page.locator('#deviceIdFilter').focus()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.selectDeviceId}-input`)).toBeVisible()

    // select device
    await page.locator('#deviceIdFilter').click()
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.selectDeviceId}-input`).fill('3aae0672-47f3-4498-78d4-b061e6105ccd')
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.selectDeviceId}-3aae0672-47f3-4498-78d4-b061e6105ccd`).click()

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step2.selectDeviceIdReset)).toBeVisible()
    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step2.selectDeviceIdDone)).toBeVisible()

    await page.getByTestId(testId.snippetService.conditions.addPage.step2.selectDeviceIdDone).click()

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-attribute`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-value`)).toBeVisible()

    // --- hrefFilter selector
    await openConditionFilter(page, testId.snippetService.conditions.addPage.step2.resourceType)

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.resourceType}-input`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.resourceType}-addButton`)).toBeVisible()

    await addAndCheckFilter(page, testId.snippetService.conditions.addPage.step2.resourceType)
    await removeAndCheck(page, testId.snippetService.conditions.addPage.step2.resourceType)
    await addAndCheckFilter(page, testId.snippetService.conditions.addPage.step2.resourceType)

    await expect(page.getByTestId(testId.snippetService.conditions.detail.bottomPanel)).toBeVisible()

    await expect(page.getByTestId(testId.snippetService.conditions.detail.bottomPanelReset)).toBeVisible()
    await expect(page.getByTestId(testId.snippetService.conditions.detail.bottomPanelSave)).toBeVisible()
    await expect(page.getByTestId(testId.snippetService.conditions.detail.bottomPanelSave)).not.toBeDisabled()

    await page.getByTestId(testId.snippetService.conditions.detail.bottomPanelSave).click()

    await expect(page).toHaveTitle(/Conditions | plgd Dashboard/)
})

test('snippet-service-conditions-detail-tab-3', async ({ page }) => {
    await openConditionItem(page)

    await page.getByTestId(testId.snippetService.conditions.detail.tabApiAccessToken).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})
