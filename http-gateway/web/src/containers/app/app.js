import { hot } from 'react-hot-loader/root'
import { useAuth0 } from '@auth0/auth0-react'
import classNames from 'classnames'
import { Router } from 'react-router-dom'
import Container from 'react-bootstrap/Container'
import { Helmet } from 'react-helmet'
import { useIntl } from 'react-intl'

import { LeftPanel } from '@/components/left-panel'
import { Menu } from '@/components/menu'
import { StatusBar } from '@/components/status-bar'
import { Footer } from '@/components/footer'
import { useLocalStorage } from '@/common/hooks'
import { Routes } from '@/routes'
import { history } from '@/store/history'
import { messages as t } from './app-i18n'
import './app.scss'

const App = () => {
  const { isLoading, isAuthenticated, error, loginWithRedirect } = useAuth0()
  const [collapsed, setCollapsed] = useLocalStorage('leftPanelCollapsed', false)
  const { formatMessage: _ } = useIntl()

  if (error) {
    return <div>Oops... {error.message}</div>
  }

  if (isLoading) {
    return <div>{_(t.loading)}</div>
  }

  if (!isLoading && !isAuthenticated) {
    loginWithRedirect({
      appState: {
        returnTo: window.location.href.substr(window.location.origin.length),
      },
    })

    return <div>{_(t.loading)}</div>
  }

  return (
    <Router history={history}>
      <Helmet
        defaultTitle={_(t.defaultTitle)}
        titleTemplate={`%s | ${_(t.defaultTitle)}`}
      />
      <Container fluid id="app" className={classNames({ collapsed })}>
        <LeftPanel>
          <Menu
            collapsed={collapsed}
            toggleCollapsed={() => setCollapsed(!collapsed)}
          />
        </LeftPanel>
        <StatusBar />
        <div id="content">
          <Routes />
          <Footer />
        </div>
      </Container>
    </Router>
  )
}

export default hot(App)
