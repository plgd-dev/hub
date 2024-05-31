import React from 'react'
import { createRoot } from 'react-dom/client'
import { Provider } from 'react-redux'
import { persistStore } from 'redux-persist'
import { PersistGate } from 'redux-persist/integration/react'
import { RecoilRoot } from 'recoil'

import IntlProvider from '@shared-ui/components/Atomic/IntlProvider'
import App from '@shared-ui/components/Atomic/App'

import { App as MainApp } from '@/containers/App'
import { store } from '@/store'
// @ts-ignore
import languages from './languages/languages.json'
import appConfig from '@/config'
import { CONFIGURATION_PAGE_FRAME, DEVICE_AUTH_CODE_SESSION_KEY } from './constants'
import reportWebVitals from './reportWebVitals'
import PreviewApp from '@/containers/Configuration/PreviewApp/PreviewApp'

let persistor = persistStore(store)

const BaseComponent = () => {
    // When the URL contains a get parameter called `code` and the pathname is set to `/devices`,
    // that means we were redirected from the get auth code endpoint and we must not render the app,
    // only set the code to the session storage, so that the caller can process it.
    const urlParams = new URLSearchParams(window.location.search)
    const code = urlParams.get('code')
    // state is present from auth request
    const state = urlParams.get('state')
    const isMockApp = window.location.pathname === '/devices-code-redirect' && !!code
    const configurationPageFrame = window.location.pathname === `/${CONFIGURATION_PAGE_FRAME}`

    // onboarding device
    if (window.location.pathname === '/devices' && code && !state) {
        localStorage.setItem(DEVICE_AUTH_CODE_SESSION_KEY, code)

        window.location.hash = ''
        window.location.href = `${window.location.origin}/devices-code-redirect?code=${code}`

        return null
    }

    if (isMockApp) {
        console.log('plgd mock app is running...')
        window.addEventListener('load', function () {
            setInterval(() => {
                if (localStorage.getItem(DEVICE_AUTH_CODE_SESSION_KEY)) {
                    window.close()
                }
            }, 200)
        })
    }

    const ProviderWrapper = ({ children }: { children: any }) => (
        <Provider store={store}>
            <RecoilRoot>
                <PersistGate persistor={persistor}>
                    <IntlProvider defaultLanguage={appConfig.defaultLanguage} languages={languages}>
                        {children}
                    </IntlProvider>
                </PersistGate>
            </RecoilRoot>
        </Provider>
    )

    if (configurationPageFrame) {
        return (
            <ProviderWrapper>
                <App toastContainerPortalTarget={document.getElementById('toast-root')}>
                    <PreviewApp />
                </App>
            </ProviderWrapper>
        )
    }

    // save the current pathname to the local storage before sign-out
    if (!/devices-code-redirect/.test(window.location.pathname)) {
        window.localStorage.setItem('storedPathname', window.location.pathname.toString())
    }

    return (
        <ProviderWrapper>
            <MainApp mockApp={isMockApp} />
        </ProviderWrapper>
    )
}

const renderApp = () => {
    const root = createRoot(document.getElementById('root') as Element)
    root.render(<BaseComponent />)
}

renderApp()

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals()
