import { useMemo } from 'react'
import { BrowserRouter } from 'react-router-dom'
import { useAuth } from 'oidc-react'
import { Global, ThemeProvider } from '@emotion/react'
import { Helmet } from 'react-helmet'

import { InitServices } from '@shared-ui/common/services/init-services'
import { BrowserNotificationsContainer } from '@shared-ui/components/Atomic/Toast'
import { ToastContainer } from '@shared-ui/components/Atomic/Notification'
import { useLocalStorage } from '@shared-ui/common/hooks'
import light from '@shared-ui/components/Atomic/_theme/light'
import { clientAppSetings, security } from '@shared-ui/common/services'

import { AppContext } from '@/containers/App/AppContext'
import appConfig from '@/config'
import AppLoader from '@/containers/App/AppLoader/AppLoader'
import { Props } from './AppInner.types'
import { deviceStatusListener } from '../../Devices/websockets'
import { globalStyle } from './AppInner.global.styles'
import { AppContextType } from '@/containers/App/AppContext.types'
import AppLayout from '@/containers/App/AppLayout/AppLayout'

const AppInner = (props: Props) => {
    const { wellKnownConfig, openTelemetry } = props
    const { userData, userManager, signOutRedirect } = useAuth()

    const [footerExpanded, setFooterExpanded] = useLocalStorage('footerPanelExpanded', false)
    const [collapsed, setCollapsed] = useLocalStorage('leftPanelCollapsed', true)

    const toastNotifications = false

    const contextValue: AppContextType = useMemo(
        () => ({
            footerExpanded,
            collapsed,
            setCollapsed,
            setFooterExpanded,
            ...wellKnownConfig,
            wellKnownConfig,
            telemetryWebTracer: openTelemetry.getWebTracer(),
            buildInformation: wellKnownConfig?.buildInfo,
        }),
        [footerExpanded, collapsed, setCollapsed, setFooterExpanded, wellKnownConfig, openTelemetry]
    )

    if (userData) {
        security.setAccessToken(userData.access_token)

        // for remote clients
        clientAppSetings.setUserData(userData)
        clientAppSetings.setSignOutRedirect(signOutRedirect)

        if (userManager) {
            security.setUserManager(userManager)
        }
    } else {
        return <AppLoader />
    }

    return (
        <AppContext.Provider value={contextValue}>
            <ThemeProvider theme={light}>
                <InitServices deviceStatusListener={deviceStatusListener} />
                <Helmet defaultTitle={appConfig.appName} titleTemplate={`%s | ${appConfig.appName}`} />
                <BrowserRouter>
                    <AppLayout
                        buildInformation={wellKnownConfig?.buildInfo}
                        collapsed={collapsed}
                        setCollapsed={setCollapsed}
                        signOutRedirect={signOutRedirect}
                        userData={userData}
                    />
                    <Global styles={globalStyle(toastNotifications)} />
                    <ToastContainer portalTarget={document.getElementById('toast-root')} showNotifications={true} />
                    <BrowserNotificationsContainer />
                </BrowserRouter>
            </ThemeProvider>
        </AppContext.Provider>
    )
}

AppInner.displayName = 'AppInner'

export default AppInner
