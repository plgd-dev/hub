import { chromium, firefox, expect, FullConfig } from '@playwright/test'
import { login } from './tests/utils'

async function globalSetup(config: FullConfig) {
    const wellKnowConfigurationAddress = process.env.REACT_APP_HTTP_WELL_KNOW_CONFIGURATION_ADDRESS

    const getBrowser = (browserName: string) => {
        if (browserName === 'chromium') {
            return chromium
        } else if (browserName === 'firefox') {
            return firefox
        } else if (browserName === 'webkit') {
            return firefox
        } else {
            throw new Error(`Unknown browser: ${browserName}`)
        }
    }

    for (const browserName of config.projects.map((x) => x.name)) {
        console.log(`Preparing browser: ${browserName}`)
        const start = Date.now()
        const browser = await getBrowser(browserName).launch()
        const context = await browser.newContext()

        const page = await context.newPage()

        const response = await page.request.get(`${wellKnowConfigurationAddress}/.well-known/configuration`)
        await expect(response).toBeOK()

        const data = await response.json()

        process.env.WELL_KNOWN_CONFIG = JSON.stringify(data)

        await login(page)

        const r = await page.request.get('https://api.github.com/repos/plgd-dev/hub/releases/latest')
        const body = await r.json()

        const versionData = {
            requestedDatetime: new Date(),
            latest: body.tag_name.replace('v', ''),
            latest_url: body.html_url,
        }

        await page.context().storageState({ path: `storageState.${browserName}.json` })

        const storage = await page.context().storageState()

        const root = JSON.parse(storage.origins[0].localStorage.find((x) => x.name === 'persist:root')?.value || '{}')
        const rootApp = { ...JSON.parse(root['app'] || '{}'), version: versionData }

        const newRoot = JSON.stringify({ ...root, app: JSON.stringify(rootApp) })

        await page.waitForLoadState('networkidle')
        await page.evaluate((newRoot) => localStorage.setItem('persist:root', JSON.stringify(newRoot)), newRoot)

        await page.context().storageState({ path: `storageState.${browserName}.json` })

        const end = Date.now()
        console.log(`Done. Execution time: ${end - start} ms`)
        await browser.close()
    }
}

export default globalSetup
