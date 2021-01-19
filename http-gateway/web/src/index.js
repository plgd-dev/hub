import React from 'react'
import ReactDOM from 'react-dom'
// import { IntlProvider } from 'react-intl'
import { Provider } from 'react-redux'
import { Auth0Provider } from '@auth0/auth0-react'
// import { useHistory } from 'react-router-dom'

import { App } from '@/containers/app'
import { store } from '@/store'
// import languages from '@/languages/languages.json'
import { history } from '@/store/history'
import { IntlProvider } from '@/components/intl-provider'

// import config from './auth_config.json'
import reportWebVitals from './reportWebVitals'

fetch('/auth_config.json')
  .then(response => response.json())
  .then(config => {
    const BaseComponent = () => {
      const onRedirectCallback = appState => {
        const { returnTo } = appState || {}
        // Use the router's history module to replace the url
        if (returnTo && returnTo !== '/') {
          history.replace(appState.returnTo)
        }
      }

      return (
        <Provider store={store}>
          <IntlProvider>
            <Auth0Provider
              domain={config.domain}
              clientId={config.clientId}
              redirectUri={window.location.origin}
              onRedirectCallback={onRedirectCallback}
              audience={config.audience}
              scope={config.scope}
            >
              <App />
            </Auth0Provider>
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
  })
