import { useCallback, useMemo } from 'react'
import { BrowserRouter } from 'react-router-dom'
import { useAuth } from 'oidc-react'
import { Global } from '@emotion/react'

import { InitServices } from '@shared-ui/common/services/init-services'
import { BrowserNotificationsContainer } from '@shared-ui/components/Atomic/Toast'
import { useLocalStorage, WellKnownConfigType } from '@shared-ui/common/hooks'
import { clientAppSettings, security } from '@shared-ui/common/services'
import { AppContextType } from '@shared-ui/app/share/AppContext.types'
import AppContext from '@shared-ui/app/share/AppContext'
import { useDocumentTitle } from 'usehooks-ts'

import AppLoader from '@/containers/App/AppLoader/AppLoader'
import { Props } from './AppInner.types'
import { deviceStatusListener } from '../../Devices/websockets'
import { globalStyle } from './AppInner.global.styles'
import AppLayout from '@/containers/App/AppLayout/AppLayout'
import appConfig from '@/config'
import isFunction from 'lodash/isFunction'

const AppInner = (props: Props) => {
    const { wellKnownConfig, openTelemetry, collapsed, setCollapsed } = props
    const { userData, userManager, signOutRedirect, isLoading } = useAuth()

    const [footerExpanded, setFooterExpanded] = useLocalStorage('footerPanelExpanded', false)

    const toastNotifications = false

    const unauthorizedCallback = useCallback(() => {
        isFunction(signOutRedirect) &&
            signOutRedirect({
                post_logout_redirect_uri: window.location.origin,
            })
    }, [signOutRedirect])

    const contextValue: AppContextType = useMemo(
        () => ({
            footerExpanded,
            collapsed,
            setCollapsed,
            setFooterExpanded,
            telemetryWebTracer: openTelemetry.getWebTracer(),
            buildInformation: wellKnownConfig?.buildInfo,
            isHub: true,
            unauthorizedCallback,
        }),
        [footerExpanded, collapsed, setCollapsed, setFooterExpanded, openTelemetry, wellKnownConfig?.buildInfo, unauthorizedCallback]
    )

    useDocumentTitle(appConfig.appName)

    if (!userData || isLoading) {
        return <AppLoader />
    } else {
        security.setAccessToken(userData.access_token)

        // for remote clients
        clientAppSettings.setSignOutRedirect(signOutRedirect)

        if (userManager) {
            security.setUserManager(userManager)
        }

        const wellKnownConfig = security.getWellKnownConfig() as WellKnownConfigType & {
            defaultCommandTimeToLive: number
        }

        security.setWellKnownConfig({ ...wellKnownConfig, unauthorizedCallback })
    }

    return (
        <AppContext.Provider value={contextValue}>
            <InitServices deviceStatusListener={deviceStatusListener} />
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
