import React from 'react'
import { createRoot } from 'react-dom/client'
import { Provider } from 'react-redux'
import { persistStore } from 'redux-persist'
import { PersistGate } from 'redux-persist/integration/react'

import IntlProvider from '@shared-ui/components/Atomic/IntlProvider'

import { App } from '@/containers/App'
import { store } from '@/store'
// @ts-ignore
import languages from './languages/languages.json'
import appConfig from '@/config'
import { DEVICE_AUTH_CODE_SESSION_KEY } from './constants'
import reportWebVitals from './reportWebVitals'

let persistor = persistStore(store)

const BaseComponent = () => {
    // When the URL contains a get parameter called `code` and the pathname is set to `/devices`,
    // that means we were redirected from the get auth code endpoint and we must not render the app,
    // only set the code to the session storage, so that the caller can process it.
    const urlParams = new URLSearchParams(window.location.search)
    const code = urlParams.get('code')
    const isMockApp = window.location.pathname === '/devices-code-redirect' && !!code

    if (window.location.pathname === '/devices' && code) {
        localStorage.setItem(DEVICE_AUTH_CODE_SESSION_KEY, code)

        window.location.hash = ''
        window.location.href = `${window.location.origin}/devices-code-redirect?code=${code}`

        return null
    }

    if (isMockApp) {
        window.addEventListener('load', function () {
            setInterval(() => {
                if (localStorage.getItem(DEVICE_AUTH_CODE_SESSION_KEY)) {
                    // window.close()
                }
            }, 200)
        })
    }

    return (
        <Provider store={store}>
            <PersistGate persistor={persistor}>
                <IntlProvider defaultLanguage={appConfig.defaultLanguage} languages={languages}>
                    <App mockApp={isMockApp} />
                </IntlProvider>
            </PersistGate>
        </Provider>
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
