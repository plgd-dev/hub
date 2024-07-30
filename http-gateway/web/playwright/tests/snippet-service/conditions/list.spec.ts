import { test, expect, Page } from '@playwright/test'
import testId from '../../../../src/testId'

test('snippet-service-configurations-list-open', async ({ page }) => {
    await page.goto('')
    await page.getByTestId(testId.menu.snippetService.link).click()
    await page.getByTestId(testId.menu.snippetService.conditions).click()

    await expect(page).toHaveTitle(/Conditions | plgd Dashboard/)
    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
})

test('snippet-service-configurations-list-link-to-detail-name', async ({ page }) => {
    await page.goto('')
    await page.getByTestId(testId.menu.snippetService.link).click()
    await page.getByTestId(testId.menu.snippetService.conditions).click()

    await expect(page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0-name`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0-name`).click()

    await expect(page).toHaveTitle(/jkralik-cond-0 | plgd Dashboard/)
})

test('snippet-service-configurations-list-link-to-detail-icon', async ({ page }) => {
    await page.goto('')
    await page.getByTestId(testId.menu.snippetService.link).click()
    await page.getByTestId(testId.menu.snippetService.conditions).click()

    page.setViewportSize({ width: 1600, height: 800 })

    await expect(page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0-detail`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.conditions.list.table}-row-0-detail`).click()

    await expect(page).toHaveTitle(/jkralik-cond-0 | plgd Dashboard/)
})

