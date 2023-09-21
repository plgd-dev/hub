import { FC, useEffect, useState, useContext, useMemo, useRef } from 'react'
import ReactDOM from 'react-dom'
import { useIntl } from 'react-intl'

import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import { TagTypeType } from '@shared-ui/components/Atomic/StatusTag/StatusTag.types'
import TableActions from '@shared-ui/components/Atomic/TableNew/TableActions'
import { IconTrash } from '@shared-ui/components/Atomic'

import { Props } from './PendingCommandsExpandableList.types'
import PendingCommandsList from '../PendingCommandsList'
import { AppContext } from '@/containers/App/AppContext'
import { motion, AnimatePresence } from 'framer-motion'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import { getPendingCommandStatusColorAndLabel, hasCommandExpired } from '@/containers/PendingCommands/utils'
import { messages as t } from '../PendingCommands.i18n'
import { PENDING_COMMANDS_LIST_REFRESH_INTERVAL_MS } from '@/containers/PendingCommands/constants'
import { PendingCommandsListRefType } from '../PendingCommandsList/PendingCommandsList.types'

const PendingCommandsExpandableList: FC<Props> = ({ deviceId }) => {
    const { formatMessage: _ } = useIntl()

    const [domReady, setDomReady] = useState(false)
    const [currentTime, setCurrentTime] = useState(Date.now())

    const { footerExpanded } = useContext(AppContext)

    const pendingCommandsListRef = useRef<PendingCommandsListRefType>(null)

    useEffect(() => {
        setDomReady(true)
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
                                    pendingCommandsListRef?.current?.setDetailsModalData({
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
                            return <StatusTag variant={color as TagTypeType}>{_(label)}</StatusTag>
                        }

                        return <StatusTag variant={color as TagTypeType}>{_(label)}</StatusTag>
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
                                        icon: <IconTrash />,
                                        onClick: () => pendingCommandsListRef?.current?.setConfirmModalData({ deviceId: rowDeviceId, href, correlationId }),
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
                    Cell: ({ row }: { row: any }) => <span className='no-wrap-text'>{row?.original?.resourceId?.deviceId || row?.original?.deviceId}</span>,
                })
            }

            return cols
        },
        [currentTime] // eslint-disable-line
    )

    return (
        <>
            {domReady &&
                footerExpanded &&
                ReactDOM.createPortal(
                    <AnimatePresence mode='wait'>
                        {footerExpanded && (
                            <motion.div
                                layout
                                animate={{ opacity: 1, paddingTop: 12 }}
                                exit={{
                                    opacity: 0,
                                    paddingTop: 0,
                                }}
                                initial={{ opacity: 0, paddingTop: 0 }}
                                transition={{
                                    duration: 0.3,
                                }}
                            >
                                <PendingCommandsList columns={columns} deviceId={deviceId} ref={pendingCommandsListRef} />
                            </motion.div>
                        )}
                    </AnimatePresence>,
                    document.querySelector('#recentTasksPortalTarget') as Element
                )}
        </>
    )
}

PendingCommandsExpandableList.displayName = 'PendingCommandsExpandableList'

export default PendingCommandsExpandableList
