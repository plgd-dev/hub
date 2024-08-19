import { Browser, expect, Page, test } from '@playwright/test'
import testId from '../../../../src/testId'

const openAppliedConfigurationItem = async (page: Page, browser: Browser) => {
    await page.goto('', { waitUntil: 'networkidle' })
    await page.request.get('http://localhost:8181/snippet-service/api/v1/applied-configurations/api-reset')

    await page.getByTestId(testId.menu.snippetService.link).click()

    //  wait for submenu to be visible
    if (browser.browserType().name() === 'webkit') {
        await page.waitForTimeout(1000)
    }

    await page.getByTestId(testId.menu.snippetService.appliedConfigurations).click()

    await page.setViewportSize({ width: 1600, height: 800 })
    await page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0-detail`).click()
}

test('snippet-service-conditions-detail', async ({ page, browser }) => {
    await openAppliedConfigurationItem(page, browser)

    await expect(page).toHaveTitle(/dps-endpoint-is-set | plgd Dashboard/)

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})

test('snippet-service-conditions-detail-delete', async ({ page, browser }) => {
    await openAppliedConfigurationItem(page, browser)

    await expect(page.getByTestId(testId.snippetService.appliedConfigurations.detail.deleteButton)).toBeVisible()
    await page.getByTestId(testId.snippetService.appliedConfigurations.detail.deleteButton).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(testId.snippetService.appliedConfigurations.detail.deleteModal)).toBeVisible()

    await expect(page.getByTestId(testId.snippetService.appliedConfigurations.detail.deleteButtonCancel)).toBeVisible()
    await expect(page.getByTestId(testId.snippetService.appliedConfigurations.detail.deleteButtonConfirm)).toBeVisible()

    await page.getByTestId(testId.snippetService.appliedConfigurations.detail.deleteButtonConfirm).click()

    await expect(page).toHaveTitle(/Applied Configurations | plgd Dashboard/)
    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0`)).not.toBeVisible()
})

test('snippet-service-conditions-detail-tab-1-configuration-link', async ({ page, browser }) => {
    await openAppliedConfigurationItem(page, browser)

    await expect(page.getByTestId(testId.snippetService.appliedConfigurations.detail.tab1.configurationLink)).toBeVisible()
    await page.getByTestId(testId.snippetService.appliedConfigurations.detail.tab1.configurationLink).click()

    await expect(page).toHaveTitle(/my-cfg-1 | plgd Dashboard/)
})

test('snippet-service-conditions-detail-tab-1-condition-link', async ({ page, browser }) => {
    await openAppliedConfigurationItem(page, browser)

    await expect(page.getByTestId(testId.snippetService.appliedConfigurations.detail.tab1.conditionLink)).toBeVisible()
    await page.getByTestId(testId.snippetService.appliedConfigurations.detail.tab1.conditionLink).click()

    await expect(page).toHaveTitle(/jkralik-cond-0 | plgd Dashboard/)
})

test('snippet-service-conditions-detail-tab-2', async ({ page, browser }) => {
    await openAppliedConfigurationItem(page, browser)

    await page.getByTestId(testId.snippetService.appliedConfigurations.detail.tabListOfResources).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.detail.tab2.resourceToggleCreator}-1-view-button`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.appliedConfigurations.detail.tab2.resourceToggleCreator}-1-view-button`).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})
