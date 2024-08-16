import { defineConfig, devices } from '@playwright/test'

/**
 * Read environment variables from file.
 * https://github.com/motdotla/dotenv
 */
require('dotenv').config()

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
    globalSetup: require.resolve('./global-setup'),
    testDir: './tests',
    /* Maximum time one test can run for. */
    timeout: 2 * 60 * 1000,
    expect: {
        timeout: 2 * 60 * 1000,
        toHaveScreenshot: { maxDiffPixels: 100 },
    },
    /* Run tests in files in parallel */
    fullyParallel: true,
    /* Fail the build on CI if you accidentally left test.only in the source code. */
    forbidOnly: !!process.env.CI,
    /* Retry on CI only */
    retries: process.env.CI ? 2 : 0,
    /* Opt out of parallel tests on CI. */
    workers: process.env.CI ? 1 : undefined,
    /* Reporter to use. See https://playwright.dev/docs/test-reporters */
    reporter: 'html',
    /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
    use: {
        /* Maximum time each action such as `click()` can take. Defaults to 0 (no limit). */
        actionTimeout: 2 * 60 * 1000,
        baseURL: 'http://localhost:3000/',
        storageState: 'storageState.json',
        trace: 'on-first-retry',
        testIdAttribute: 'data-test-id',
        viewport: {
            width: 1400,
            height: 800,
        },
    },

    /* Configure projects for major browsers */
    projects: [
        {
            name: 'chromium',
            use: {
                ...devices['Desktop Chrome'],
            },
        },

        {
            name: 'firefox',
            use: {
                ...devices['Desktop Firefox'],
            },
        },
        
        {
            name: 'webkit',
            use: {
                ...devices['Desktop Safari'],
            },
        },

        /* Test against mobile viewports. */
        // {
        //   name: 'Mobile Chrome',
        //   use: {
        //     ...devices['Pixel 5'],
        //   },
        // },
        // {
        //   name: 'Mobile Safari',
        //   use: {
        //     ...devices['iPhone 12'],
        //   },
        // },

        // {
        //     name: 'Microsoft Edge',
        //     use: {
        //         channel: 'msedge',
        //     },
        // },
        // {
        //     name: 'Google Chrome',
        //     use: {
        //         channel: 'chrome',
        //     },
        // },
    ],

    /* Folder for test artifacts such as screenshots, videos, traces, etc. */
    outputDir: 'test-results/',

    /* Run your local dev server before starting the tests */
    // webServer: {
    //     command: 'npm run start',
    //     port: 3000,
    // },
})
