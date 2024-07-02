import { test, expect } from '@playwright/test'
import testId from '../../../../src/testId'

const urlBase = 'http://localhost:3000'

test('snippet-service-configurations-list-open', async ({ page }) => {
    await page.goto(urlBase)
    await page.getByTestId(testId.menu.snippetService.link).click()
    await page.getByTestId(testId.menu.snippetService.configurations).click()

    await expect(page).toHaveTitle(/Configuraions | plgd Dashboard/)
    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true })
})
