import { SyntheticEvent, useMemo, useState } from 'react'
import { AppContext } from '@/containers/App/AppContext'
import { Router } from 'react-router-dom'
import { history } from '@/store'
import { InitServices } from '@shared-ui/common/services/init-services'
import { Helmet } from 'react-helmet'
import appConfig from '@/config'
import { menu, Routes } from '@/routes'
import Footer from '@shared-ui/components/new/Layout/Footer'
import { BrowserNotificationsContainer, ToastContainer } from '@shared-ui/components/new/Toast'
import { useLocalStorage, WellKnownConfigType } from '@shared-ui/common/hooks'
import { useAuth } from 'oidc-react'
import { security } from '@shared-ui/common/services'
import AppLoader from '@/containers/App/AppLoader/AppLoader'
import { Props } from './AppInner.types'
import { deviceStatusListener } from '../../Devices/websockets'
import Layout from '@shared-ui/components/new/Layout'
import Header from '@shared-ui/components/new/Layout/Header'
import UserWidget from '@shared-ui/components/new/Layout/Header/UserWidget'
import LeftPanel from '@shared-ui/components/new/Layout/LeftPanel'
import VersionMark from '@shared-ui/components/new/VersionMark'
import { severities } from '@shared-ui/components/new/VersionMark/constants'
import { ThemeProvider } from '@emotion/react'
import light from '@shared-ui/components/new/_theme/light'
import { MenuItem } from '@shared-ui/components/new/Layout/LeftPanel/LeftPanel.types'

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
    const [collapsed, setCollapsed] = useLocalStorage('leftPanelCollapsed', true)
    const [footerExpanded, setFooterExpanded] = useLocalStorage('footerPanelExpanded', false)
    const [activeItem, setActiveItem] = useState('1')

    const contextValue = useMemo(
        () => ({
            collapsed,
            footerExpanded,
            setFooterExpanded,
            ...wellKnownConfig,
            wellKnownConfig,
            telemetryWebTracer: openTelemetry.getWebTracer(),
            buildInformation: buildInformation || undefined,
        }),
        [collapsed, footerExpanded, wellKnownConfig, openTelemetry, buildInformation, setFooterExpanded]
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

    return (
        <AppContext.Provider value={contextValue}>
            <ThemeProvider theme={light}>
                <Router history={history}>
                    <InitServices deviceStatusListener={deviceStatusListener} />
                    <Helmet defaultTitle={appConfig.appName} titleTemplate={`%s | ${appConfig.appName}`} />
                    <Layout
                        collapsedMenu={collapsed}
                        content={
                            <div id='content'>
                                <Routes />
                            </div>
                        }
                        footer={
                            <Footer
                                footerExpanded={footerExpanded}
                                paginationComponent={<div id='paginationPortalTarget'></div>}
                                recentTasksPortal={<div id='recentTasksPortalTarget'></div>}
                                recentTasksPortalTitle={<span id='recentTasksPortalTitleTarget'></span>}
                                setFooterExpanded={setFooterExpanded}
                            />
                        }
                        header={
                            <Header
                                breadcrumbs={<div id='breadcrumbsPortalTarget'></div>}
                                onCollapseToggle={() => setCollapsed(!collapsed)}
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
                            <LeftPanel
                                activeId={activeItem}
                                collapsed={collapsed}
                                menu={menu}
                                newFeature={{
                                    onClick: () => console.log('click'),
                                    onClose: () => console.log('close'),
                                }}
                                onItemClick={handleItemClick}
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
