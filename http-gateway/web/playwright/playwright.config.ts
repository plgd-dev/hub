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
    timeout: 60 * 1000,
    expect: {
        timeout: 60 * 1000,
        toHaveScreenshot: { maxDiffPixels: 100 },
    },
    /* Run tests in files in parallel */
    fullyParallel: true,
    /* Fail the build on CI if you accidentally left test.only in the source code. */
    forbidOnly: !!process.env.CI,
    /* Retry on CI only */
    retries: process.env.CI ? 2 : 0,
    /* Opt out of parallel tests on CI. */
    workers: 1,
    /* Reporter to use. See https://playwright.dev/docs/test-reporters */
    reporter: 'html',
    /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
    use: {
        /* Maximum time each action such as `click()` can take. Defaults to 0 (no limit). */
        actionTimeout: 60 * 1000,
        baseURL: 'http://localhost:3000/',
        screenshot: 'only-on-failure',
        trace: 'on-first-retry',
        testIdAttribute: 'data-test-id',
        video: 'retain-on-failure',
        viewport: {
            width: 1600,
            height: 800,
        },
    },

    /* Configure projects for major browsers */
    projects: [
        {
            name: 'chromium',
            use: {
                ...devices['Desktop Chrome'],
                storageState: 'storageState.chromium.json',
            },
        },
        // {
        //     name: 'firefox',
        //     use: {
        //         ...devices['Desktop Firefox'],
        //         storageState: 'storageState.firefox.json',
        //     },
        //     fullyParallel: false,
        // },
        {
            name: 'webkit',
            use: {
                ...devices['Desktop Safari'],
                storageState: 'storageState.webkit.json',
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
