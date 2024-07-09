import { test, expect, Page } from '@playwright/test'
import testId from '../../../../src/testId'

const urlBase = 'http://localhost:3000'

const openConfigurationItem = async (page: Page) => {
    await page.goto(urlBase)
    await page.getByTestId(testId.menu.snippetService.link).click()
    await page.getByTestId(testId.menu.snippetService.configurations).click()
    await page.getByTestId(`${testId.snippetService.configurations.list.table}-row-0`).click()
}

test('snippet-service-configurations-detail-version', async ({ page }) => {
    await openConfigurationItem(page)

    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.versionSelector}`)).toBeVisible()
})
