import { AppContext } from '@/containers/App/AppContext'
import { Router } from 'react-router-dom'
import { history } from '@/store'
import { InitServices } from '@/common/services/init-services'
import { Helmet } from 'react-helmet'
import appConfig from '@/config'
import Container from 'react-bootstrap/Container'
import classNames from 'classnames'
import { StatusBar } from '@/components/status-bar'
import { LeftPanel } from '@/components/left-panel'
import { Menu } from '@/components/menu'
import { Routes } from '@/routes'
import { Footer } from '@shared-ui/components/old/footer'
import {
  BrowserNotificationsContainer,
  ToastContainer,
} from '@/components/toast'
import { useLocalStorage } from '@shared-ui/common/hooks'
import { useAuth } from 'oidc-react'
import { security } from '@shared-ui/common/services'
import AppLoader from '@/containers/App/AppLoader/AppLoader'

const AppInner = props => {
  const { wellKnownConfig, openTelemetry } = props
  const { userData, userManager } = useAuth()
  const [collapsed, setCollapsed] = useLocalStorage('leftPanelCollapsed', true)

  if (userData) {
    security.setAccessToken(userData.access_token)

    if (userManager) {
      security.setUserManager(userManager)
    }
  } else {
    return <AppLoader />
  }

  return (
    <AppContext.Provider
      value={{
        collapsed,
        ...wellKnownConfig,
        wellKnownConfig,
        telemetryWebTracer: openTelemetry.getWebTracer(),
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
            <Footer
              links={[
                {
                  to: 'https://github.com/plgd-dev/hub/raw/main/http-gateway/swagger.yaml',
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

export default AppInner
