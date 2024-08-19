import { expect, Page, test } from '@playwright/test'
import testId from '../../../src/testId'
import { takeScreenshot } from '../utils'

const openApiTokensList = async (page: Page) => {
    await page.goto('', { waitUntil: 'networkidle' })
    await page.request.get('http://localhost:8181/m2m-oauth-server/api/v1/api-reset')

    await page.getByTestId(testId.menu.apiTokens).click()
    await page.setViewportSize({ width: 1600, height: 800 })
}

test('api-tokens-list-open', async ({ page, browser }) => {
    await openApiTokensList(page)

    await expect(page).toHaveTitle(/API Tokens | plgd Dashboard/)
    await takeScreenshot(page, browser)
})

test('api-tokens-list-link-to-detail-name', async ({ page }) => {
    await openApiTokensList(page)

    await expect(page.getByTestId(`${testId.apiTokens.list.table}-row-3-name`)).toBeVisible()
    await page.getByTestId(`${testId.apiTokens.list.table}-row-3-name`).click()

    await expect(page).toHaveTitle(/jkralik-cond-0-condition | plgd Dashboard/)
})

test('api-tokens-list-link-to-detail-icon', async ({ page }) => {
    await openApiTokensList(page)

    await expect(page.getByTestId(`${testId.apiTokens.list.table}-row-3-name`)).toBeVisible()
    await page.getByTestId(`${testId.apiTokens.list.table}-row-3-detail`).click()

    await expect(page).toHaveTitle(/jkralik-cond-0-condition | plgd Dashboard/)
})

test('api-tokens-list-create-token', async ({ page, browser }) => {
    await openApiTokensList(page)

    await expect(page.getByTestId(testId.apiTokens.list.createTokenButton)).toBeVisible()
    await page.getByTestId(testId.apiTokens.list.createTokenButton).click()

    await expect(page.getByTestId(testId.apiTokens.list.addModal)).toBeVisible()

    await takeScreenshot(page, browser)

    await page.getByTestId(`${testId.apiTokens.list.addModal}-form-name`).fill('new-token-name')

    await expect(page.getByTestId(`${testId.apiTokens.list.addModal}-reset`)).toBeVisible()
    await expect(page.getByTestId(`${testId.apiTokens.list.addModal}-generate`)).toBeVisible()

    await page.getByTestId(`${testId.apiTokens.list.addModal}-generate`).click()

    await expect(page.getByTestId(`${testId.apiTokens.list.addModal}-alert`)).toBeVisible()
    await expect(page.getByTestId(`${testId.apiTokens.list.addModal}-copy`)).toBeVisible()

    await expect(page.getByTestId(`${testId.apiTokens.list.addModal}-close`)).toBeVisible()
    await page.getByTestId(`${testId.apiTokens.list.addModal}-close`).click()

    await expect(page.getByTestId(testId.apiTokens.list.addModal)).not.toBeVisible()
})

test('api-tokens-list-link-to-delete-icon', async ({ page, browser }) => {
    await openApiTokensList(page)

    await expect(page.getByTestId(`${testId.apiTokens.list.table}-row-3-name`)).toBeVisible()
    await page.getByTestId(`${testId.apiTokens.list.table}-row-3-delete`).click()

    await takeScreenshot(page, browser)

    await expect(page.getByTestId(`${testId.apiTokens.list.page}-delete-modal`)).toBeVisible()

    await expect(page.getByTestId(`${testId.apiTokens.list.page}-delete-modal-cancel`)).toBeVisible()
    await expect(page.getByTestId(`${testId.apiTokens.list.page}-delete-modal-delete`)).toBeVisible()
    await page.getByTestId(`${testId.apiTokens.list.page}-delete-modal-delete`).click()

    await expect(page.getByTestId(`${testId.apiTokens.list.page}-delete-modal`)).not.toBeVisible()
})