test('snippet-service-configurations-list-add-open-close', async ({ page }) => {
    await page.goto('')
    await page.getByTestId(testId.menu.snippetService.link).click()
    await page.getByTestId(testId.menu.snippetService.conditions).click()

    page.setViewportSize({ width: 1600, height: 800 })

    await expect(page.getByTestId(testId.snippetService.conditions.list.addButton)).toBeVisible()
    await page.getByTestId(testId.snippetService.conditions.list.addButton).click()

    await expect(page).toHaveTitle(/Add Condition | plgd Dashboard/)

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.wizard)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.wizard}-close`)).toBeVisible()

    await page.getByTestId(`${testId.snippetService.conditions.addPage.wizard}-close`).click()
    await expect(page).toHaveTitle(/Conditions | plgd Dashboard/)
})

const openConditionFilter = async (page: Page, locator: string) => {
    await expect(page.getByTestId(locator)).toBeVisible()
    await page.getByTestId(`${locator}-header`).click()
    await expect(page.getByTestId(`${locator}-content`)).toBeVisible()
}

const addAndCheckFilter = async (page: Page, locator: string) => {
    await page.getByTestId(`${locator}-input`).fill('/oic/p')
    await page.getByTestId(`${locator}-addButton`).click()

    await expect(page.getByTestId(`${locator}-content-table-row-0-attribute`)).toBeVisible()
    await expect(page.getByTestId(`${locator}-content-table-row-0-value`)).toBeVisible()
}

const removeAndCheck = async (page: Page, locator: string) => {
    await page.getByTestId(`${locator}-content-table-row-0-remove`).click()

    await expect(page.getByTestId(`${locator}-content-table-row-0-attribute`)).not.toBeVisible()
    await expect(page.getByTestId(`${locator}-content-table-row-0-value`)).not.toBeVisible()
}

test('snippet-service-configurations-list-add', async ({ page }) => {
    await page.goto('')
    await page.getByTestId(testId.menu.snippetService.link).click()
    await page.getByTestId(testId.menu.snippetService.conditions).click()

    page.setViewportSize({ width: 1600, height: 800 })

    await expect(page.getByTestId(testId.snippetService.conditions.list.addButton)).toBeVisible()
    await page.getByTestId(testId.snippetService.conditions.list.addButton).click()

    await expect(page).toHaveTitle(/Add Condition | plgd Dashboard/)

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await page.getByTestId(testId.snippetService.conditions.addPage.step1.form.name).fill('cond-02')

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step1.buttons}-continue`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step1.buttons}-continue`)).not.toBeDisabled()

    await page.getByTestId(`${testId.snippetService.conditions.addPage.step1.buttons}-continue`).click()

    // ******** STEP2
    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })

    await expect(page).toHaveURL(/\/snippet-service\/conditions\/add\/apply-filters/)
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-back`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-back`)).not.toBeDisabled()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-continue`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-continue`)).toBeDisabled()

    // --- device Id selector
    await openConditionFilter(page, testId.snippetService.conditions.addPage.step2.filterDeviceId)

    await page.locator('#deviceIdFilter').focus()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.selectDeviceId}-input`)).toBeVisible()

    // select device
    await page.locator('#deviceIdFilter').click()
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.selectDeviceId}-input`).fill('3aae0672-47f3-4498-78d4-b061e6105ccd')
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.selectDeviceId}-3aae0672-47f3-4498-78d4-b061e6105ccd`).click()

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step2.selectDeviceIdReset)).toBeVisible()
    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step2.selectDeviceIdDone)).toBeVisible()

    await page.getByTestId(testId.snippetService.conditions.addPage.step2.selectDeviceIdDone).click()

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-attribute`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-value`)).toBeVisible()

    // remove selected device
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-remove`).click()

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-attribute`)).not.toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-value`)).not.toBeVisible()

    // select device back
    await page.locator('#deviceIdFilter').click()
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.selectDeviceId}-input`).fill('3aae0672-47f3-4498-78d4-b061e6105ccd')
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.selectDeviceId}-3aae0672-47f3-4498-78d4-b061e6105ccd`).click()

    await page.getByTestId(testId.snippetService.conditions.addPage.step2.selectDeviceIdDone).click()

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-attribute`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.filterDeviceId}-content-table-row-0-value`)).toBeVisible()

    // --- resourceType selector
    await openConditionFilter(page, testId.snippetService.conditions.addPage.step2.resourceType)

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.resourceType}-input`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.resourceType}-addButton`)).toBeVisible()

    await addAndCheckFilter(page, testId.snippetService.conditions.addPage.step2.resourceType)
    await removeAndCheck(page, testId.snippetService.conditions.addPage.step2.resourceType)
    await addAndCheckFilter(page, testId.snippetService.conditions.addPage.step2.resourceType)

    // --- hrefFilter selector
    await openConditionFilter(page, testId.snippetService.conditions.addPage.step2.hrefFilter)

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.hrefFilter}-input`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.hrefFilter}-addButton`)).toBeVisible()

    await addAndCheckFilter(page, testId.snippetService.conditions.addPage.step2.hrefFilter)
    await removeAndCheck(page, testId.snippetService.conditions.addPage.step2.hrefFilter)
    await addAndCheckFilter(page, testId.snippetService.conditions.addPage.step2.hrefFilter)

    // --- jqExpression filter
    await openConditionFilter(page, testId.snippetService.conditions.addPage.step2.jqExpressionFilter)

    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.jqExpressionFilter}-input`).fill('.n == "new name value')

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-back`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-back`)).not.toBeDisabled()

    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-continue`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-continue`)).not.toBeDisabled()

    await page.getByTestId(`${testId.snippetService.conditions.addPage.step2.buttons}-continue`).click()

    // ******** STEP3

    await expect(page).toHaveScreenshot({ fullPage: true, omitBackground: true, animations: 'disabled' })
    await expect(page).toHaveURL(/\/snippet-service\/conditions\/add\/select-configuration/)

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step3.selectConfiguration)).toBeVisible()
    await page.locator('#configurationId').click()
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step3.selectConfiguration}-48998f7d-2a70-46a4-8a68-745b69d55489`).click()

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step3.apiToken)).toBeVisible()
    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step3.generateApiToken)).toBeVisible()

    await page.getByTestId(testId.snippetService.conditions.addPage.step3.generateApiToken).click()

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step3.generateApiTokenModal)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step3.generateApiTokenModal}-invoke`)).toBeVisible()
    await page.getByTestId(`${testId.snippetService.conditions.addPage.step3.generateApiTokenModal}-invoke`).click()

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step3.generateApiTokenModal)).not.toBeVisible()

    await expect(page.getByTestId(testId.snippetService.conditions.addPage.step3.apiToken)).toHaveValue(
        'eyJhbGciOiJFUzI1NiIsImtpZCI6IjdlODU5NTExLWIyMDYtNTlmYi05MGZmLTE0NTQzZTBjNWRjZCIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsiaHR0cHM6Ly90cnkucGxnZC5jbG91ZCJdCiwiY2xpZW50X2lkIjoiand0LXByaXZhdGUta2V5IiwiZXhwIjoxNzI0ODUwMzczCiwiaHR0cHM6Ly9wbGdkLmRldi9vcmlnaW5hbENsYWltcyI6eyJhY3IiOiIwIiwiYWxsb3dlZC1vcmlnaW5zIjpbImh0dHA6Ly8xMjcuMC4wLjE6MzAwMCIsImh0dHA6Ly9sb2NhbGhvc3Q6MzAwMCIsImh0dHBzOi8vdHJ5LnBsZ2QuY2xvdWQiXSwiYXVkIjoiYWNjb3VudCIsImF1dGhfdGltZSI6MTcyMjI0NDIzMCwiYXpwIjoiTFhaOU9oS1dXUllxZjEyVzBCNU9YZHVxdDAycTB6alMiLCJlbWFpbCI6InRlc3QudXNlci5vY2ZjbG91ZEBnbWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwiZXhwIjoxNzIyMjYyNjYwLCJmYW1pbHlfbmFtZSI6IlRlc3QiLCJnaXZlbl9uYW1lIjoiVGVzdCIsImlhdCI6MTcyMjI1NTQ2MCwiaXNzIjoiaHR0cHM6Ly9hdXRoLnBsZ2QuY2xvdWQvcmVhbG1zL3NoYXJlZCIsImp0aSI6IjJlOGNjZGRmLTE2MjYtNDQwMC1iZDU5LTdlZjE3ODQ4Nzc4YiIsIm5hbWUiOiJUZXN0IFRlc3QiLCJvd25lci1pZCI6ImJlYjMyNzc3LTk2ODAtNGY0Mi04NzYxLTM1MGVlYmU3NmE4NSIsInByZWZlcnJlZF91c2VybmFtZSI6InRlc3QudXNlci5vY2ZjbG91ZEBnbWFpbC5jb20iLCJyZWFsbV9hY2Nlc3MiOnsicm9sZXMiOlsiZGVmYXVsdC1yb2xlcy1zaGFyZWQiLCJvZmZsaW5lX2FjY2VzcyIsInVtYV9hdXRob3JpemF0aW9uIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYWNjb3VudCI6eyJyb2xlcyI6WyJtYW5hZ2UtYWNjb3VudCIsIm1hbmFnZS1hY2NvdW50LWxpbmtzIiwidmlldy1wcm9maWxlIl19fSwic2NvcGUiOiJvcGVuaWQgZW1haWwgcHJvZmlsZSIsInNlc3Npb25fc3RhdGUiOiJlMzBlMjllNS1hYmM5LTQ3ZGQtOTVhMC05NjljMmZiYTVjNGIiLCJzaWQiOiJlMzBlMjllNS1hYmM5LTQ3ZGQtOTVhMC05NjljMmZiYTVjNGIiLCJzdWIiOiJiZWIzMjc3Ny05NjgwLTRmNDItODc2MS0zNTBlZWJlNzZhODUiLCJ0eXAiOiJCZWFyZXIifSwiaWF0IjoxNzIyMjU4OTM5CiwiaXNzIjoiaHR0cHM6Ly90cnkucGxnZC5jbG91ZCIsImp0aSI6IjMxZDFjMjAyLWJlN2QtNDAzMy1hZTAwLTczODcwMzNlMWFhZiIsIm5hbWUiOiJlZWUtY29uZGl0aW9uIiwib3duZXItaWQiOiJiZWIzMjc3Ny05NjgwLTRmNDItODc2MS0zNTBlZWJlNzZhODUiLCJzY29wZSI6IiIsInN1YiI6ImJlYjMyNzc3LTk2ODAtNGY0Mi04NzYxLTM1MGVlYmU3NmE4NSJ9.wc_iLUEhMAiUEkvOe_1_flnQeGfbSt98cG7HctPlxuPseOxEdQp7OVi9qlXykk9-6ZGpiLey9VExaXuemPRYHA'
    )

    // buttons
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step3.buttons}-back`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step3.buttons}-back`)).not.toBeDisabled()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step3.buttons}-continue`)).toBeVisible()
    await expect(page.getByTestId(`${testId.snippetService.conditions.addPage.step3.buttons}-continue`)).not.toBeDisabled()

    await page.getByTestId(`${testId.snippetService.conditions.addPage.step3.buttons}-continue`).click()

    await expect(page).toHaveURL(/\/snippet-service\/conditions/)
    await expect(page).toHaveTitle(/Conditions | plgd Dashboard/)
})
