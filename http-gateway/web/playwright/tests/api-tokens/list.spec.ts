import { expect, Page, test } from '@playwright/test'
import testId from '../../../src/testId'

const openApiTokensList = async (page: Page) => {
    await page.goto('')
    await page.getByTestId(testId.menu.apiTokens).click()
    await page.setViewportSize({ width: 1600, height: 800 })
}

test('api-tokens-list-open', async ({ page }) => {
    await openApiTokensList(page)

    await expect(page).toHaveTitle(/API Tokens | plgd Dashboard/)
    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
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

test('api-tokens-list-create-token', async ({ page }) => {
    await openApiTokensList(page)

    await expect(page.getByTestId(`${testId.apiTokens.list.table}-row-3-name`)).toBeVisible()
    await page.getByTestId(`${testId.apiTokens.list.table}-row-3-detail`).click()

    await expect(page.getByTestId(testId.apiTokens.list.createTokenButton)).toBeVisible()
    await page.getByTestId(testId.apiTokens.list.createTokenButton).click()

    await expect(page.getByTestId(testId.apiTokens.list.addModal)).toBeVisible()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

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

test('api-tokens-list-link-to-delete-icon', async ({ page }) => {
    await openApiTokensList(page)

    await expect(page.getByTestId(`${testId.apiTokens.list.table}-row-3-name`)).toBeVisible()
    await page.getByTestId(`${testId.apiTokens.list.table}-row-3-delete`).click()

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(`${testId.apiTokens.list.page}-delete-modal`)).toBeVisible()

    await expect(page.getByTestId(`${testId.apiTokens.list.page}-delete-modal-cancel`)).toBeVisible()
    await expect(page.getByTestId(`${testId.apiTokens.list.page}-delete-modal-delete`)).toBeVisible()
    await page.getByTestId(`${testId.apiTokens.list.page}-delete-modal-delete`).click()

    await expect(page.getByTestId(`${testId.apiTokens.list.page}-delete-modal`)).not.toBeVisible()
})
