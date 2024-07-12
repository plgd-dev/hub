import { chromium, expect, FullConfig } from '@playwright/test'
import { login } from './tests/utils'

async function globalSetup(config: FullConfig) {
    const wellKnowConfigurationAddress = process.env.REACT_APP_HTTP_WELL_KNOW_CONFIGURATION_ADDRESS
    const browser = await chromium.launch()
    const page = await browser.newPage()

    const response = await page.request.get(`${wellKnowConfigurationAddress}/.well-known/configuration`)
    await expect(response).toBeOK()

    const data = await response.json()

    process.env.WELL_KNOWN_CONFIG = JSON.stringify(data)

    await login(page)

    await page.context().storageState({ path: 'storageState.json' })
    await browser.close()
}

export default globalSetup
