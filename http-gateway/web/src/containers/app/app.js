import { hot } from 'react-hot-loader/root'
import { useContext, useState, useEffect } from 'react'
import { useAuth0 } from '@auth0/auth0-react'
import classNames from 'classnames'
import { Router } from 'react-router-dom'
import Container from 'react-bootstrap/Container'
import { Helmet } from 'react-helmet'
import { useIntl } from 'react-intl'
import {
  ToastContainer,
  BrowserNotificationsContainer,
} from '@shared-ui/components/old/toast'
import { PageLoader } from '@shared-ui/components/old/page-loader'
import { LeftPanel } from '@shared-ui/components/old/left-panel'
import { Menu } from '@shared-ui/components/old/menu'
import { StatusBar } from '@shared-ui/components/old/status-bar'
import { Footer } from '@shared-ui/components/old/footer'
import { useLocalStorage } from '@shared-ui/common/hooks'
import { Routes } from '@/routes'
import { history } from '@/store/history'
import { security } from '@shared-ui/common/services/security'
import { openTelemetry } from '@shared-ui/common/services/opentelemetry'
import { InitServices } from '@shared-ui/common/services/init-services'
import appConfig from '@/config'
import { fetchApi } from '@shared-ui/common/services'
import { messages as t } from './app-i18n'
import { AppContext } from './app-context'
import './app.scss'

const App = ({ config }) => {
  const {
    isLoading,
    isAuthenticated,
    error,
    loginWithRedirect,
    getAccessTokenSilently,
  } = useAuth0()
  const [collapsed, setCollapsed] = useLocalStorage('leftPanelCollapsed', true)
  const { formatMessage: _ } = useIntl()
  const [wellKnownConfig, setWellKnownConfig] = useState(null)
  const [wellKnownConfigFetched, setWellKnownConfigFetched] = useState(false)
  const [configError, setConfigError] = useState(null)

  // Set the getAccessTokenSilently method to the security singleton
  security.setAccessTokenSilently(getAccessTokenSilently)

  // Set the auth configurations
  const {
    webOauthClient,
    deviceOauthClient,
    openTelemetry: openTelemetryConfig,
    ...generalConfig
  } = config
  security.setGeneralConfig({ ...generalConfig, useSecurity: true })
  security.setWebOAuthConfig(webOauthClient)
  security.setDeviceOAuthConfig(deviceOauthClient)
  openTelemetryConfig !== false && openTelemetry.init('hub')

  useEffect(() => {
    if (
      !isLoading &&
      isAuthenticated &&
      !wellKnownConfig &&
      !wellKnownConfigFetched
    ) {
      const fetchWellKnownConfig = async () => {
        try {
          const { data: wellKnown } = await openTelemetry.withTelemetry(
            () =>
              fetchApi(
                `${config.httpGatewayAddress}/.well-known/hub-configuration`
              ),
            'get-hub-configuration'
          )

          setWellKnownConfigFetched(true)
          setWellKnownConfig(wellKnown)
        } catch (e) {
          setConfigError(
            new Error(
              'Could not retrieve the well-known ocfcloud configuration.'
            )
          )
        }
      }

      fetchWellKnownConfig()
    }
  }, [
    isLoading,
    isAuthenticated,
    wellKnownConfig,
    wellKnownConfigFetched,
    config.httpGatewayAddress,
  ])

  // Render an error box with an auth error
  if (error || configError) {
    return (
      <div className="client-error-message">
        {`${_(t.authError)}: ${error?.message || configError?.message}`}
      </div>
    )
  }

  // Placeholder loader while waiting for the auth status
  const renderLoader = () => {
    return (
      <>
        <PageLoader className="auth-loader" loading />
        <div className="page-loading-text">{`${_(t.loading)}...`}</div>
      </>
    )
  }

  // If the loading is finished but still unauthenticated, it means the user is not logged in.
  // Calling the loginWithRedirect will make a redirect to the login page where the user can login.
  if (!isLoading && !isAuthenticated) {
    loginWithRedirect({
      appState: {
        returnTo: window.location.href.substr(window.location.origin.length),
      },
    })

    return renderLoader()
  }

  if (isLoading || !wellKnownConfig) {
    return renderLoader()
  }

  return (
    <AppContext.Provider
      value={{
        ...config,
        collapsed,
        wellKnownConfig,
        telemetryWebTracer:
          openTelemetryConfig !== false
            ? openTelemetry.getWebTracer()
            : undefined,
        useSecurity: true,
      }}
    >
      <Router history={history}>
        <InitServices />
        <Helmet
          defaultTitle={appConfig.appName}
          titleTemplate={`%s | ${appConfig.appName}`}
        />
        <Container fluid id="app" className={classNames({ collapsed })}>
          <StatusBar />
          <LeftPanel>
            <Menu
              menuItems={[
                {
                  to: '/',
                  icon: 'fa-list',
                  nameKey: 'devices',
                  className: 'devices',
                },
                {
                  to: '/pending-commands',
                  icon: 'fa-compress-alt',
                  nameKey: 'pendingCommands',
                },
              ]}
              collapsed={collapsed}
              toggleCollapsed={() => setCollapsed(!collapsed)}
            />
          </LeftPanel>
          <div id="content">
            <Routes />
            <Footer
              links={[
                {
                  to: 'https://github.com/plgd-dev/hub/raw/master/http-gateway/swagger.yaml',
                  i18key: 'API',
                },
                {
                  to: 'https://plgd.dev/documentation',
                  i18key: 'docs',
                },
                {
                  to: 'https://github.com/plgd-dev/hub',
                  i18key: 'contribute',
                },
              ]}
            />
          </div>
        </Container>
        <ToastContainer />
        <BrowserNotificationsContainer />
      </Router>
    </AppContext.Provider>
  )
}

export const useAppConfig = () => useContext(AppContext)

export default hot(App)
