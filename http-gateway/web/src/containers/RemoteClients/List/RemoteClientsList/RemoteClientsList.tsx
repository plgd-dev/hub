import React, { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useIntl } from 'react-intl'
import ReactDOM from 'react-dom'

import Table, { TableSelectionPanel } from '@shared-ui/components/Atomic/TableNew'
import { DEVICES_DEFAULT_PAGE_SIZE } from '@shared-ui/common/constants'
import Button from '@shared-ui/components/Atomic/Button'
import StatusPill from '@shared-ui/components/Atomic/StatusPill'
import { states } from '@shared-ui/components/Atomic/StatusPill/constants'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { IconTrash } from '@shared-ui/components/Atomic/Icon'
import Breadcrumbs from '@shared-ui/components/Layout/Header/Breadcrumbs'
import { remoteClientStatuses, RemoteClientStatusesType } from '@shared-ui/app/clientApp/RemoteClients/constants'
import IconEdit from '@shared-ui/components/Atomic/Icon/components/IconEdit'
import AppContext from '@shared-ui/app/share/AppContext'
import IconRefresh from '@shared-ui/components/Atomic/Icon/components/IconRefresh'
import { reset } from '@shared-ui/app/clientApp/App/AppRest'

import { Props } from './RemoteClientsList.types'
import { messages as t } from '../../RemoteClients.i18n'
import { messages as g } from '../../../Global.i18n'
import { NO_DEVICE_NAME } from '@/containers/Devices/constants'

const RemoteClientsList: FC<Props> = (props) => {
    const { data, isAllSelected, selectedClients, setIsAllSelected, setSelectedClients, handleOpenDeleteModal } = props
    const { formatMessage: _ } = useIntl()
    const [isDomReady, setIsDomReady] = useState(false)
    const selectedCount = useMemo(() => Object.keys(selectedClients).length, [selectedClients])
    const navigate = useNavigate()

    const { collapsed } = useContext(AppContext)

    useEffect(() => {
        setIsDomReady(true)
    }, [])

    const getStatusData = useCallback((status: RemoteClientStatusesType) => {
        switch (status) {
            case remoteClientStatuses.DIFFERENT_OWNER:
                return {
                    message: _(t.occupied),
                    status: states.OCCUPIED,
                }
            case remoteClientStatuses.UNREACHABLE:
                return {
                    message: _(t.unReachable),
                    status: states.OFFLINE,
                }
            case remoteClientStatuses.REACHABLE:
            default:
                return {
                    message: _(t.reachable),
                    status: states.ONLINE,
                }
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const handleResetClient = useCallback((clientUrl: string) => {
        reset(clientUrl).then()
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
                Cell: ({ value }: { value: string }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(g.status),
                accessor: 'status',
                style: { width: '200px' },
                Cell: ({ row }: { row: any }) => {
                    const statusData = getStatusData(row.original.status)
                    return <StatusPill label={statusData.message} status={statusData.status} />
                },
            },
            {
                Header: _(t.version),
                accessor: 'version',
                style: { width: '200px' },
                Cell: ({ value }: { value: string }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(g.action),
                accessor: 'action',
                style: { width: '66px' },
                disableSortBy: true,
                Cell: ({ row }: any) => (
                    <TableActionButton
                        items={[
                            {
                                onClick: () => handleResetClient(row.original.clientUrl),
                                label: _(g.reset),
                                icon: <IconRefresh />,
                                hidden: process.env.NODE_ENV !== 'development',
                            },
                            {
                                onClick: () => navigate(`/remote-clients/${row.original.id}/configuration`),
                                label: _(g.edit),
                                icon: <IconEdit />,
                            },
                            {
                                onClick: () => handleOpenDeleteModal(row.original.id),
                                label: _(g.delete),
                                icon: <IconTrash />,
                            },
                        ]}
                    />
                ),
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
                autoHeight
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
