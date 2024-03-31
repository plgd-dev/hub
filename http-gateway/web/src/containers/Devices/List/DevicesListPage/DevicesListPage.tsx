import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import { getApiErrorMessage } from '@shared-ui/common/utils'
import { useIsMounted } from '@shared-ui/common/hooks'
import { Emitter } from '@shared-ui/common/services/emitter'
import { messages as menuT } from '@shared-ui/components/Atomic/Menu/Menu.i18n'
import { DeleteModal } from '@shared-ui/components/Atomic/Modal'
import StatusPill from '@shared-ui/components/Atomic/StatusPill'
import { states } from '@shared-ui/components/Atomic/StatusPill/constants'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { IconArrowDetail, IconTrash, StatusTag } from '@shared-ui/components/Atomic'
import { clientAppSettings } from '@shared-ui/common/services'

import { DEVICES_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY, devicesStatuses, NO_DEVICE_NAME, RESET_COUNTER } from '../../constants'
import { useDevicesList } from '../../hooks'
import DevicesListHeader from '../DevicesListHeader/DevicesListHeader'
import { deleteDevicesApi } from '../../rest'
import { handleDeleteDevicesErrors, isDeviceOnline, sleep } from '../../utils'
import { messages as t } from '../../Devices.i18n'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import { messages as g } from '@/containers/Global.i18n'
import notificationId from '@/notificationId'
import PageLayout from '@/containers/Common/PageLayout'
import TableList from '@/containers/Common/TableList/TableList'
import { pages } from '@/routes'

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

    const [selected, setSelected] = useState<string[]>([])
    const [deleting, setDeleting] = useState(false)
    const [unselectRowsToken, setUnselectRowsToken] = useState(1)
    const isMounted = useIsMounted()
    const navigate = useNavigate()

    clientAppSettings.reset()

    useEffect(() => {
        deviceError &&
            Notification.error(
                { title: _(t.deviceError), message: getApiErrorMessage(deviceError) },
                { notificationId: notificationId.HUB_DEVICES_LIST_PAGE_DEVICE_ERROR }
            )
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [deviceError])

    const handleOpenDeleteModal = useCallback((_isAllSelected: boolean, selection: string[]) => {
        setSelected(selection)
    }, [])

    const handleCloseDeleteModal = () => {
        setSelected([])
        setUnselectRowsToken((prev) => prev + 1)
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
            await deleteDevicesApi(selected)
            await sleep(200)

            if (isMounted.current) {
                Notification.success(
                    { title: _(t.devicesDeleted), message: _(t.devicesDeletedMessage) },
                    { notificationId: notificationId.HUB_DEVICES_LIST_PAGE_DELETE_DEVICES }
                )

                setDeleting(false)
                setSelected([])
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
                        <a
                            data-test-id={`device-row-${row.id}`}
                            href={generatePath(pages.DEVICES.DETAIL.LINK, { id: row.original?.id, tab: '', section: '' })}
                            onClick={(e) => {
                                e.preventDefault()
                                navigate(generatePath(pages.DEVICES.DETAIL.LINK, { id: row.original?.id, tab: '', section: '' }))
                            }}
                        >
                            <span className='no-wrap-text'>{deviceName}</span>
                        </a>
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
                    return (
                        <StatusPill
                            label={isOnline ? _(t.online) : _(t.offline)}
                            status={isOnline ? states.ONLINE : states.OFFLINE}
                            tooltipText={
                                <DateFormat
                                    prefixTest={isOnline ? `${_(t.connectedAt)}: ` : `${_(t.lastTimeOnline)}: `}
                                    value={row.original?.metadata?.connection.connectedAt}
                                />
                            }
                        />
                    )
                },
            },
            {
                Header: _(t.twinSynchronization),
                accessor: 'metadata.twinEnabled',
                Cell: ({ value }: { value: string | number }) => {
                    const isTwinEnabled = value
                    return <StatusTag variant={isTwinEnabled ? 'success' : 'error'}>{isTwinEnabled ? _(t.enabled) : _(t.disabled)}</StatusTag>
                },
            },
            {
                Header: _(g.action),
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
                                    onClick: () => handleOpenDeleteModal(false, [id]),
                                    label: _(t.delete),
                                    icon: <IconTrash />,
                                },
                                {
                                    onClick: () => navigate(generatePath(pages.DEVICES.DETAIL.LINK, { id, tab: '', section: '' })),
                                    label: _(t.view),
                                    icon: <IconArrowDetail />,
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

    const selectedDevicesCount = useMemo(() => selected.length, [selected])
    const selectedDeviceName = useMemo(
        () => (selectedDevicesCount === 1 && data ? data.find?.((d: any) => d.id === selected[0])?.name : null),
        [selected, selectedDevicesCount, data]
    )

    return (
        <PageLayout
            pendingCommands
            breadcrumbs={[{ label: _(menuT.devices), link: '/' }]}
            header={<DevicesListHeader loading={loading} refresh={handleRefresh} />}
            loading={loading}
            title={_(menuT.devices)}
        >
            <TableList
                columns={columns}
                data={data}
                defaultSortBy={[
                    {
                        id: 'name',
                        desc: false,
                    },
                ]}
                i18n={{
                    multiSelected: _(t.devices),
                    singleSelected: _(t.device),
                }}
                onDeleteClick={handleOpenDeleteModal}
                unselectRowsToken={unselectRowsToken}
            />

            <DeleteModal
                deleteInformation={
                    selectedDevicesCount === 1
                        ? [
                              { label: _(t.deviceName), value: selectedDeviceName },
                              { label: _(t.deviceId), value: selected[0] },
                          ]
                        : undefined
                }
                footerActions={[
                    {
                        label: _(t.cancel),
                        onClick: handleCloseDeleteModal,
                        variant: 'tertiary',
                        disabled: loading,
                    },
                    {
                        label: _(t.delete),
                        onClick: deleteDevices,
                        variant: 'primary',
                        loading: deleting,
                        loadingText: _(g.loading),
                        disabled: loading,
                    },
                ]}
                fullSizeButtons={selectedDevicesCount > 1}
                maxWidth={440}
                maxWidthTitle={320}
                onClose={handleCloseDeleteModal}
                show={selectedDevicesCount > 0}
                subTitle={_(t.deleteDeviceMessageSubTitle)}
                title={selectedDevicesCount === 1 ? _(t.deleteDeviceMessage) : _(t.deleteDevicesMessage, { count: selectedDevicesCount })}
            />
        </PageLayout>
    )
}

DevicesListPage.displayName = 'DevicesListPage'

export default DevicesListPage
