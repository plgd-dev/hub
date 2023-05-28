import { SyntheticEvent, useMemo, useState } from 'react'
import { Router } from 'react-router-dom'
import { useAuth } from 'oidc-react'
import { Global, ThemeProvider } from '@emotion/react'
import { Helmet } from 'react-helmet'

import Layout from '@shared-ui/components/Layout'
import Header from '@shared-ui/components/Layout/Header'
import UserWidget from '@shared-ui/components/Layout/Header/UserWidget'
import { parseActiveItem } from '@shared-ui/components/Layout/LeftPanel'
import VersionMark from '@shared-ui/components/Atomic/VersionMark'
import { severities } from '@shared-ui/components/Atomic/VersionMark/constants'
import { InitServices } from '@shared-ui/common/services/init-services'
import { BrowserNotificationsContainer } from '@shared-ui/components/Atomic/Toast'
import { ToastContainer } from '@shared-ui/components/Atomic/Notification'
import { useLocalStorage, WellKnownConfigType } from '@shared-ui/common/hooks'
import light from '@shared-ui/components/Atomic/_theme/light'
import { MenuItem } from '@shared-ui/components/Layout/LeftPanel/LeftPanel.types'
import { security } from '@shared-ui/common/services'

import { AppContext } from '@/containers/App/AppContext'
import { history } from '@/store'
import appConfig from '@/config'
import { mather, menu, Routes } from '@/routes'
import AppLoader from '@/containers/App/AppLoader/AppLoader'
import { Props } from './AppInner.types'
import { deviceStatusListener } from '../../Devices/websockets'
import LeftPanelWrapper from '@/containers/App/AppInner/LeftPanelWrapper/LeftPanelWrapper'
import { globalStyle } from './AppInner.global.styles'
import { AppContextType } from '@/containers/App/AppContext.types'

const getBuildInformation = (wellKnownConfig: WellKnownConfigType) => ({
    buildDate: wellKnownConfig?.buildDate || '',
    commitHash: wellKnownConfig?.commitHash || '',
    commitDate: wellKnownConfig?.commitDate || '',
    releaseUrl: wellKnownConfig?.releaseUrl || '',
    version: wellKnownConfig?.version || '',
})

const AppInner = (props: Props) => {
    const { wellKnownConfig, openTelemetry } = props
    const { userData, userManager } = useAuth()
    const buildInformation = getBuildInformation(wellKnownConfig)

    const [footerExpanded, setFooterExpanded] = useLocalStorage('footerPanelExpanded', false)
    const [activeItem, setActiveItem] = useState(parseActiveItem(history.location.pathname, menu, mather))
    const [collapsed, setCollapsed] = useLocalStorage('leftPanelCollapsed', true)

    // TODO: redux store on user switch Notification
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
            buildInformation: buildInformation || undefined,
        }),
        [footerExpanded, collapsed, setCollapsed, setFooterExpanded, wellKnownConfig, openTelemetry, buildInformation]
    )

    if (userData) {
        security.setAccessToken(userData.access_token)

        if (userManager) {
            security.setUserManager(userManager)
        }
    } else {
        return <AppLoader />
    }

    const handleItemClick = (item: MenuItem, e: SyntheticEvent) => {
        e.preventDefault()

        setActiveItem(item.id)
        history.push(item.link)
    }

    const handleLocationChange = (id: string) => {
        id !== activeItem && setActiveItem(id)
    }

    return (
        <AppContext.Provider value={contextValue}>
            <ThemeProvider theme={light}>
                <Router history={history}>
                    <InitServices deviceStatusListener={deviceStatusListener} />
                    <Helmet defaultTitle={appConfig.appName} titleTemplate={`%s | ${appConfig.appName}`} />
                    <Layout
                        content={<Routes />}
                        header={
                            <Header
                                breadcrumbs={<div id='breadcrumbsPortalTarget'></div>}
                                userWidget={
                                    <UserWidget
                                        description={userData?.profile?.family_name}
                                        image={userData?.profile?.picture}
                                        name={userData?.profile?.name || ''}
                                    />
                                }
                            />
                        }
                        leftPanel={
                            <LeftPanelWrapper
                                activeId={activeItem}
                                collapsed={collapsed}
                                menu={menu}
                                onItemClick={handleItemClick}
                                onLocationChange={handleLocationChange}
                                setCollapsed={setCollapsed}
                                // newFeature={{
                                //     onClick: () => console.log('click'),
                                //     onClose: () => console.log('close'),
                                // }}
                                versionMark={<VersionMark severity={severities.SUCCESS} versionText='Version 2.02' />}
                            />
                        }
                    />
                    <Global styles={globalStyle(toastNotifications)} />

                    <ToastContainer portalTarget={document.getElementById('toast-root')} showNotifications={true} />

                    <BrowserNotificationsContainer />
                </Router>
            </ThemeProvider>
        </AppContext.Provider>
    )
}

AppInner.displayName = 'AppInner'

export default AppInner
