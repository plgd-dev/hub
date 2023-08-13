import React, { FC, useContext, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useIntl } from 'react-intl'
import ReactDOM from 'react-dom'

import Table, { TableSelectionPanel } from '@shared-ui/components/Atomic/TableNew'
import { DEVICES_DEFAULT_PAGE_SIZE } from '@shared-ui/common/constants'
import Button from '@shared-ui/components/Atomic/Button'
import StatusPill from '@shared-ui/components/Atomic/StatusPill'
import { states } from '@shared-ui/components/Atomic/StatusPill/constants'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { IconTrash } from '@shared-ui/components/Atomic'
import Breadcrumbs from '@shared-ui/components/Layout/Header/Breadcrumbs'
import { remoteClientStatuses } from '@shared-ui/app/clientApp/RemoteClients/constants'

import { Props } from './RemoteClientsList.types'
import { messages as t } from '../../RemoteClients.i18n'
import { messages as g } from '../../../Global.i18n'
import { NO_DEVICE_NAME } from '@/containers/Devices/constants'
import { AppContext } from '@/containers/App/AppContext'

const RemoteClientsList: FC<Props> = (props) => {
    const { data, isAllSelected, selectedClients, setIsAllSelected, setSelectedClients, handleOpenDeleteModal } = props
    const { formatMessage: _ } = useIntl()
    const [isDomReady, setIsDomReady] = useState(false)
    const selectedCount = useMemo(() => Object.keys(selectedClients).length, [selectedClients])

    const { collapsed } = useContext(AppContext)

    useEffect(() => {
        setIsDomReady(true)
    }, [])

    const columns = useMemo(
        () => [
            {
                Header: _(g.name),
                accessor: 'clientName',
                Cell: ({ value, row }: { value: string; row: any }) => {
                    const remoteClientName = value || NO_DEVICE_NAME

                    if (row.original.status === remoteClientStatuses.UNREACHABLE) {
                        return <span>{remoteClientName}</span>
                    }
                    return (
                        <Link to={`/remote-clients/${row.original?.id}`}>
                            <span className='no-wrap-text'>{remoteClientName}</span>
                        </Link>
                    )
                },
            },
            {
                Header: _(t.ipAddress),
                accessor: 'clientUrl',
                style: { width: '350px' },
                Cell: ({ value }: { value: string }) => {
                    return <span className='no-wrap-text'>{value}</span>
                },
            },
            {
                Header: _(g.status),
                accessor: 'status',
                style: { width: '200px' },
                Cell: ({ row }: { row: any }) => {
                    const isReachable = row.original.status === remoteClientStatuses.REACHABLE
                    return <StatusPill label={isReachable ? _(t.reachable) : _(t.unReachable)} status={isReachable ? states.ONLINE : states.OFFLINE} />
                },
            },
            {
                Header: _(t.version),
                accessor: 'version',
                style: { width: '200px' },
                Cell: ({ value }: { value: string }) => {
                    return <span className='no-wrap-text'>{value}</span>
                },
            },
            {
                Header: _(g.action),
                accessor: 'action',
                style: { width: '66px' },
                disableSortBy: true,
                Cell: ({ row }: any) => {
                    return (
                        <TableActionButton
                            items={[
                                {
                                    onClick: () => handleOpenDeleteModal(row.original.id),
                                    label: _(g.delete),
                                    icon: <IconTrash />,
                                },
                            ]}
                        />
                    )
                },
                className: 'actions',
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )
    return (
        <>
            {isDomReady &&
                ReactDOM.createPortal(
                    <Breadcrumbs items={[{ label: _(t.remoteClients), link: '/' }]} />,
                    document.querySelector('#breadcrumbsPortalTarget') as Element
                )}
            <Table
                autoHeight={true}
                columns={columns}
                data={data}
                defaultPageSize={DEVICES_DEFAULT_PAGE_SIZE}
                defaultSortBy={[
                    {
                        id: 'clientName',
                        desc: false,
                    },
                ]}
                i18n={{
                    search: _(g.search),
                }}
                onRowsSelect={(isAllRowsSelected, selection) => {
                    isAllRowsSelected !== isAllSelected && setIsAllSelected && setIsAllSelected(isAllRowsSelected)
                    setSelectedClients(selection)
                }}
                paginationPortalTargetId='paginationPortalTarget'
                primaryAttribute='clientName'
            />
            <TableSelectionPanel
                actionPrimary={
                    <Button onClick={handleOpenDeleteModal} variant='primary'>
                        {_(g.delete)}
                    </Button>
                }
                i18n={{
                    select: _(g.select),
                }}
                leftPanelCollapsed={collapsed}
                selectionInfo={`${selectedCount} ${selectedCount > 1 ? _(t.remoteClients) : _(t.remoteClient)} `}
                show={selectedCount > 0}
            />
        </>
    )
}

RemoteClientsList.displayName = 'RemoteClientsList'

export default RemoteClientsList
