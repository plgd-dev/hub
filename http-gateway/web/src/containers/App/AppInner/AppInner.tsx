import { useMemo } from 'react'
import { BrowserRouter } from 'react-router-dom'
import { useAuth } from 'oidc-react'
import { Global } from '@emotion/react'
import { Helmet } from 'react-helmet'

import { InitServices } from '@shared-ui/common/services/init-services'
import { BrowserNotificationsContainer } from '@shared-ui/components/Atomic/Toast'
import { useLocalStorage } from '@shared-ui/common/hooks'
import { clientAppSettings, security } from '@shared-ui/common/services'
import { AppContextType } from '@shared-ui/app/share/AppContext.types'
import AppContext from '@shared-ui/app/share/AppContext'

import appConfig from '@/config'
import AppLoader from '@/containers/App/AppLoader/AppLoader'
import { Props } from './AppInner.types'
import { deviceStatusListener } from '../../Devices/websockets'
import { globalStyle } from './AppInner.global.styles'
import AppLayout from '@/containers/App/AppLayout/AppLayout'

const AppInner = (props: Props) => {
    const { wellKnownConfig, openTelemetry, collapsed, setCollapsed } = props
    const { userData, userManager, signOutRedirect, isLoading } = useAuth()

    const [footerExpanded, setFooterExpanded] = useLocalStorage('footerPanelExpanded', false)

    const toastNotifications = false

    const contextValue: AppContextType = useMemo(
        () => ({
            footerExpanded,
            collapsed,
            setCollapsed,
            setFooterExpanded,
            telemetryWebTracer: openTelemetry.getWebTracer(),
            buildInformation: wellKnownConfig?.buildInfo,
            isHub: true,
        }),
        [footerExpanded, collapsed, setCollapsed, setFooterExpanded, wellKnownConfig, openTelemetry]
    )

    if (!userData || isLoading) {
        return <AppLoader />
    } else {
        security.setAccessToken(userData.access_token)

        // for remote clients
        clientAppSettings.setSignOutRedirect(signOutRedirect)

        if (userManager) {
            security.setUserManager(userManager)
        }
    }

    return (
        <AppContext.Provider value={contextValue}>
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
                <BrowserNotificationsContainer />
            </BrowserRouter>
        </AppContext.Provider>
    )
}

AppInner.displayName = 'AppInner'

export default AppInner
