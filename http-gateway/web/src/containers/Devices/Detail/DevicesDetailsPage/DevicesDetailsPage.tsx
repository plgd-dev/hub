import React, { useEffect, useState } from 'react'
import ReactDOM from 'react-dom'
import { useIntl } from 'react-intl'
import { useParams } from 'react-router-dom'
import NotFoundPage from '@/containers/NotFoundPage'
import { useIsMounted } from '@shared-ui/common/hooks'

import { messages as menuT } from '@shared-ui/components/new/Menu/Menu.i18n'

import DevicesDetailsHeader from '../DevicesDetailsHeader'
import { devicesStatuses, NO_DEVICE_NAME } from '../../constants'
import { handleTwinSynchronizationErrors, isDeviceOnline } from '../../utils'
import { updateDeviceTwinSynchronizationApi } from '../../rest'
import { useDeviceDetails, useDevicesResources } from '../../hooks'

import { messages as t } from '../../Devices.i18n'
import './DevicesDetailsPage.scss'
import PageLayout from '@shared-ui/components/new/PageLayout'
import Tabs from '@shared-ui/components/new/Tabs'
import Breadcrumbs from '@plgd/shared-ui/src/components/new/Layout/Header/Breadcrumbs'
import StatusTag from '@shared-ui/components/new/StatusTag'
import Tab1 from './Tabs/Tab1'
import Tab2 from '@/containers/Devices/Detail/DevicesDetailsPage/Tabs/Tab2'

const DevicesDetailsPage = () => {
    const { formatMessage: _ } = useIntl()
    const {
        id,
    }: {
        id: string
    } = useParams()
    const [domReady, setDomReady] = useState(false)
    const [twinSyncLoading, setTwinSyncLoading] = useState(false)

    const isMounted = useIsMounted()
    const { data, updateData, loading, error: deviceError } = useDeviceDetails(id)
    const { data: resourcesData, loading: loadingResources, error: resourcesError } = useDevicesResources(id)

    const [isTwinEnabled, setIsTwinEnabled] = useState<boolean>(data?.metadata?.twinEnabled || false)

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
    }

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={<DevicesDetailsHeader deviceId={id} deviceName={deviceName} isUnregistered={isUnregistered} />}
            headlineStatusTag={<StatusTag variant={isOnline ? 'success' : 'error'}>{isOnline ? _(t.online) : _(t.offline)}</StatusTag>}
            loading={loading || twinSyncLoading}
            title={deviceName}
        >
            {/* <DevicesDetailsTitle*/}
            {/*    className={classNames(*/}
            {/*        {*/}
            {/*            shimmering: loading,*/}
            {/*        },*/}
            {/*        greyedOutClassName*/}
            {/*    )}*/}
            {/*    deviceId={id}*/}
            {/*    deviceName={deviceName}*/}
            {/*    isOnline={isOnline}*/}
            {/*    links={resources}*/}
            {/*    loading={loading}*/}
            {/*    ttl={ttl}*/}
            {/*    updateDeviceName={updateDeviceNameInData}*/}
            {/* />*/}

            {domReady &&
                ReactDOM.createPortal(
                    <Breadcrumbs items={[{ label: _(menuT.devices), link: '/' }, { label: deviceName }]} />,
                    document.querySelector('#breadcrumbsPortalTarget') as Element
                )}

            <Tabs
                onItemChange={(activeItem) => console.log(`Active item: ${activeItem}`)}
                tabs={[
                    {
                        name: 'Device information',
                        content: (
                            <Tab1
                                deviceId={id}
                                isTwinEnabled={isTwinEnabled}
                                setTwinSynchronization={setTwinSynchronization}
                                twinSyncLoading={twinSyncLoading}
                                types={data?.types}
                            />
                        ),
                    },
                    {
                        name: 'Resources',
                        content: (
                            <Tab2
                                deviceName={deviceName}
                                deviceStatus={deviceStatus}
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

            {/* <PendingCommandsExpandableList deviceId={id} />*/}
        </PageLayout>
    )
}

DevicesDetailsPage.displayName = 'DevicesDetailsPage'

export default DevicesDetailsPage
