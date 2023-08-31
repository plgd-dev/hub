import React, { FC, useCallback, useContext, useEffect, useState } from 'react'
import ReactDOM from 'react-dom'
import { useIntl } from 'react-intl'
import { useNavigate, useParams } from 'react-router-dom'
import isFunction from 'lodash/isFunction'

import NotFoundPage from '@shared-ui/components/Templates/NotFoundPage'
import { useIsMounted, WellKnownConfigType } from '@shared-ui/common/hooks'
import { messages as menuT } from '@shared-ui/components/Atomic/Menu/Menu.i18n'
import PageLayout from '@shared-ui/components/Atomic/PageLayout'
import Tabs from '@shared-ui/components/Atomic/Tabs'
import Breadcrumbs from '@shared-ui/components/Layout/Header/Breadcrumbs'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { clientAppSettings, security } from '@shared-ui/common/services'
import Footer from '@shared-ui/components/Layout/Footer'
import EditDeviceNameModal from '@shared-ui/components/Organisms/EditDeviceNameModal'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import DevicesDetailsHeader from '../DevicesDetailsHeader'
import { devicesStatuses, NO_DEVICE_NAME } from '../../constants'
import { getDeviceChangeResourceHref, handleTwinSynchronizationErrors, isDeviceOnline } from '../../utils'
import { updateDevicesResourceApi, updateDeviceTwinSynchronizationApi } from '../../rest'
import { useDeviceDetails, useDevicePendingCommands, useDevicesResources, useDeviceSoftwareUpdateDetails } from '../../hooks'
import { messages as t } from '../../Devices.i18n'
import './DevicesDetailsPage.scss'
import Tab1 from './Tabs/Tab1'
import Tab2 from './Tabs/Tab2'
import { PendingCommandsExpandableList } from '@/containers/PendingCommands'
import { AppContext } from '@/containers/App/AppContext'
import { Props } from './DevicesDetailsPage.types'
import notificationId from '@/notificationId'
import testId from '@/testId'

const DevicesDetailsPage: FC<Props> = (props) => {
    const { defaultActiveTab } = props
    const { formatMessage: _ } = useIntl()
    const { id: routerId } = useParams()
    const navigate = useNavigate()
    const id = routerId || ''

    const [domReady, setDomReady] = useState(false)
    const [activeTabItem, setActiveTabItem] = useState(defaultActiveTab ?? 0)
    const [twinSyncLoading, setTwinSyncLoading] = useState(false)

    const isMounted = useIsMounted()
    const { data, updateData, loading, error: deviceError } = useDeviceDetails(id)
    const { data: softwareUpdateData, refresh: refreshSoftwareUpdate } = useDeviceSoftwareUpdateDetails(id)
    const { data: resourcesData, loading: loadingResources, error: resourcesError, refresh } = useDevicesResources(id)
    const { data: pendingCommandsData, refresh: refreshPendingCommands } = useDevicePendingCommands(id)

    const wellKnownConfig = security.getWellKnowConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }
    const [ttl] = useState(wellKnownConfig?.defaultCommandTimeToLive || 0)

    const [isTwinEnabled, setIsTwinEnabled] = useState<boolean>(data?.metadata?.twinEnabled ?? false)
    const [showEditNameModal, setShowEditNameModal] = useState(false)
    const [deviceNameLoading, setDeviceNameLoading] = useState(false)

    const { footerExpanded, setFooterExpanded } = useContext(AppContext)

    clientAppSettings.reset()

    useEffect(() => {
        setDomReady(true)
    }, [])

    useEffect(() => {
        if (data?.metadata?.twinEnabled !== isTwinEnabled) {
            setIsTwinEnabled(data?.metadata?.twinEnabled ?? false)
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [data, loading])

    const refreshResources = useCallback(() => {
        refresh()
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const handleOpenEditDeviceNameModal = useCallback(() => {
        setShowEditNameModal(true)
    }, [])

    const handleTabChange = useCallback((i: number) => {
        setActiveTabItem(i)

        navigate(`/devices/${id}${i === 1 ? '/resources' : ''}`, { replace: true })

        refreshPendingCommands()
        refreshSoftwareUpdate()
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

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
                    Notification.error(
                        { title: _(t.deviceNameChangeFailed), message: getApiErrorMessage(error) },
                        { notificationId: notificationId.HUB_DEVICES_DETAILS_PAGE_UPDATE_DEVICE_NAME }
                    )
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
            dataTestId={testId.devices.detail.layout}
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
                    setFooterExpanded={setFooterExpanded}
                />
            }
            header={
                <DevicesDetailsHeader
                    deviceId={id}
                    deviceName={deviceName}
                    handleOpenEditDeviceNameModal={handleOpenEditDeviceNameModal}
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
                activeItem={activeTabItem}
                fullHeight={true}
                onItemChange={handleTabChange}
                tabs={[
                    {
                        name: _(t.deviceInformation),
                        id: 0,
                        dataTestId: testId.devices.detail.tabInformation,
                        content: (
                            <Tab1
                                deviceId={id}
                                deviceName={deviceName}
                                firmware={data?.data?.content?.sv}
                                isActiveTab={activeTabItem === 0}
                                isTwinEnabled={isTwinEnabled}
                                model={data?.data?.content?.dmno}
                                pendingCommandsData={pendingCommandsData}
                                setTwinSynchronization={setTwinSynchronization}
                                softwareUpdateData={softwareUpdateData?.result?.data?.content}
                                twinSyncLoading={twinSyncLoading}
                                types={data?.types}
                            />
                        ),
                    },
                    {
                        name: _(t.resources),
                        id: 1,
                        dataTestId: testId.devices.detail.tabResources,
                        content: (
                            <Tab2
                                deviceName={deviceName}
                                deviceStatus={deviceStatus}
                                isActiveTab={activeTabItem === 1}
                                isOnline={isOnline}
                                isUnregistered={isUnregistered}
                                loading={loading}
                                loadingResources={loadingResources}
                                refreshResources={refreshResources}
                                resourcesData={resourcesData}
                            />
                        ),
                    },
                ]}
            />

            <EditDeviceNameModal
                dataTestId={testId.devices.detail.editNameModal}
                deviceName={deviceName}
                deviceNameLoading={deviceNameLoading}
                handleClose={() => setShowEditNameModal(false)}
                handleSubmit={updateDeviceName}
                i18n={{
                    close: _(t.close),
                    deviceName: _(t.deviceName),
                    edit: _(t.edit),
                    name: _(t.name),
                    reset: _(t.reset),
                    saveChange: _(t.saveChange),
                    savingChanges: _(t.savingChanges),
                }}
                show={showEditNameModal}
            />

            <PendingCommandsExpandableList deviceId={id} />
        </PageLayout>
    )
}

DevicesDetailsPage.displayName = 'DevicesDetailsPage'

export default DevicesDetailsPage
