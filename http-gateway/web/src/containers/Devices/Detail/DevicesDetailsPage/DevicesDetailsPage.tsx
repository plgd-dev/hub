import React, { useContext, useEffect, useState } from 'react'
import ReactDOM from 'react-dom'
import { useIntl } from 'react-intl'
import { useParams } from 'react-router-dom'

import NotFoundPage from '@shared-ui/components/templates/NotFoundPage'
import { useIsMounted, WellKnownConfigType } from '@shared-ui/common/hooks'
import { messages as menuT } from '@shared-ui/components/new/Menu/Menu.i18n'
import PageLayout from '@shared-ui/components/new/PageLayout'
import Tabs from '@shared-ui/components/new/Tabs'
import Breadcrumbs from '@plgd/shared-ui/src/components/new/Layout/Header/Breadcrumbs'
import StatusTag from '@shared-ui/components/new/StatusTag'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { security } from '@shared-ui/common/services'
import { showErrorToast } from '@shared-ui/components/new'
import NotificationMessage from '@shared-ui/components/new/NotificationMessage'
import Footer from '@plgd/shared-ui/src/components/new/Layout/Footer'

import DevicesDetailsHeader from '../DevicesDetailsHeader'
import { devicesStatuses, NO_DEVICE_NAME } from '../../constants'
import { getDeviceChangeResourceHref, handleTwinSynchronizationErrors, isDeviceOnline } from '../../utils'
import { updateDevicesResourceApi, updateDeviceTwinSynchronizationApi } from '../../rest'
import { useDeviceDetails, useDevicesResources } from '../../hooks'
import { messages as t } from '../../Devices.i18n'
import './DevicesDetailsPage.scss'
import Tab1 from './Tabs/Tab1'
import Tab2 from '@/containers/Devices/Detail/DevicesDetailsPage/Tabs/Tab2'
import { PendingCommandsExpandableList } from '@/containers/PendingCommands'
import EditNameModal from '@/containers/Devices/Detail/EditNameModal/EditNameModal'
import isFunction from 'lodash/isFunction'
import { AppContext } from '@/containers/App/AppContext'

