import { forwardRef, useEffect, useImperativeHandle, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'

import ConfirmModal from '@shared-ui/components/Atomic/ConfirmModal'
import Table from '@shared-ui/components/Atomic/TableNew'
import { useIsMounted } from '@shared-ui/common/hooks'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { WebSocketEventClient, eventFilters } from '@shared-ui/common/services'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import PendingCommandDetailsModal from '../PendingCommandDetailsModal'
import { EMBEDDED_PENDING_COMMANDS_DEFAULT_PAGE_SIZE, NEW_PENDING_COMMAND_WS_KEY, UPDATE_PENDING_COMMANDS_WS_KEY } from '../constants'
import { handleEmitNewPendingCommand, handleEmitUpdatedCommandEvents } from '../utils'
import { usePendingCommandsList } from '../hooks'
import { cancelPendingCommandApi } from '../rest'
import { messages as t } from '../PendingCommands.i18n'
import { ConfirmModalData, ModalData, PendingCommandsListRefType, Props } from './PendingCommandsList.types'
import notificationId from '@/notificationId'

const PendingCommandsList = forwardRef<PendingCommandsListRefType, Props>((props, ref) => {
    const { columns, onLoading, embedded, deviceId, isPage } = props
    const { formatMessage: _ } = useIntl()

    const { data, loading, error } = usePendingCommandsList(deviceId)

    const [canceling, setCanceling] = useState(false)
    const [confirmModalData, setConfirmModalData] = useState<null | ConfirmModalData>(null)
    const [detailsModalData, setDetailsModalData] = useState<null | ModalData>(null)
    const isMounted = useIsMounted()
    const deviceIdWsFilters = useMemo(() => (deviceId ? { deviceIdFilter: [deviceId] } : {}), [deviceId])

    useImperativeHandle(ref, () => ({
        setDetailsModalData: (data: ModalData | null) => setDetailsModalData(data),
        setConfirmModalData: (data: ConfirmModalData | null) => setConfirmModalData(data),
    }))

    useEffect(() => {
        error &&
            Notification.error(
                {
                    title: _(t.error),
                    message: getApiErrorMessage(error),
                },
                { notificationId: notificationId.HUB_PENDING_COMMANDS_LIST_ERROR }
            )
    }, [_, error])

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
            (eventData: any) => handleEmitUpdatedCommandEvents(eventData, _)
        )

        return () => {
            WebSocketEventClient.unsubscribe(NEW_PENDING_COMMAND_WS_KEY)
            WebSocketEventClient.unsubscribe(UPDATE_PENDING_COMMANDS_WS_KEY)
        }
    }, [_, deviceIdWsFilters])

    const onCloseViewModal = () => {
        setDetailsModalData(null)
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

            Notification.error(
                {
                    title: _(t.error),
                    message: getApiErrorMessage(error),
                },
                { notificationId: notificationId.HUB_PENDING_COMMANDS_LIST_CANCEL_COMMAND }
            )
        }
    }

    const loadingPendingCommands = loading || canceling

    useEffect(() => {
        if (onLoading) {
            onLoading(loadingPendingCommands)
        }
    }, [loadingPendingCommands]) // eslint-disable-line

    return (
        <>
            <Table
                autoHeight={!deviceId}
                columns={columns}
                data={(isPage ? data : data.slice(0, 10)) || []}
                defaultPageSize={embedded ? EMBEDDED_PENDING_COMMANDS_DEFAULT_PAGE_SIZE : 1000}
                defaultSortBy={[
                    {
                        id: 'eventMetadata.timestamp',
                        desc: true,
                    },
                ]}
                globalSearch={!isPage}
                height={isPage ? undefined : 350}
                i18n={{
                    search: _(t.search),
                }}
                paginationPortalTargetId={isPage ? 'paginationPortalTarget' : undefined}
                rowHeight={!isPage ? 40 : 54}
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
})

PendingCommandsList.displayName = 'PendingCommandsList'

export default PendingCommandsList
