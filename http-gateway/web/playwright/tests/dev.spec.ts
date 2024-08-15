import { expect, test } from '@playwright/test'
import testId from '../../src/testId'

test('dev-test', async ({ page }) => {
    await page.goto('')

    await expect(page.getByTestId(testId.app.layout)).toBeVisible()
})
