import { test, expect } from '@playwright/test'
import testId from '../../src/testId'

test('logout action', async ({ page }) => {
    await page.goto('http://localhost:3000/')

    await page.getByTestId(testId.app.logout).click()

    await page.getByTestId(testId.app.logoutBtn).click()

    await expect(page).toHaveTitle(/Login | plgd.dev/)
})
