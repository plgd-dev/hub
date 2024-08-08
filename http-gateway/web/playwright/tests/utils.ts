import { expect, Page } from '@playwright/test'

export const login = async (page: Page) => {
    await page.goto('http://localhost:3000/')

    // keycloak
    await expect(page).toHaveTitle(/Login | plgd.dev/, { timeout: 30000 })
    await expect(page).toHaveURL(/auth.plgd.cloud/)

    await page.locator('#username').fill(process.env.REACT_APP_TEST_LOGIN_USERNAME || '')
    await page.locator('#password').fill(process.env.REACT_APP_TEST_LOGIN_PASSWORD || '')
    await page.getByRole('button', { name: 'Sign In' }).click()
}
