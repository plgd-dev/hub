import React, { FC, SyntheticEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useIntl } from 'react-intl'
import { useDispatch, useSelector } from 'react-redux'
import isFunction from 'lodash/isFunction'
import { useTheme } from '@emotion/react'
import isEqual from 'lodash/isEqual'

import Header from '@shared-ui/components/Layout/Header'
import NotificationCenter from '@shared-ui/components/Atomic/NotificationCenter'
import UserWidget from '@shared-ui/components/Layout/Header/UserWidget'
import VersionMark from '@shared-ui/components/Atomic/VersionMark'
import Layout from '@shared-ui/components/Layout'
import { MenuItem, SubMenuItem } from '@shared-ui/components/Layout/LeftPanel/LeftPanel.types'
import { parseActiveItem } from '@shared-ui/components/Layout/LeftPanel/utils'
import { getVersionMarkData } from '@shared-ui/components/Atomic/VersionMark/utils'
import { severities } from '@shared-ui/components/Atomic/VersionMark/constants'
import { flushDevices } from '@shared-ui/app/clientApp/Devices/slice'
import { reset } from '@shared-ui/app/clientApp/App/AppRest'
import { App } from '@shared-ui/components/Atomic'
import { ThemeType } from '@shared-ui/components/Atomic/_theme'
import { clientAppSettings, security } from '@shared-ui/common/services'
import { useAppVersion, WellKnownConfigType } from '@shared-ui/common/hooks'
import Logo from '@shared-ui/components/Atomic/Logo'

import { Props } from './AppLayout.types'
import { mather, getMenu, Routes } from '@/routes'
import { messages as t } from '@/containers/App/App.i18n'
import { messages as g } from '../../Global.i18n'
import { readAllNotifications, setNotifications } from '@/containers/Notifications/slice'
import LeftPanelWrapper from '@/containers/App/AppInner/LeftPanelWrapper/LeftPanelWrapper'
import { CombinedStoreType } from '@/store/store'
import { setVersion } from '@/containers/App/slice'
import { deleteAllRemoteClients } from '@/containers/RemoteClients/slice'
import testId from '@/testId'
import PreviewApp from '@/containers/Configuration/PreviewApp/PreviewApp'
import { CONFIGURATION_PAGE_FRAME } from '@/constants'

const AppLayout: FC<Props> = (props) => {
    const { buildInformation, collapsed, mockApp, userData, signOutRedirect, setCollapsed } = props
    const { formatMessage: _ } = useIntl()
    const location = useLocation()
    const dispatch = useDispatch()
    const navigate = useNavigate()
    const configurationPageFrame = window.location.pathname === `/${CONFIGURATION_PAGE_FRAME}`

    const wellKnownConfig = security.getWellKnowConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }

    const menu = useMemo(() => getMenu(wellKnownConfig.ui.visibility.mainSidebar), [wellKnownConfig.ui.visibility.mainSidebar])

    const [activeItem, setActiveItem] = useState(parseActiveItem(location.pathname, menu, mather))
    const notifications = useSelector((state: CombinedStoreType) => state.notifications)
    const appStore = useSelector((state: CombinedStoreType) => state.app)
    const storedRemoteStore = useSelector((state: CombinedStoreType) => state.remoteClients)

    const theme: ThemeType = useTheme()

    const [version] = useAppVersion({ requestedDatetime: appStore.version.requestedDatetime })

    useEffect(() => {
        if (version && !isEqual(appStore.version, version)) {
            dispatch(setVersion(version))
        }
    }, [appStore.version, dispatch, version])

    const handleItemClick = (item: MenuItem | SubMenuItem, e: SyntheticEvent) => {
        e.preventDefault()

        setActiveItem(item.id)
        item.link && navigate(item.link)
    }

    const handleLocationChange = (id: string) => {
        id !== activeItem && setActiveItem(id)
    }

    const versionMarkData = useMemo(
        () =>
            getVersionMarkData({
                buildVersion: buildInformation.version,
                githubVersion: appStore.version.latest || '',
                i18n: {
                    version: _(t.version),
                    newUpdateIsAvailable: _(t.newUpdateIsAvailable),
                },
            }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [appStore.version.latest, buildInformation.version]
    )

    const logout = useCallback(() => {
        if (storedRemoteStore.remoteClients.length && !mockApp) {
            const promises = storedRemoteStore.remoteClients.map((remoteClient) => reset(remoteClient.clientUrl))

            Promise.all(promises)
                .then(() => {})
                .catch((e) => {
                    console.log(e)
                })
                .finally(() => {
                    dispatch(deleteAllRemoteClients())
                    dispatch(flushDevices())
                    isFunction(signOutRedirect) &&
                        signOutRedirect({
                            post_logout_redirect_uri: window.location.origin,
                        })
                })
        } else {
            isFunction(signOutRedirect) &&
                signOutRedirect({
                    post_logout_redirect_uri: window.location.origin,
                })
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [signOutRedirect])

    // reset
    clientAppSettings.setUseToken(true)

    if (configurationPageFrame) {
        return (
            <App toastContainerPortalTarget={document.getElementById('toast-root')}>
                <PreviewApp />
            </App>
        )
    }

    return (
        <App toastContainerPortalTarget={document.getElementById('toast-root')}>
            <Layout
                content={<Routes />}
                header={
                    <Header
                        breadcrumbs={<div id='breadcrumbsPortalTarget'></div>}
                        notificationCenter={
                            <NotificationCenter
                                defaultNotification={notifications}
                                i18n={{
                                    notifications: _(t.notifications),
                                    noNotifications: _(t.noNotifications),
                                    markAllAsRead: _(t.markAllAsRead),
                                }}
                                onNotification={(n: any) => {
                                    dispatch(setNotifications(n))
                                }}
                                readAllNotifications={() => {
                                    dispatch(readAllNotifications())
                                }}
                            />
                        }
                        userWidget={
                            <UserWidget
                                dataTestId={testId.app.logout}
                                description={userData?.profile?.family_name}
                                image={userData?.profile?.picture}
                                logoutTitle={_(g.logOut)}
                                name={userData?.profile?.name ?? ''}
                                onLogout={logout}
                            />
                        }
                    />
                }
                leftPanel={
                    <LeftPanelWrapper
                        activeId={activeItem}
                        collapsed={collapsed}
                        logo={theme.logo && <Logo logo={theme.logo} onClick={() => navigate(`/`)} />}
                        menu={menu}
                        onItemClick={handleItemClick}
                        onLocationChange={handleLocationChange}
                        setCollapsed={setCollapsed}
                        // newFeature={{
                        //     onClick: () => console.log('click'),
                        //     onClose: () => console.log('close'),
                        // }}
                        versionMark={
                            appStore.version.latest && (
                                <VersionMark
                                    severity={versionMarkData.severity}
                                    update={
                                        versionMarkData.severity !== severities.SUCCESS && appStore.version.latest_url
                                            ? {
                                                  text: _(t.clickHere),
                                                  onClick: (e) => {
                                                      e.preventDefault()
                                                      window.open(appStore.version.latest_url, '_blank')
                                                  },
                                              }
                                            : undefined
                                    }
                                    versionText={versionMarkData.text}
                                />
                            )
                        }
                    />
                }
            />
        </App>
    )
}

AppLayout.displayName = 'AppLayout'

export default AppLayout
