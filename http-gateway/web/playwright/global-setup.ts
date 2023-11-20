import { chromium, expect, FullConfig } from '@playwright/test'

async function globalSetup(config: FullConfig) {
    const wellKnowConfigurationAddress = process.env.REACT_APP_HTTP_WELL_KNOW_CONFIGURATION_ADDRESS
    const browser = await chromium.launch()
    const page = await browser.newPage()

    const response = await page.request.get(`${wellKnowConfigurationAddress}/.well-known/configuration`)
    await expect(response).toBeOK()

    const data = await response.json()

    process.env.WELL_KNOWN_CONFIG = JSON.stringify(data)

    await page.goto('http://localhost:3000/')

    // login data
    await page.locator('#email').fill(process.env.REACT_APP_TEST_LOGIN_USERNAME || '')
    await page.locator('#password').fill(process.env.REACT_APP_TEST_LOGIN_PASSWORD || '')
    await page.getByRole('button', { name: 'Sign In' }).click()

    await expect(page).toHaveTitle(/Devices | plgd Dashboard/)

    await page.context().storageState({ path: 'storageState.json' })
    await browser.close()
}

export default globalSetup
