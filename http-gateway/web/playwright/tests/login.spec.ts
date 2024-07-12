import { test } from '@playwright/test'
import { login } from './utils'

test('login action', async ({ page }) => {
    await login(page)
})
