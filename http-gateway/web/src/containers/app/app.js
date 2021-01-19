import { hot } from 'react-hot-loader/root'
import { useAuth0 } from '@auth0/auth0-react'
import Container from 'react-bootstrap/Container'
import classNames from 'classnames'
import { Router } from 'react-router-dom'

import { LeftPanel } from '@/components/left-panel'
import { Menu } from '@/components/menu'
import { StatusBar } from '@/components/status-bar'
import { Footer } from '@/components/footer'
import { Routes } from '@/routes'
import { useLocalStorage } from '@/common/hooks'
import { history } from '@/store/history'
import './app.scss'

const App = () => {
  const { isLoading, isAuthenticated, error, loginWithRedirect } = useAuth0()
  const [collapsed, setCollapsed] = useLocalStorage('leftPanelCollapsed', false)

  if (error) {
    return <div>Oops... {error.message}</div>
  }

  if (isLoading) {
    return <div>{'Loading'}</div>
  }

  if (!isLoading && !isAuthenticated) {
    loginWithRedirect({ appState: { returnTo: window.location.href.substr(window.location.origin.length) } })

    return <div>{'Loading'}</div>
  }

  return (
    <Router history={history}>
      <Container fluid id="app">
        <LeftPanel collapsed={collapsed}>
          <Menu collapsed={collapsed} toggleCollapsed={() => setCollapsed(!collapsed)} />
        </LeftPanel>
        <StatusBar collapsed={collapsed} />
        <div id="content" className={classNames({ collapsed })}>
          <Routes />
          <Footer />
        </div>
      </Container>
    </Router>
  )
}

export default hot(App)
