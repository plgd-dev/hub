import { expect, Page, test } from '@playwright/test'
import testId from '../../../src/testId'

const openApiTokenDetail = async (page: Page) => {
    await page.goto('', { waitUntil: 'networkidle' })
    await page.request.get('http://localhost:8181/m2m-oauth-server/api/v1/api-reset')

    await page.getByTestId(testId.menu.apiTokens).click()
    await page.setViewportSize({ width: 1600, height: 800 })
    await page.getByTestId(`${testId.apiTokens.list.table}-row-3-detail`).click()
}

test('api-token-detail-screenshot', async ({ page }) => {
    await openApiTokenDetail(page)

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page).toHaveTitle(/jkralik-cond-0-condition | plgd Dashboard/)
})

test('api-token-detail-delete', async ({ page }) => {
    await openApiTokenDetail(page)

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(testId.apiTokens.detail.deleteButton)).toBeVisible()
    await page.getByTestId(testId.apiTokens.detail.deleteButton).click()

    await expect(page.getByTestId(testId.apiTokens.detail.deleteModal)).toBeVisible()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(testId.apiTokens.detail.deleteButtonCancel)).toBeVisible()
    await expect(page.getByTestId(testId.apiTokens.detail.deleteButtonConfirm)).toBeVisible()

    await page.getByTestId(testId.apiTokens.detail.deleteButtonConfirm).click()

    await expect(page.getByTestId(testId.apiTokens.detail.deleteModal)).not.toBeVisible()

    await expect(page).toHaveTitle(/API Tokens | plgd Dashboard/)

    await expect(page.getByTestId(`${testId.apiTokens.list.table}-row-3`)).not.toBeVisible()
})

test('api-token-detail-filter', async ({ page }) => {
    await openApiTokenDetail(page)

    await expect(page.getByTestId(testId.apiTokens.detail.tableGlobalFilter)).toBeVisible()
    await expect(page.getByTestId(`${testId.apiTokens.detail.tableGlobalFilter}-input`)).toBeVisible()
    await page.getByTestId(`${testId.apiTokens.detail.tableGlobalFilter}-input`).fill('id')

    await expect(page.getByTestId(testId.apiTokens.detail.simpleTableLeft)).toBeVisible()
    await expect(page.getByTestId(testId.apiTokens.detail.simpleTableRight)).toBeVisible()

    await expect(page.getByTestId('account.roles-attribute')).not.toBeVisible()
    await expect(page.getByTestId('id-attribute')).toBeVisible()
})
