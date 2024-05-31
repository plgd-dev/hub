import { test, expect } from '@playwright/test'

test('login action', async ({ page }) => {
    await page.goto('http://localhost:3000/')

    // keycloak login page
    await expect(page).toHaveTitle(/Login | plgd.dev/)

    // await expect(page).toHaveURL(/auth.plgd.cloud/)
    //
    // await page.locator('#email').fill(process.env.REACT_APP_TEST_LOGIN_USERNAME || '')
    // await page.locator('#password').fill(process.env.REACT_APP_TEST_LOGIN_PASSWORD || '')
    //
    // await page.getByRole('button', { name: 'Sign In' }).click()
})
