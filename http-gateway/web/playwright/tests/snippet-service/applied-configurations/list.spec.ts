import { expect, test } from '@playwright/test'
import testId from '../../../../src/testId'

test('snippet-service-applied-configurations-list-open', async ({ page }) => {
    await page.goto('')
    await page.getByTestId(testId.menu.snippetService.link).click()
    await page.getByTestId(testId.menu.snippetService.appliedConfigurations).click()

    await expect(page).toHaveTitle(/Applied Configurations | plgd Dashboard/)
    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})
