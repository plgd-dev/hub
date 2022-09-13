import { hot } from 'react-hot-loader/root'
import { useContext, useState, useEffect } from 'react'
import { useAuth } from 'oidc-react'
import classNames from 'classnames'
import { Router } from 'react-router-dom'
import Container from 'react-bootstrap/Container'
import { Helmet } from 'react-helmet'
import { useIntl } from 'react-intl'
import {
  ToastContainer,
  BrowserNotificationsContainer,
} from '@/components/toast'
import { PageLoader } from '@/components/page-loader'
import { LeftPanel } from '@/components/left-panel'
import { Menu } from '@/components/menu'
import { StatusBar } from '@/components/status-bar'
import { Footer } from '@/components/footer'
import { useLocalStorage } from '@/common/hooks'
import { Routes } from '@/routes'
import { history } from '@/store/history'
import { security } from '@/common/services/security'
import { openTelemetry } from '@/common/services/opentelemetry'
import { InitServices } from '@/common/services/init-services'
import appConfig from '@/config'
import { fetchApi } from '@/common/services'
import { messages as t } from './app-i18n'
import { AppContext } from './app-context'
import './app.scss'

const App = ({ config }) => {
  const { isLoading, userData } = useAuth()
  const [collapsed, setCollapsed] = useLocalStorage('leftPanelCollapsed', true)
  const { formatMessage: _ } = useIntl()
  const [wellKnownConfig, setWellKnownConfig] = useState(null)
  const [wellKnownConfigFetched, setWellKnownConfigFetched] = useState(false)
  const [configError, setConfigError] = useState(null)

  // Set the auth configurations
  const {
    webOauthClient,
    deviceOauthClient,
    openTelemetry: openTelemetryConfig,
    ...generalConfig
  } = config
  security.setGeneralConfig(generalConfig)
  security.setWebOAuthConfig(webOauthClient)
  security.setDeviceOAuthConfig(deviceOauthClient)
  openTelemetryConfig !== false && openTelemetry.init('hub')

  if (userData) {
    security.setAccessToken(userData.access_token)
  }

  useEffect(() => {
    if (!isLoading && userData && !wellKnownConfig && !wellKnownConfigFetched) {
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
    userData,
    wellKnownConfig,
    wellKnownConfigFetched,
    config.httpGatewayAddress,
  ])

  // Render an error box with an auth error
  if (configError) {
    return (
      <div className="client-error-message">
        {`${_(t.authError)}: ${configError?.message}`}
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
              collapsed={collapsed}
              toggleCollapsed={() => setCollapsed(!collapsed)}
            />
          </LeftPanel>
          <div id="content">
            <Routes />
            <Footer />
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
