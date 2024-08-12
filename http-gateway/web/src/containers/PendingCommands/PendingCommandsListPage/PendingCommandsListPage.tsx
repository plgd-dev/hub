import React, { useEffect, useMemo, useRef, useState } from 'react'
import { useIntl } from 'react-intl'
import ReactDOM from 'react-dom'

import { messages as menuT } from '@shared-ui/components/Atomic/Menu/Menu.i18n'
import PageLayout from '@shared-ui/components/Atomic/PageLayout'
import Breadcrumbs from '@shared-ui/components/Layout/Header/Breadcrumbs'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import { TagTypeType } from '@shared-ui/components/Atomic/StatusTag/StatusTag.types'
import { IconTrash } from '@shared-ui/components/Atomic/Icon'
import Footer from '@shared-ui/components/Layout/Footer'
import IconArrowDetail from '@shared-ui/components/Atomic/Icon/components/IconArrowDetail'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'

import PendingCommandsList from '../PendingCommandsList'
import { PendingCommandsListRefType } from '@/containers/PendingCommands/PendingCommandsList/PendingCommandsList.types'
import { messages as t } from '@/containers/PendingCommands/PendingCommands.i18n'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import { getPendingCommandStatusColorAndLabel, hasCommandExpired } from '@/containers/PendingCommands/utils'
import { useCurrentTime } from '@/containers/PendingCommands/hooks'

const PendingCommandsListPage = () => {
    const { formatMessage: _ } = useIntl()
    const [loading, setLoading] = useState(false)
    const [domReady, setDomReady] = useState(false)
    const { currentTime } = useCurrentTime()

    // eslint-disable-next-line react-hooks/exhaustive-deps
    const breadcrumbs = useMemo(() => [{ label: _(menuT.pendingCommands), link: '/' }], [])

    const pendingCommandsListRef = useRef<PendingCommandsListRefType>(null)

    useEffect(() => {
        setDomReady(true)
    }, [])

    const columns = useMemo(
        () => [
            {
                Header: _(t.type),
                accessor: 'commandType',
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
                        <a
                            className='no-wrap-text'
                            href={href}
                            onClick={(e) => {
                                e.preventDefault()
                                pendingCommandsListRef?.current?.setDetailsModalData({
                                    content,
                                    commandType: value,
                                })
                            }}
                        >
                            {text}
                        </a>
                    )
                },
            },
            {
                Header: _(t.deviceId),
                accessor: 'resourceId.deviceId',
                Cell: ({ row }: { row: any }) => <span className='no-wrap-text'>{row?.original?.resourceId?.deviceId || row?.original?.deviceId}</span>,
            },
            {
                Header: _(t.resourceHref),
                accessor: 'resourceId.href',
                Cell: ({ value, row }: { value: any; row: any }) => {
                    const {
                        original: { content },
                    } = row

                    return (
                        <a
                            className='no-wrap-text link'
                            href={value}
                            onClick={(e) => {
                                e.preventDefault()
                                pendingCommandsListRef?.current?.setDetailsModalData({
                                    content,
                                    commandType: row.original.commandType,
                                })
                            }}
                        >
                            {value || '-'}
                        </a>
                    )
                },
            },
            {
                Header: _(t.status),
                accessor: 'status',
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
                Header: _(t.initiator),
                accessor: 'auditContext.userId',
                Cell: ({ value }: { value: any }) => <span className='no-wrap-text'>{value}</span>,
            },

            {
                Header: _(t.expiresAt),
                accessor: 'validUntil',
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

                    return (
                        <TableActionButton
                            items={[
                                {
                                    icon: <IconTrash />,
                                    onClick: () => pendingCommandsListRef?.current?.setConfirmModalData({ deviceId: rowDeviceId, href, correlationId }),
                                    id: `delete-row-${rowDeviceId}`,
                                    label: _(t.cancel),
                                    hidden: status || hasCommandExpired(validUntil, currentTime),
                                },
                                {
                                    icon: <IconArrowDetail />,
                                    onClick: () =>
                                        pendingCommandsListRef?.current?.setDetailsModalData({
                                            content: row.original.content,
                                            commandType: row.original.commandType,
                                        }),
                                    id: `detail-row-${rowDeviceId}`,
                                    label: _(t.details),
                                },
                            ]}
                        />
                    )
                },
                className: 'actions',
            },
        ],
        [currentTime] // eslint-disable-line
    )

    return (
        <PageLayout
            footer={<Footer footerExpanded={false} paginationComponent={<div id='paginationPortalTarget'></div>} />}
            loading={loading}
            title={_(menuT.pendingCommands)}
        >
            {domReady && ReactDOM.createPortal(<Breadcrumbs items={breadcrumbs} />, document.querySelector('#breadcrumbsPortalTarget') as Element)}
            <PendingCommandsList isPage columns={columns} onLoading={setLoading} ref={pendingCommandsListRef} />
        </PageLayout>
    )
}

PendingCommandsListPage.displayName = 'PendingCommandsListPage'

export default PendingCommandsListPage
