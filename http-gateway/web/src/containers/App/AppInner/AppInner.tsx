import { SyntheticEvent, useMemo, useState } from 'react'
import { Router } from 'react-router-dom'
import { useAuth } from 'oidc-react'
import { ThemeProvider } from '@emotion/react'
import { Helmet } from 'react-helmet'

import Layout from '@shared-ui/components/new/Layout'
import Header from '@shared-ui/components/new/Layout/Header'
import UserWidget from '@shared-ui/components/new/Layout/Header/UserWidget'
import { parseActiveItem } from '@shared-ui/components/new/Layout/LeftPanel'
import VersionMark from '@shared-ui/components/new/VersionMark'
import { severities } from '@shared-ui/components/new/VersionMark/constants'
import { InitServices } from '@shared-ui/common/services/init-services'
import { BrowserNotificationsContainer, ToastContainer } from '@shared-ui/components/new/Toast'
import { useLocalStorage, WellKnownConfigType } from '@shared-ui/common/hooks'
import light from '@shared-ui/components/new/_theme/light'
import { MenuItem } from '@shared-ui/components/new/Layout/LeftPanel/LeftPanel.types'
import { security } from '@shared-ui/common/services'

import { AppContext } from '@/containers/App/AppContext'
import { history } from '@/store'
import appConfig from '@/config'
import { mather, menu, Routes } from '@/routes'
import AppLoader from '@/containers/App/AppLoader/AppLoader'
import { Props } from './AppInner.types'
import { deviceStatusListener } from '../../Devices/websockets'
import LeftPanelWrapper from '@/containers/App/AppInner/LeftPanelWrapper/LeftPanelWrapper'

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

    const contextValue = useMemo(
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
                    <ToastContainer />
                    <BrowserNotificationsContainer />
                </Router>
            </ThemeProvider>
        </AppContext.Provider>
    )
}

AppInner.displayName = 'AppInner'

export default AppInner
