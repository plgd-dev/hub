import { FC, SyntheticEvent, useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useIntl } from 'react-intl'
import { useDispatch, useSelector } from 'react-redux'

import Header from '@shared-ui/components/Layout/Header'
import NotificationCenter from '@shared-ui/components/Atomic/NotificationCenter'
import UserWidget from '@shared-ui/components/Layout/Header/UserWidget'
import VersionMark from '@shared-ui/components/Atomic/VersionMark'
import { severities } from '@shared-ui/components/Atomic/VersionMark/constants'
import Layout from '@shared-ui/components/Layout'
import { MenuItem } from '@shared-ui/components/Layout/LeftPanel/LeftPanel.types'
import { parseActiveItem } from '@shared-ui/components/Layout/LeftPanel/utils'

import { Props } from './AppLayout.types'
import { mather, menu, Routes } from '@/routes'
import { messages as t } from '@/containers/App/App.i18n'
import { readAllNotifications, setNotifications } from '@/containers/Notifications/slice'
import LeftPanelWrapper from '@/containers/App/AppInner/LeftPanelWrapper/LeftPanelWrapper'
import { CombinatedStoreType } from '@/store/store'

const AppLayout: FC<Props> = (props) => {
    const { collapsed, userData, setCollapsed } = props
    const { formatMessage: _ } = useIntl()
    const location = useLocation()
    const dispatch = useDispatch()
    const navigate = useNavigate()

    const [activeItem, setActiveItem] = useState(parseActiveItem(location.pathname, menu!, mather))
    const notifications = useSelector((state: CombinatedStoreType) => state.notifications)

    const handleItemClick = (item: MenuItem, e: SyntheticEvent) => {
        e.preventDefault()

        setActiveItem(item.id)
        item.link && navigate(item.link)
    }

    const handleLocationChange = (id: string) => {
        id !== activeItem && setActiveItem(id)
    }

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
                    versionMark={<VersionMark severity={severities.SUCCESS} versionText='Version 2.02' />}
                />
            }
        />
    )
}

AppLayout.displayName = 'AppLayout'

export default AppLayout
