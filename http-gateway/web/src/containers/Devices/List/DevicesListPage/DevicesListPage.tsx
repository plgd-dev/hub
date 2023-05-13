import React, { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import isFunction from 'lodash/isFunction'
import { Link, useHistory } from 'react-router-dom'

import { getApiErrorMessage } from '@shared-ui/common/utils'
import { useIsMounted } from '@shared-ui/common/hooks'
import { Emitter } from '@shared-ui/common/services/emitter'
import { messages as menuT } from '@shared-ui/components/new/Menu/Menu.i18n'
import PageLayout from '@shared-ui/components/new/PageLayout'
import { DeleteModal } from '@shared-ui/components/new/Modal'
import Footer from '@shared-ui/components/new/Layout/Footer'
import DevicesList from '@shared-ui/components/organisms/DevicesList/DevicesList'
import StatusPill from '@shared-ui/components/new/StatusPill'
import { states } from '@shared-ui/components/new/StatusPill/constants'
import Badge from '@shared-ui/components/new/Badge'
import TableActionButton from '@shared-ui/components/organisms/TableActionButton'

import { PendingCommandsExpandableList } from '@/containers/PendingCommands'
import { DEVICES_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY, devicesStatuses, NO_DEVICE_NAME, RESET_COUNTER } from '../../constants'
import { useDevicesList } from '../../hooks'
import DevicesListHeader from '../DevicesListHeader/DevicesListHeader'
import { deleteDevicesApi } from '../../rest'
import { handleDeleteDevicesErrors, isDeviceOnline, sleep } from '../../utils'
import { messages as t } from '../../Devices.i18n'
import { AppContext } from '@/containers/App/AppContext'
import Notification from '@shared-ui/components/new/Notification/Toast'

const { UNREGISTERED } = devicesStatuses

const DevicesListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const {
        data,
        loading,
        error: deviceError,
        refresh,
    }: {
        data: any
        loading: boolean
        error: any
        refresh: () => void
    } = useDevicesList()
    const [deleteModalOpen, setDeleteModalOpen] = useState(false)
    const [isAllSelected, setIsAllSelected] = useState(false)
    const [selectedDevices, setSelectedDevices] = useState([])
    const [singleDevice, setSingleDevice] = useState<null | string>(null)
    const [deleting, setDeleting] = useState(false)
    const [unselectRowsToken, setUnselectRowsToken] = useState(1)
    const isMounted = useIsMounted()
    const history = useHistory()
    const combinedSelectedDevices = singleDevice ? [singleDevice] : selectedDevices
    const { footerExpanded, setFooterExpanded, collapsed } = useContext(AppContext)

    useEffect(() => {
        deviceError && Notification.error({ title: _(t.deviceError), message: getApiErrorMessage(deviceError) })
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [deviceError])

    const handleOpenDeleteModal = useCallback(
        (deviceId?: string) => {
            if (typeof deviceId === 'string') {
                setSingleDevice(deviceId)
            } else if (singleDevice && !deviceId) {
                setSingleDevice(null)
            }

            setDeleteModalOpen(true)
        },
        [singleDevice]
    )

    const handleCloseDeleteModal = () => {
        setSingleDevice(null)
        setDeleteModalOpen(false)
    }

    const handleRefresh = () => {
        refresh()

        // Unselect all rows from the table
        setUnselectRowsToken((prevValue) => prevValue + 1)

        // Reset the counter on the Refresh button
        Emitter.emit(DEVICES_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY, RESET_COUNTER)
    }

    const deleteDevices = async () => {
        try {
            setDeleting(true)
            await deleteDevicesApi(combinedSelectedDevices)
            await sleep(200)

            if (isMounted.current) {
                Notification.success({ title: _(t.devicesDeleted), message: _(t.devicesDeletedMessage) })

                setDeleting(false)
                setDeleteModalOpen(false)
                setSingleDevice(null)
                setSelectedDevices([])
                handleCloseDeleteModal()
                handleRefresh()
            }
        } catch (error) {
            setDeleting(false)
            handleDeleteDevicesErrors(error, _)
        }
    }

    const columns = useMemo(
        () => [
            {
                Header: _(t.name),
                accessor: 'name',
                Cell: ({ value, row }: any) => {
                    const deviceName = value || NO_DEVICE_NAME

                    if (row.original?.metadata?.connection?.status === UNREGISTERED) {
                        return <span>{deviceName}</span>
                    }
                    return (
                        <Link to={`/devices/${row.original?.id}`}>
                            <span className='no-wrap-text'>{deviceName}</span>
                        </Link>
                    )
                },
            },
            {
                Header: 'ID',
                accessor: 'id',
                Cell: ({ value }: { value: string | number }) => {
                    return <span className='no-wrap-text'>{value}</span>
                },
            },
            {
                Header: _(t.status),
                accessor: 'metadata.connection.status',
                style: { width: '120px' },
                Cell: ({ row }: { row: any }) => {
                    const isOnline = isDeviceOnline(row.original)

                    const connectedAtDate = new Date(row.original?.metadata?.connection.connectedAt / 1000000)

                    return (
                        <StatusPill
                            label={isOnline ? 'Online' : 'Offline'}
                            // pending={value.pending.show ? { text: `${value.pending.number} pending commands`, onClick: console.log } : undefined}
                            status={isOnline ? states.ONLINE : states.OFFLINE}
                            tooltipText={
                                isOnline ? (
                                    `Connected at: ${connectedAtDate?.toLocaleDateString('en-US')}`
                                ) : (
                                    <span style={{ whiteSpace: 'nowrap' }}>
                                        Last time online: <strong>31.9.2022 - 16:32</strong>
                                    </span>
                                )
                            }
                        />
                    )
                },
            },
            // {
            //     Header: 'Protocol',
            //     accessor: 'protocol',
            //     Cell: ({ row }: { row: any }) => {
            //         return 'Protocol'
            //     },
            // },
            // {
            //     Header: 'Shared',
            //     accessor: 'shared',
            //     style: { width: '120px' },
            //     Cell: ({ row }: { row: any }) => (
            //         <Tag icon='link' onClick={console.log}>
            //             Yes
            //         </Tag>
            //     ),
            // },
            {
                Header: _(t.twinSynchronization),
                accessor: 'metadata.twinEnabled',
                Cell: ({ value }: { value: string | number }) => {
                    const isTwinEnabled = value
                    return <Badge className={isTwinEnabled ? 'green' : 'red'}>{isTwinEnabled ? _(t.enabled) : _(t.disabled)}</Badge>
                },
            },
            {
                Header: _(t.action),
                accessor: 'action',
                disableSortBy: true,
                Cell: ({ row }: any) => {
                    const {
                        original: { id },
                    } = row
                    return (
                        <TableActionButton
                            items={[
                                {
                                    onClick: () => handleOpenDeleteModal(id),
                                    label: _(t.delete),
                                    icon: 'trash',
                                },
                                {
                                    onClick: () => history.push(`/devices/${id}`),
                                    label: _(t.view),
                                    icon: 'icon-show-password',
                                },
                            ]}
                        />
                    )
                },
                className: 'actions',
            },
        ],
        [] // eslint-disable-line
    )

    const selectedDevicesCount = combinedSelectedDevices.length
    const selectedDeviceName = selectedDevicesCount === 1 && data ? data.find?.((d: any) => d.id === combinedSelectedDevices[0])?.name : null
    const loadingOrDeleting = loading || deleting

    return (
        <PageLayout
            breadcrumbs={[
                {
                    label: _(menuT.devices),
                },
            ]}
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
            header={<DevicesListHeader loading={loading} refresh={handleRefresh} />}
            loading={loading}
            title={_(menuT.devices)}
        >
            <DevicesList
                collapsed={collapsed ?? false}
                columns={columns}
                data={data}
                i18n={{
                    delete: _(t.delete),
                    search: _(t.search),
                }}
                isAllSelected={isAllSelected}
                loading={loadingOrDeleting}
                onDeleteClick={handleOpenDeleteModal}
                selectedDevices={selectedDevices}
                setIsAllSelected={setIsAllSelected}
                setSelectedDevices={setSelectedDevices}
                unselectRowsToken={unselectRowsToken}
            />

            <PendingCommandsExpandableList />

            <DeleteModal
                deleteInformation={
                    selectedDevicesCount === 1
                        ? [
                              { label: _(t.deviceName), value: selectedDeviceName },
                              { label: _(t.deviceId), value: combinedSelectedDevices[0] },
                          ]
                        : undefined
                }
                footerActions={[
                    {
                        label: _(t.cancel),
                        onClick: handleCloseDeleteModal,
                        variant: 'tertiary',
                    },
                    {
                        label: _(t.delete),
                        onClick: deleteDevices,
                        variant: 'primary',
                    },
                ]}
                fullSizeButtons={selectedDevicesCount > 1}
                maxWidth={440}
                maxWidthTitle={320}
                onClose={handleCloseDeleteModal}
                show={deleteModalOpen}
                subTitle={_(t.deleteDeviceMessageSubTitle)}
                title={selectedDevicesCount === 1 ? _(t.deleteDeviceMessage) : _(t.deleteDevicesMessage, { count: selectedDevicesCount })}
            />
        </PageLayout>
    )
}

DevicesListPage.displayName = 'DevicesListPage'

export default DevicesListPage