const DevicesDetailsPage = () => {
    const { formatMessage: _ } = useIntl()
    const {
        id,
    }: {
        id: string
    } = useParams()
    const [domReady, setDomReady] = useState(false)
    const [activeTabItem, setActiveTabItem] = useState(0)
    const [notificationMessage, setNotificationMessage] = useState<string | undefined>(undefined)
    const [twinSyncLoading, setTwinSyncLoading] = useState(false)

    const isMounted = useIsMounted()
    const { data, updateData, loading, error: deviceError } = useDeviceDetails(id)
    const { data: resourcesData, loading: loadingResources, error: resourcesError } = useDevicesResources(id)

    const wellKnownConfig = security.getWellKnowConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }
    const [ttl] = useState(wellKnownConfig?.defaultCommandTimeToLive || 0)

    const [isTwinEnabled, setIsTwinEnabled] = useState<boolean>(data?.metadata?.twinEnabled || false)
    const [showEditNameModal, setShowEditNameModal] = useState(false)
    const [deviceNameLoading, setDeviceNameLoading] = useState(false)

    const { footerExpanded, setFooterExpanded } = useContext(AppContext)

    useEffect(() => {
        setDomReady(true)
    }, [])

    useEffect(() => {
        if (data?.metadata?.twinEnabled && data?.metadata?.twinEnabled !== isTwinEnabled) {
            setIsTwinEnabled(data?.metadata?.twinEnabled)
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [data, loading])

    if (deviceError) {
        return <NotFoundPage message={_(t.deviceNotFoundMessage, { id })} title={_(t.deviceNotFound)} />
    }

    if (resourcesError) {
        return <NotFoundPage message={_(t.deviceResourcesNotFoundMessage, { id })} title={_(t.deviceResourcesNotFound)} />
    }

    const resources = resourcesData?.[0]?.resources || []
    const deviceStatus = data?.metadata?.connection?.status
    const isOnline = isDeviceOnline(data)
    const isUnregistered = devicesStatuses.UNREGISTERED === deviceStatus
    const deviceName = data?.name || NO_DEVICE_NAME
    const breadcrumbs = [
        {
            to: '/',
            label: _(menuT.devices),
        },
    ]

    if (deviceName) {
        breadcrumbs.push({ label: deviceName, to: '#' })
    }

    // Handler for setting the twin synchronization on a device
    const setTwinSynchronization = async (newTwinEnabled: boolean) => {
        setTwinSyncLoading(true)
        setNotificationMessage(_(t.twinUpdateMessage))

        try {
            await updateDeviceTwinSynchronizationApi(id, newTwinEnabled)

            if (isMounted.current) {
                setTwinSyncLoading(false)
                setIsTwinEnabled(newTwinEnabled)
            }
        } catch (error) {
            if (error && isMounted.current) {
                handleTwinSynchronizationErrors(error, _)
                setTwinSyncLoading(false)
            }
        }
    }

    // Update the device name in the data object
    const updateDeviceNameInData = (name: string) => {
        updateData({
            ...data,
            name,
        })
        setShowEditNameModal(false)
    }

    const updateDeviceName = async (name: string) => {
        if (name.trim() !== '' && name !== deviceName) {
            const href = getDeviceChangeResourceHref(resources)

            setDeviceNameLoading(true)

            try {
                const { data } = await updateDevicesResourceApi(
                    { deviceId: id, href: href!, ttl },
                    {
                        n: name,
                    }
                )

                if (isMounted.current) {
                    setDeviceNameLoading(false)
                    updateDeviceNameInData(data?.n || name)
                }
            } catch (error) {
                if (error && isMounted.current) {
                    showErrorToast({
                        title: _(t.deviceNameChangeFailed),
                        message: getApiErrorMessage(error),
                    })
                    setDeviceNameLoading(false)
                    setShowEditNameModal(false)
                }
            }
        } else {
            setDeviceNameLoading(false)
            setShowEditNameModal(false)
        }
    }

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            footer={
                <Footer
                    footerExpanded={footerExpanded}
                    paginationComponent={<div id='paginationPortalTarget'></div>}
                    recentTasksPortal={<div id='recentTasksPortalTarget'></div>}
                    recentTasksPortalTitle={
                        <span
                            id='recentTasksPortalTitleTarget'
                            onClick={() => {
                                isFunction(setFooterExpanded) && setFooterExpanded(!footerExpanded)
                            }}
                        >
                            {_(t.recentTasks)}
                        </span>
                    }
                    setFooterExpanded={setFooterExpanded!}
                />
            }
            header={
                <DevicesDetailsHeader
                    deviceId={id}
                    deviceName={deviceName}
                    handleOpenEditDeviceNameModal={() => setShowEditNameModal(true)}
                    isOnline={isOnline}
                    isUnregistered={isUnregistered}
                    links={resources}
                />
            }
            headlineStatusTag={<StatusTag variant={isOnline ? 'success' : 'error'}>{isOnline ? _(t.online) : _(t.offline)}</StatusTag>}
            loading={loading || twinSyncLoading}
            title={deviceName}
        >
            {domReady &&
                ReactDOM.createPortal(
                    <Breadcrumbs items={[{ label: _(menuT.devices), link: '/' }, { label: deviceName }]} />,
                    document.querySelector('#breadcrumbsPortalTarget') as Element
                )}

            <Tabs
                fullHeight={true}
                onItemChange={(i) => setActiveTabItem(i)}
                tabs={[
                    {
                        name: _(t.deviceInformation),
                        content: (
                            <Tab1
                                deviceId={id}
                                deviceName={deviceName}
                                isActiveTab={activeTabItem === 0}
                                isTwinEnabled={isTwinEnabled}
                                setTwinSynchronization={setTwinSynchronization}
                                twinSyncLoading={twinSyncLoading}
                                types={data?.types}
                            />
                        ),
                    },
                    {
                        name: _(t.resources),
                        content: (
                            <Tab2
                                deviceName={deviceName}
                                deviceStatus={deviceStatus}
                                isActiveTab={activeTabItem === 1}
                                isOnline={isOnline}
                                isUnregistered={isUnregistered}
                                loading={loading}
                                loadingResources={loadingResources}
                                resourcesData={resourcesData}
                            />
                        ),
                    },
                ]}
            />

            <NotificationMessage message={notificationMessage} onExit={() => setNotificationMessage(undefined)} show={!!notificationMessage} />

            <EditNameModal
                deviceName={deviceName}
                deviceNameLoading={deviceNameLoading}
                handleClose={() => setShowEditNameModal(false)}
                handleSubmit={updateDeviceName}
                show={showEditNameModal}
            />

            <PendingCommandsExpandableList deviceId={id} />
        </PageLayout>
    )
}

DevicesDetailsPage.displayName = 'DevicesDetailsPage'

export default DevicesDetailsPage
