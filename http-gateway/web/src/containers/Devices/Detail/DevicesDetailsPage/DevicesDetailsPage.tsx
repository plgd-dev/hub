import React, { FC, lazy, useCallback, useEffect, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'
import isEmpty from 'lodash/isEmpty'

import NotFoundPage from '@shared-ui/components/Templates/NotFoundPage'
import { useIsMounted, WellKnownConfigType } from '@shared-ui/common/hooks'
import { messages as menuT } from '@shared-ui/components/Atomic/Menu/Menu.i18n'
import Tabs from '@shared-ui/components/Atomic/Tabs'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { clientAppSettings, security } from '@shared-ui/common/services'
import EditNameModal from '@shared-ui/components/Organisms/EditNameModal'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { BreadcrumbItem } from '@shared-ui/components/Layout/Header/Breadcrumbs/Breadcrumbs.types'

import DevicesDetailsHeader from '../DevicesDetailsHeader'
import { devicesStatuses, NO_DEVICE_NAME } from '../../constants'
import { getDeviceChangeResourceHref, handleTwinSynchronizationErrors, isDeviceOnline } from '../../utils'
import { updateDevicesResourceApi, updateDeviceTwinSynchronizationApi } from '../../rest'
import {
    useDeviceDetails,
    useDevicePendingCommands,
    useDevicesResources,
    useDeviceSoftwareUpdateDetails,
    useDeviceCertificates,
    useDeviceProvisioningRecord,
} from '../../hooks'
import { messages as t } from '../../Devices.i18n'
import './DevicesDetailsPage.scss'
import { Props } from './DevicesDetailsPage.types'
import notificationId from '@/notificationId'
import testId from '@/testId'
import PageLayout from '@/containers/Common/PageLayout'
import { pages } from '@/routes'

const Tab1 = lazy(() => import('./Tabs/Tab1/Tab1'))
const Tab2 = lazy(() => import('./Tabs/Tab2/Tab2'))
const Tab3 = lazy(() => import('./Tabs/Tab3/Tab3'))
const Tab4 = lazy(() => import('./Tabs/Tab4/Tab4'))

const DevicesDetailsPage: FC<Props> = (props) => {
    const { defaultActiveTab } = props
    const { formatMessage: _ } = useIntl()
    const { id: routerId } = useParams()
    const navigate = useNavigate()
    const id = routerId || ''

    const [activeTabItem, setActiveTabItem] = useState(defaultActiveTab ?? 0)
    const [twinSyncLoading, setTwinSyncLoading] = useState(false)

    const isMounted = useIsMounted()
    const { data, updateData, loading, error: deviceError } = useDeviceDetails(id)
    const { data: softwareUpdateData, refresh: refreshSoftwareUpdate } = useDeviceSoftwareUpdateDetails(id)
    const { data: resourcesData, loading: loadingResources, error: resourcesError, refresh } = useDevicesResources(id)
    const { data: pendingCommandsData, refresh: refreshPendingCommands, loading: pendingCommandsLoading } = useDevicePendingCommands(id)
    const { data: certificates, loading: certificatesLoading, refresh: certificateRefresh } = useDeviceCertificates(id)
    const { data: provisioningRecords, loading: provisioningRecordsLoading } = useDeviceProvisioningRecord(id)

    const wellKnownConfig = security.getWellKnownConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }
    const [ttl] = useState(wellKnownConfig?.defaultCommandTimeToLive || 0)

    const [isTwinEnabled, setIsTwinEnabled] = useState<boolean>(data?.metadata?.twinEnabled ?? false)
    const [showEditNameModal, setShowEditNameModal] = useState(false)
    const [deviceNameLoading, setDeviceNameLoading] = useState(false)

    clientAppSettings.reset()

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

    const handleTabChange = useCallback(
        (i: number) => {
            setActiveTabItem(i)

            navigate(generatePath(pages.DEVICES.DETAIL.LINK, { id, tab: pages.DEVICES.DETAIL.TABS[i], section: '' }))

            refreshPendingCommands()
            refreshSoftwareUpdate()
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [id]
    )

    if (deviceError) {
        return <NotFoundPage message={_(t.deviceNotFound)} title={_(t.deviceNotFoundMessage, { id })} />
    }

    if (resourcesError) {
        return <NotFoundPage message={_(t.deviceResourcesNotFound)} title={_(t.deviceResourcesNotFoundMessage, { id })} />
    }

    if (certificates?.length === 0 && defaultActiveTab === 2) {
        return <NotFoundPage message={_(t.deviceCertificatesNotFound)} title={_(t.deviceCertificatesNotFoundMessage, { id })} />
    }

    if (isEmpty(provisioningRecords) && defaultActiveTab === 3) {
        return <NotFoundPage message={_(t.deviceProvisioningRecordNotFound)} title={_(t.deviceProvisioningRecordNotFoundMessage, { id })} />
    }

    const resources = resourcesData?.[0]?.resources || []
    const deviceStatus = data?.metadata?.connection?.status
    const isOnline = isDeviceOnline(data)
    const isUnregistered = devicesStatuses.UNREGISTERED === deviceStatus
    const deviceName = data?.name || NO_DEVICE_NAME
    const breadcrumbs: BreadcrumbItem[] = [
        {
            link: generatePath(pages.DEVICES.LINK),
            label: _(menuT.devices),
        },
    ]

    if (deviceName) {
        breadcrumbs.push({ label: deviceName })
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
        <div
            style={{
                height: '100%',
                display: 'flex',
                flexDirection: 'column',
            }}
        >
            <PageLayout
                pendingCommands
                breadcrumbs={breadcrumbs}
                dataTestId={testId.devices.detail.layout}
                deviceId={id}
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
                loading={loading || twinSyncLoading || pendingCommandsLoading || certificatesLoading || provisioningRecordsLoading}
                title={deviceName}
                xPadding={false}
            >
                <Tabs
                    fullHeight
                    innerPadding
                    isAsync
                    activeItem={activeTabItem}
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
                                    endpoints={data?.metadata?.connection?.localEndpoints}
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
                        {
                            content: (
                                <Tab3
                                    certificates={certificates}
                                    handleTabChange={handleTabChange}
                                    loading={certificatesLoading}
                                    refresh={certificateRefresh}
                                />
                            ),
                            dataTestId: testId.devices.detail.tabCertificates,
                            disabled: !certificates || certificates?.length === 0 || certificatesLoading,
                            id: 2,
                            name: _(t.certificates),
                            innerPadding: false,
                        },
                        {
                            content: <Tab4 provisioningRecords={provisioningRecords} />,
                            dataTestId: testId.devices.detail.tabProvisioningRecords,
                            disabled: provisioningRecords?.length === 0 || isEmpty(provisioningRecords) || provisioningRecordsLoading,
                            id: 3,
                            name: _(t.dps),
                        },
                    ]}
                />

                <EditNameModal
                    dataTestId={testId.devices.detail.editNameModal}
                    handleClose={() => setShowEditNameModal(false)}
                    handleSubmit={updateDeviceName}
                    i18n={{
                        close: _(t.close),
                        namePlaceholder: _(t.deviceName),
                        edit: _(t.edit),
                        name: _(t.name),
                        reset: _(t.reset),
                        saveChange: _(t.saveChange),
                        savingChanges: _(t.savingChanges),
                    }}
                    loading={deviceNameLoading}
                    name={deviceName}
                    show={showEditNameModal}
                />
            </PageLayout>
        </div>
    )
}

DevicesDetailsPage.displayName = 'DevicesDetailsPage'

export default DevicesDetailsPage
