import { test, expect, Page } from '@playwright/test'
import testId from '../../../../src/testId'
import { JTW_TOKEN } from '../../constants'
import { addAndCheckFilter, openConditionFilter, removeAndCheck } from '../../utils'

const openConditionsList = async (page: Page) => {
    await page.goto('')
    await page.getByTestId(testId.menu.snippetService.link).click()
    await page.getByTestId(testId.menu.snippetService.conditions).click()
    await page.setViewportSize({ width: 1600, height: 800 })
}

test('snippet-service-configurations-list-open', async ({ page }) => {
    await openConditionsList(page)

    await expect(page).toHaveTitle(/Conditions | plgd Dashboard/)
    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})

test('snippet-service-configurations-list-link-to-detail-name', async ({ page }) => {
    await openConditionsList(page)

    await expect(page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0-name`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0-name`).click()

    await expect(page).toHaveTitle(/jkralik-cond-0 | plgd Dashboard/)
})

test('snippet-service-configurations-list-link-to-detail-icon', async ({ page }) => {
    await openConditionsList(page)

    await expect(page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0-detail`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0-detail`).click()

    await expect(page).toHaveTitle(/jkralik-cond-0 | plgd Dashboard/)
})

test('snippet-service-configurations-list-add-open-close', async ({ page }) => {
    await openConditionsList(page)

    await expect(page.getByTestId(testId.snippetService.conditions.list.addButton)).toBeVisible()
    await page.getByTestId(testId.snippetService.conditions.list.addButton).click()

    await expect(page).toHaveTitle(/Add Condition | plgd Dashboard/)

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.wizard)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.wizard}-close`)).toBeVisible()

    await page.getByTestId(`${testId.snippetService.conditions.addPage.wizard}-close`).click()
    await expect(page).toHaveTitle(/Conditions | plgd Dashboard/)
})

test('snippet-service-configurations-list-add', async ({ page }) => {
    await openConditionsList(page)

    await expect(page.getByTestId(testId.snippetService.conditions.list.addButton)).toBeVisible()
    await page.getByTestId(testId.snippetService.conditions.list.addButton).click()

    await expect(page).toHaveTitle(/Add Condition | plgd Dashboard/)

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await page.getByTestId(testId.snippetService.conditions.addPage.step1.form.name).fill('cond-02')

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step1.buttons}-continue`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step1.buttons}-continue`)).not.toBeDisabled()

    await page.getByTestId(`${testId.snippetService.conditions.addPage.step1.buttons}-continue`).click()

    // ******** STEP2
    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page).toHaveURL(/\/snippet-service\/conditions\/add\/apply-filters/)
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-back`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-back`)).not.toBeDisabled()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-continue`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-continue`)).toBeDisabled()

    // --- device Id selector
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

    // remove selected device
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-remove`).click()

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-attribute`)).not.toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-value`)).not.toBeVisible()

    // select device back
    await page.locator('#deviceIdFilter').click()
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.selectDeviceId}-input`).fill('3aae0672-47f3-4498-78d4-b061e6105ccd')
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.selectDeviceId}-3aae0672-47f3-4498-78d4-b061e6105ccd`).click()

    await page.getByTestId(testId.snippetService.conditions.addPage.step2.selectDeviceIdDone).click()

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-attribute`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-value`)).toBeVisible()

    // --- resourceType selector
    await openConditionFilter(page, testId.snippetService.conditions.addPage.step2.resourceType)

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.resourceType}-input`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.resourceType}-addButton`)).toBeVisible()

    await addAndCheckFilter(page, testId.snippetService.conditions.addPage.step2.resourceType)
    await removeAndCheck(page, testId.snippetService.conditions.addPage.step2.resourceType)
    await addAndCheckFilter(page, testId.snippetService.conditions.addPage.step2.resourceType)

    // --- hrefFilter selector
    await openConditionFilter(page, testId.snippetService.conditions.addPage.step2.hrefFilter)

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.hrefFilter}-input`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.hrefFilter}-addButton`)).toBeVisible()

    await addAndCheckFilter(page, testId.snippetService.conditions.addPage.step2.hrefFilter)
    await removeAndCheck(page, testId.snippetService.conditions.addPage.step2.hrefFilter)
    await addAndCheckFilter(page, testId.snippetService.conditions.addPage.step2.hrefFilter)

    // --- jqExpression filter
    await openConditionFilter(page, testId.snippetService.conditions.addPage.step2.jqExpressionFilter)

    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.jqExpressionFilter}-input`).fill('.n == "new name value')

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-back`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-back`)).not.toBeDisabled()

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-continue`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-continue`)).not.toBeDisabled()

    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-continue`).click()

    // ******** STEP3

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
    await expect(page).toHaveURL(/\/snippet-service\/conditions\/add\/select-configuration/)

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step3.selectConfiguration)).toBeVisible()
    await page.locator('#configurationId').click()
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step3.selectConfiguration}-48998f7d-2a70-46a4-8a68-745b69d55489`).click()

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step3.apiToken)).toBeVisible()
    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step3.generateApiToken)).toBeVisible()

    await page.getByTestId(testId.snippetService.conditions.addPage.step3.generateApiToken).click()

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step3.generateApiTokenModal)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step3.generateApiTokenModal}-invoke`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step3.generateApiTokenModal}-invoke`).click()

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step3.generateApiTokenModal)).not.toBeVisible()

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step3.apiToken)).toHaveValue(JTW_TOKEN)

    // buttons
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step3.buttons}-back`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step3.buttons}-back`)).not.toBeDisabled()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step3.buttons}-continue`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step3.buttons}-continue`)).not.toBeDisabled()

    await page.getByTestId(`${testId.snippetService.conditions.addPage.step3.buttons}-continue`).click()

    await expect(page).toHaveURL(/\/snippet-service\/conditions/)
    await expect(page).toHaveTitle(/Conditions | plgd Dashboard/)
})

test('snippet-service-configurations-list-delete', async ({ page }) => {
    await openConditionsList(page)

    await expect(page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0-delete`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0-delete`).click()

    await expect(page.getByTestId(`${testId.snippetService.conditions.list.pageTemplate}-delete-modal`)).toBeVisible()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.snippetService.conditions.list.pageTemplate}-delete-modal-cancel`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.list.pageTemplate}-delete-modal-delete`)).toBeVisible()

    await page.getByTestId(`${testId.snippetService.conditions.list.pageTemplate}-delete-modal-delete`).click()

    await expect(page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0-delete`)).not.toBeVisible()
})
