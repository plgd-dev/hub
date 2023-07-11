import { FC, SyntheticEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useIntl } from 'react-intl'
import { useDispatch, useSelector } from 'react-redux'

import Header from '@shared-ui/components/Layout/Header'
import NotificationCenter from '@shared-ui/components/Atomic/NotificationCenter'
import UserWidget from '@shared-ui/components/Layout/Header/UserWidget'
import VersionMark from '@shared-ui/components/Atomic/VersionMark'
import Layout from '@shared-ui/components/Layout'
import { MenuItem } from '@shared-ui/components/Layout/LeftPanel/LeftPanel.types'
import { parseActiveItem } from '@shared-ui/components/Layout/LeftPanel/utils'
import { getMinutesBetweenDates } from '@shared-ui/common/utils'
import { getVersionMarkData } from '@shared-ui/components/Atomic/VersionMark/utils'
import { severities } from '@shared-ui/components/Atomic/VersionMark/constants'

import { Props } from './AppLayout.types'
import { mather, menu, Routes } from '@/routes'
import { messages as t } from '@/containers/App/App.i18n'
import { readAllNotifications, setNotifications } from '@/containers/Notifications/slice'
import LeftPanelWrapper from '@/containers/App/AppInner/LeftPanelWrapper/LeftPanelWrapper'
import { CombinedStoreType } from '@/store/store'
import { setVersion } from '@/containers/App/slice'
import { getVersionNumberFromGithub } from '@/containers/App/AppRest'
import { GITHUB_VERSION_REQUEST_INTERVAL } from '@/constants'

const AppLayout: FC<Props> = (props) => {
    const { buildInformation, collapsed, userData, setCollapsed } = props
    const { formatMessage: _ } = useIntl()
    const location = useLocation()
    const dispatch = useDispatch()
    const navigate = useNavigate()

    const [activeItem, setActiveItem] = useState(parseActiveItem(location.pathname, menu, mather))
    const notifications = useSelector((state: CombinedStoreType) => state.notifications)
    const appStore = useSelector((state: CombinedStoreType) => state.app)

    const requestVersion = useCallback((now: Date) => {
        getVersionNumberFromGithub().then((ret) => {
            dispatch(
                setVersion({
                    requestedDatetime: now,
                    latest: ret.data.tag_name.replace('v', ''),
                    latest_url: ret.data.html_url,
                })
            )
        })

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    useEffect(() => {
        const now: Date = new Date()

        if (
            !appStore.version.requestedDatetime ||
            getMinutesBetweenDates(new Date(appStore.version.requestedDatetime), now) > GITHUB_VERSION_REQUEST_INTERVAL
        ) {
            requestVersion(now)
        }

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const handleItemClick = (item: MenuItem, e: SyntheticEvent) => {
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

    return (
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
                        <UserWidget description={userData?.profile?.family_name} image={userData?.profile?.picture} name={userData?.profile?.name || ''} />
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
    )
}

AppLayout.displayName = 'AppLayout'

export default AppLayout
