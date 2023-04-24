import { FC, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { toast } from 'react-toastify'

import ConfirmModal from '@shared-ui/components/new/ConfirmModal'
import Badge from '@shared-ui/components/new/Badge'
import Table from '@shared-ui/components/new/TableNew'
import { useIsMounted } from '@shared-ui/common/hooks'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { WebSocketEventClient, eventFilters } from '@shared-ui/common/services'
import TableActions from '@shared-ui/components/new/TableNew/TableActions'

import PendingCommandDetailsModal from '../PendingCommandDetailsModal'
import {
    PENDING_COMMANDS_DEFAULT_PAGE_SIZE,
    EMBEDDED_PENDING_COMMANDS_DEFAULT_PAGE_SIZE,
    PENDING_COMMANDS_LIST_REFRESH_INTERVAL_MS,
    NEW_PENDING_COMMAND_WS_KEY,
    UPDATE_PENDING_COMMANDS_WS_KEY,
} from '../constants'
import { getPendingCommandStatusColorAndLabel, hasCommandExpired, handleEmitNewPendingCommand, handleEmitUpdatedCommandEvents } from '../utils'
import { usePendingCommandsList } from '../hooks'
import { cancelPendingCommandApi } from '../rest'
import { messages as t } from '../PendingCommands.i18n'
import { Props } from './PendingCommandsList.types'
import DateFormat from '@/containers/PendingCommands/DateFormat'

type ModalData = {
    content: any
    commandType?: any
}

type ConfirmModalData = {
    deviceId: string
    href: string
    correlationId: string
}

// This component contains also all the modals and websocket connections, used for
// interacting with pending commands because it is reused on three different places.
const PendingCommandsList: FC<Props> = ({ onLoading, embedded, deviceId }) => {
    const { formatMessage: _ } = useIntl()
    const [currentTime, setCurrentTime] = useState(Date.now())

    const { data, loading, error } = usePendingCommandsList(deviceId)

    const [canceling, setCanceling] = useState(false)
    const [confirmModalData, setConfirmModalData] = useState<null | ConfirmModalData>(null)
    const [detailsModalData, setDetailsModalData] = useState<null | ModalData>(null)
    const isMounted = useIsMounted()
    const deviceIdWsFilters = useMemo(() => (deviceId ? { deviceIdFilter: [deviceId] } : {}), [deviceId])

    useEffect(() => {
        if (error) {
            toast.error(getApiErrorMessage(error))
        }
    }, [error])

    useEffect(() => {
        // WS for adding a new pending command to the list
        WebSocketEventClient.subscribe(
            {
                eventFilter: [
                    eventFilters.RESOURCE_CREATE_PENDING,
                    eventFilters.RESOURCE_DELETE_PENDING,
                    eventFilters.RESOURCE_UPDATE_PENDING,
                    eventFilters.RESOURCE_RETRIEVE_PENDING,
                    eventFilters.DEVICE_METADATA_UPDATE_PENDING,
                ],
                ...deviceIdWsFilters,
            },
            NEW_PENDING_COMMAND_WS_KEY,
            handleEmitNewPendingCommand
        )

        // WS for updating the status of a pending command
        WebSocketEventClient.subscribe(
            {
                eventFilter: [
                    eventFilters.RESOURCE_CREATED,
                    eventFilters.RESOURCE_DELETED,
                    eventFilters.RESOURCE_UPDATED,
                    eventFilters.RESOURCE_RETRIEVED,
                    eventFilters.DEVICE_METADATA_UPDATED,
                ],
                ...deviceIdWsFilters,
            },
            UPDATE_PENDING_COMMANDS_WS_KEY,
            handleEmitUpdatedCommandEvents
        )

        return () => {
            WebSocketEventClient.unsubscribe(NEW_PENDING_COMMAND_WS_KEY)
            WebSocketEventClient.unsubscribe(UPDATE_PENDING_COMMANDS_WS_KEY)
        }
    }, [deviceIdWsFilters])

    const onViewClick = ({ content, commandType }: ModalData) => {
        setDetailsModalData({ content, commandType })
    }

    const onCloseViewModal = () => {
        setDetailsModalData(null)
    }

    const onCancelClick = (data: ConfirmModalData) => {
        setConfirmModalData(data)
    }

    const onCloseCancelModal = () => {
        setConfirmModalData(null)
    }

    const cancelCommand = async () => {
        try {
            setCanceling(true)
            await cancelPendingCommandApi(confirmModalData as ConfirmModalData)

            if (isMounted.current) {
                setCanceling(false)
                onCloseCancelModal()
            }
        } catch (error) {
            onCloseCancelModal()
            toast.error(getApiErrorMessage(error))
        }
    }

    const loadingPendingCommands = loading || canceling

    useEffect(() => {
        if (onLoading) {
            onLoading(loadingPendingCommands)
        }
    }, [loadingPendingCommands]) // eslint-disable-line

    useEffect(() => {
        const timeout = setInterval(() => {
            setCurrentTime(Date.now())
        }, PENDING_COMMANDS_LIST_REFRESH_INTERVAL_MS)

        return () => {
            clearInterval(timeout)
        }
    }, [])

    const columns = useMemo(
        () => {
            const cols = [
                {
                    Header: 'CI',
                    accessor: 'auditContext.correlationId',
                    disableSortBy: true,
                    Cell: ({ value }: { value: any }) => value,
                },
                {
                    Header: _(t.created),
                    accessor: 'eventMetadata.timestamp',
                    disableSortBy: true,
                    Cell: ({ value }: { value: any }) => <DateFormat value={value} />,
                },
                {
                    Header: _(t.type),
                    accessor: 'commandType',
                    disableSortBy: true,
                    Cell: ({ value, row }: { value: any; row: any }) => {
                        const {
                            original: { content },
                        } = row
                        const href = row.original?.resourceId?.href
                        // @ts-ignore
                        const text = _(t[value])

                        if (!content && !href) {
                            // @ts-ignore
                            return <span className='no-wrap-text'>{text}</span>
                        }

                        return (
                            <span
                                className='no-wrap-text link'
                                onClick={() =>
                                    onViewClick({
                                        content,
                                        commandType: value,
                                    })
                                }
                            >
                                {text}
                            </span>
                        )
                    },
                },
                {
                    Header: _(t.resourceHref),
                    accessor: 'resourceId.href',
                    disableSortBy: true,
                    Cell: ({ value }: { value: any }) => {
                        return <span className='no-wrap-text'>{value || '-'}</span>
                    },
                },
                {
                    Header: _(t.status),
                    accessor: 'status',
                    disableSortBy: true,
                    Cell: ({ value, row }: { value: any; row: any }) => {
                        const { validUntil } = row.original
                        const { color, label } = getPendingCommandStatusColorAndLabel(value, validUntil, currentTime)

                        if (!value) {
                            return <Badge className={color}>{_(label)}</Badge>
                        }

                        return <Badge className={color}>{_(label)}</Badge>
                    },
                },
                {
                    Header: _(t.validUntil),
                    accessor: 'validUntil',
                    disableSortBy: true,
                    Cell: ({ value }: { value: any }) => {
                        if (value === '0') return _(t.forever)

                        return <DateFormat value={value} />
                    },
                },
                {
                    Header: _(t.actions),
                    accessor: 'actions',
                    disableSortBy: true,
                    Cell: ({ row }: { row: any }) => {
                        const {
                            original: {
                                auditContext: { correlationId },
                                status,
                                validUntil,
                            },
                        }: any = row

                        const href = row.original?.resourceId?.href
                        const rowDeviceId = row?.original?.resourceId?.deviceId || row?.original?.deviceId

                        if (status || hasCommandExpired(validUntil, currentTime)) {
                            return <div className='no-action' />
                        }

                        return (
                            <TableActions
                                items={[
                                    {
                                        icon: 'trash',
                                        onClick: () => onCancelClick({ deviceId: rowDeviceId, href, correlationId }),
                                        id: `delete-row-${deviceId}`,
                                        tooltipText: _(t.cancel),
                                    },
                                ]}
                            />
                        )
                    },
                    className: 'actions',
                },
            ]

            // Only show device id column when not on the device details
            if (!deviceId) {
                cols.splice(2, 0, {
                    Header: _(t.deviceId),
                    accessor: 'resourceId.deviceId',
                    disableSortBy: true,
                    Cell: ({ row }: { row: any }) => {
                        return <span className='no-wrap-text'>{row?.original?.resourceId?.deviceId || row?.original?.deviceId}</span>
                    },
                })
            }

            return cols
        },
        [currentTime] // eslint-disable-line
    )

    return (
        <>
            <Table
                columns={columns}
                data={data || []}
                defaultPageSize={embedded ? EMBEDDED_PENDING_COMMANDS_DEFAULT_PAGE_SIZE : PENDING_COMMANDS_DEFAULT_PAGE_SIZE}
                defaultSortBy={[
                    {
                        id: 'eventMetadata.timestamp',
                        desc: true,
                    },
                ]}
                globalSearch={false}
                height={350}
                i18n={{
                    search: _(t.search),
                }}
                rowHeight={40}
            />

            <PendingCommandDetailsModal {...detailsModalData} onClose={onCloseViewModal} />

            <ConfirmModal
                body={_(t.doYouWantToCancelThisCommand)}
                cancelButtonText={_(t.no)}
                confirmButtonText={_(t.yes)}
                loading={canceling}
                onClose={onCloseCancelModal}
                onConfirm={cancelCommand}
                show={!!confirmModalData}
                title={<>{_(t.cancelPendingCommand)}</>}
            />
        </>
    )
}

PendingCommandsList.displayName = 'PendingCommandsList'

export default PendingCommandsList