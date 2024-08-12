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

export const openConditionFilter = async (page: Page, locator: string) => {
    await expect(page.getByTestId(locator)).toBeVisible()
    await page.getByTestId(`${locator}-header`).click()
    await expect(page.getByTestId(`${locator}-content`)).toBeVisible()
}

export const addAndCheckFilter = async (page: Page, locator: string) => {
    await page.getByTestId(`${locator}-input`).fill('/oic/p')
    await page.getByTestId(`${locator}-addButton`).click()

    await expect(page.getByTestId(`${locator}-content-table-row-0-attribute`)).toBeVisible()
    await expect(page.getByTestId(`${locator}-content-table-row-0-value`)).toBeVisible()
}

export const removeAndCheck = async (page: Page, locator: string) => {
    await page.getByTestId(`${locator}-content-table-row-0-remove`).click()

    await expect(page.getByTestId(`${locator}-content-table-row-0-attribute`)).not.toBeVisible()
    await expect(page.getByTestId(`${locator}-content-table-row-0-value`)).not.toBeVisible()
}
