import React from 'react'
import ReactDOM from 'react-dom'
import { Provider } from 'react-redux'
import { App } from '@/containers/App'
import { store } from '@/store'
import { IntlProvider } from '@/components/intl-provider'

import { DEVICE_AUTH_CODE_SESSION_KEY } from './constants'
import reportWebVitals from './reportWebVitals'

const BaseComponent = () => {
  // When the URL contains a get parameter called `code` and the pathname is set to `/devices`,
  // that means we were redirected from the get auth code endpoint and we must not render the app,
  // only set the code to the session storage, so that the caller can process it.
  const urlParams = new URLSearchParams(window.location.search)
  const code = urlParams.get('code')
  if (window.location.pathname === '/devices' && code) {
    sessionStorage.setItem(DEVICE_AUTH_CODE_SESSION_KEY, code)
    return null
  }

  return (
    <Provider store={store}>
      <IntlProvider>
        <App />
      </IntlProvider>
    </Provider>
  )
}

const render = () => {
  ReactDOM.render(<BaseComponent />, document.getElementById('root'))
}

render()

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals()
