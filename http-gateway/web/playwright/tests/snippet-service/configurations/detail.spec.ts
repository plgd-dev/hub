import { test, expect, Page, Browser } from '@playwright/test'
import testId from '../../../../src/testId'

const openConfigurationItem = async (page: Page, browser: Browser) => {
    await page.goto('', { waitUntil: 'networkidle' })
    page.on('console', (msg) => console.log(msg.text()))
    await page.request.get('http://localhost:8181/snippet-service/api/v1/configurations/api-reset')

    await page.getByTestId(testId.menu.snippetService.link).click()

    //  wait for submenu to be visible
    if (browser.browserType().name() === 'webkit') {
        await page.waitForTimeout(1000)
    }

    await page.getByTestId(testId.menu.snippetService.configurations).click()
    await page.setViewportSize({ width: 1600, height: 800 })
    await page.getByTestId(`${testId.snippetService.configurations.list.table}-row-0-detail`).click()
}

test('snippet-service-configurations-detail-version', async ({ page, browser }) => {
    await openConfigurationItem(page, browser)

    await expect(page).toHaveTitle(/my-cfg-1 | plgd Dashboard/)
    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.versionSelector}`)).toBeVisible()

    await page.locator('#version').click()
    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.versionSelector}-select-input`)).toBeVisible()

    await page.getByTestId(`${testId.snippetService.configurations.detail.versionSelector}-select-0`).click()

    await expect(page).toHaveTitle(/my-cfg-0 | plgd Dashboard/)
})

