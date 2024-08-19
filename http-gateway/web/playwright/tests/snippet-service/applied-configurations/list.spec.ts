import { Browser, expect, Page, test } from '@playwright/test'
import testId from '../../../../src/testId'

const openAppliedConfigurationsList = async (page: Page, browser: Browser) => {
    await page.goto('', { waitUntil: 'networkidle' })
    await page.getByTestId(testId.menu.snippetService.link).click()

    //  wait for submenu to be visible
    if (browser.browserType().name() === 'webkit') {
        await page.waitForTimeout(500)
    }

    await page.getByTestId(testId.menu.snippetService.appliedConfigurations).click()

    await page.request.get('http://localhost:8181/snippet-service/api/v1/applied-configurations/api-reset')

    await expect(page).toHaveURL(/snippet-service\/applied-configurations/)
    await page.setViewportSize({ width: 1600, height: 800 })
}

test('snippet-service-applied-configurations-list-open', async ({ page, browser }) => {
    await openAppliedConfigurationsList(page, browser)

    await expect(page).toHaveTitle(/Applied Configurations | plgd Dashboard/)
    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})

test('snippet-service-applied-configurations-list-open-detail-name', async ({ page, browser }) => {
    await openAppliedConfigurationsList(page, browser)

    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0-name`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0-name`).click()

    await expect(page).toHaveTitle(/dps-endpoint-is-set| plgd Dashboard/)
})

test('snippet-service-applied-configurations-list-open-detail-link', async ({ page, browser }) => {
    await openAppliedConfigurationsList(page, browser)

    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0-detail`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0-detail`).click()

    await expect(page).toHaveTitle(/dps-endpoint-is-set| plgd Dashboard/)
})

test('snippet-service-applied-configurations-list-delete', async ({ page, browser }) => {
    await openAppliedConfigurationsList(page, browser)

    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0-delete`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0-delete`).click()

    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.pageTemplate}-delete-modal`)).toBeVisible()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.pageTemplate}-delete-modal-cancel`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.pageTemplate}-delete-modal-delete`)).toBeVisible()

    await page.getByTestId(`${testId.snippetService.appliedConfigurations.list.pageTemplate}-delete-modal-delete`).click()

    await expect(page).toHaveTitle(/Applied Configurations | plgd Dashboard/)
    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0`)).not.toBeVisible()
})

test('snippet-service-applied-configurations-list-link', async ({ page, browser }) => {
    await openAppliedConfigurationsList(page, browser)

    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0-condition`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.appliedConfigurations.list.table}-row-0-condition`).click()

    await expect(page).toHaveTitle(/jkralik-cond-0 | plgd Dashboard/)
})