test('snippet-service-configurations-detail-invoke', async ({ page, browser }) => {
    await openConfigurationItem(page, browser)

    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.invokeButton}`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.configurations.detail.invokeButton}`).click()

    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.invokeModal}`)).toBeVisible()

    await page.locator('#deviceId').click()
    await page.getByTestId(`${testId.snippetService.configurations.detail.invokeModal}-select-input`).fill('3aae0672-47f3-4498-78d4-b061e6105ccd')
    await page.getByTestId(`${testId.snippetService.configurations.detail.invokeModal}-select-3aae0672-47f3-4498-78d4-b061e6105ccd`).click()

    if (browser.browserType().name() !== 'webkit') {
        await expect(page.getByTestId(`${testId.snippetService.configurations.detail.invokeModal}-footer-reset`)).toBeVisible()
        await expect(page.getByTestId(`${testId.snippetService.configurations.detail.invokeModal}-footer-done`)).toBeVisible()

        await page.getByTestId(`${testId.snippetService.configurations.detail.invokeModal}-footer-done`).click()
    }
    await page.getByTestId(`${testId.snippetService.configurations.detail.invokeModal}-force-label`).click()

    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.invokeModal}-reset`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.invokeModal}-invoke`)).toBeVisible()

    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.invokeModal}-reset`)).not.toBeDisabled()
    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.invokeModal}-invoke`)).not.toBeDisabled()

    await page.getByTestId(`${testId.snippetService.configurations.detail.invokeModal}-invoke`).click()
})

test('snippet-service-configurations-detail-delete', async ({ page, browser }) => {
    await openConfigurationItem(page, browser)

    await expect(page.getByTestId(testId.snippetService.configurations.detail.deleteButton)).toBeVisible()
    await page.getByTestId(testId.snippetService.configurations.detail.deleteButton).click()

    await expect(page.getByTestId(testId.snippetService.configurations.detail.deleteModal)).toBeVisible()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(testId.snippetService.configurations.detail.deleteButtonCancel)).toBeVisible()
    await expect(page.getByTestId(testId.snippetService.configurations.detail.deleteButtonConfirm)).toBeVisible()

    await page.getByTestId(testId.snippetService.configurations.detail.deleteButtonConfirm).click()

    await expect(page.getByTestId(testId.snippetService.configurations.detail.deleteModal)).not.toBeVisible()

    await expect(page).toHaveTitle(/Configurations | plgd Dashboard/)

    await expect(page.getByTestId(`${testId.snippetService.configurations.list.table}-row-0`)).not.toBeVisible()
})

test('snippet-service-configurations-detail-update-fields', async ({ page, browser }) => {
    await openConfigurationItem(page, browser)

    await expect(page.getByTestId(testId.snippetService.configurations.addPage.form.name)).toBeVisible()
    await page.getByTestId(testId.snippetService.configurations.addPage.form.name).fill('my-cfg-2')

    await expect(page.getByTestId(testId.snippetService.configurations.detail.saveButton)).toBeVisible()
    await expect(page.getByTestId(testId.snippetService.configurations.detail.resetButton)).toBeVisible()

    await page.getByTestId(testId.snippetService.configurations.detail.resetButton).click()

    await expect(page.getByTestId(testId.snippetService.configurations.addPage.form.name)).toHaveValue('my-cfg-1')

    await expect(page.getByTestId(testId.snippetService.configurations.detail.saveButton)).not.toBeVisible()
    await expect(page.getByTestId(testId.snippetService.configurations.detail.resetButton)).not.toBeVisible()

    await page.getByTestId(testId.snippetService.configurations.addPage.form.name).fill('my-cfg-2')

    await page.getByTestId(testId.snippetService.configurations.addPage.form.addResourceButton).click()
    await expect(page.getByTestId(`${testId.snippetService.configurations.addPage.form.createResourceModal}-modal`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.configurations.addPage.form.createResourceModal}-input-href`).fill('/oc/con/2')
    await page.getByTestId(`${testId.snippetService.configurations.addPage.form.createResourceModal}-editor-input`).fill('123')

    await expect(page.getByTestId(`${testId.snippetService.configurations.addPage.form.createResourceModal}-confirm-button`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.configurations.addPage.form.createResourceModal}-confirm-button`).click()
    await expect(page.getByTestId(`${testId.snippetService.configurations.addPage.form.createResourceModal}-modal`)).not.toBeVisible()

    await expect(page.getByTestId(`${testId.snippetService.configurations.addPage.form.resourceTable}-row-2`)).toBeVisible()

    await expect(page.getByTestId(testId.snippetService.configurations.detail.saveButton)).toBeVisible()
    await page.getByTestId(testId.snippetService.configurations.detail.saveButton).click()

    await expect(page).toHaveTitle(/Configurations | plgd Dashboard/)
})

test('snippet-service-configurations-detail-tab-conditions', async ({ page, browser }) => {
    await openConfigurationItem(page, browser)

    await expect(page.getByTestId(testId.snippetService.configurations.detail.tabConditions)).toBeVisible()
    await page.getByTestId(testId.snippetService.configurations.detail.tabConditions).click()
    await expect(page.getByTestId(testId.snippetService.configurations.detail.conditionsTable)).toBeVisible()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.conditionsTable}-row-0`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.conditionsTable}-detail`)).toBeVisible()

    await page.getByTestId(`${testId.snippetService.configurations.detail.conditionsTable}-detail`).click()

    await expect(page).toHaveTitle(/jkralik-cond-0 | plgd Dashboard/)
})

test('snippet-service-configurations-detail-tab-applied-configurations', async ({ page, browser }) => {
    await openConfigurationItem(page, browser)

    await expect(page.getByTestId(testId.snippetService.configurations.detail.tabAppliedConfiguration)).toBeVisible()
    await page.getByTestId(testId.snippetService.configurations.detail.tabAppliedConfiguration).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.appliedConfigurationsTable}-detail-link-name`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.configurations.detail.appliedConfigurationsTable}-detail-link-name`).click()

    await expect(page).toHaveTitle(/dps-endpoint-is-set | plgd Dashboard/)
    await expect(page).toHaveURL(/localhost:3000\/snippet-service\/applied-configurations\/79c2a88a-1244-4e8a-a526-420e6cd5d34a/)

    await openConfigurationItem(page, browser)
    await expect(page.getByTestId(testId.snippetService.configurations.detail.tabAppliedConfiguration)).toBeVisible()
    await page.getByTestId(testId.snippetService.configurations.detail.tabAppliedConfiguration).click()

    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.appliedConfigurationsTable}-detail`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.configurations.detail.appliedConfigurationsTable}-detail`).click()

    await expect(page).toHaveTitle(/dps-endpoint-is-set | plgd Dashboard/)
    await expect(page).toHaveURL(/localhost:3000\/snippet-service\/applied-configurations\/79c2a88a-1244-4e8a-a526-420e6cd5d34a/)

    await openConfigurationItem(page, browser)

    await expect(page.getByTestId(testId.snippetService.configurations.detail.tabAppliedConfiguration)).toBeVisible()
    await page.getByTestId(testId.snippetService.configurations.detail.tabAppliedConfiguration).click()

    await expect(page.getByTestId(`${testId.snippetService.configurations.detail.appliedConfigurationsTable}-row-0-condition`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.configurations.detail.appliedConfigurationsTable}-row-0-condition`).click()

    await expect(page).toHaveTitle(/jkralik-cond-0 | plgd Dashboard/)
    await expect(page).toHaveURL(/localhost:3000\/snippet-service\/conditions\/00fa41ad-b3bf-4f00-bfe1-c71c439e4cda/)
})
