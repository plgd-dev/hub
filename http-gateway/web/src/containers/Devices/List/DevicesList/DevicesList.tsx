import React, { FC, useContext, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { Link, useHistory } from 'react-router-dom'
import { useResizeDetector } from 'react-resize-detector'

import Button from '@shared-ui/components/Atomic/Button'
import Badge from '@shared-ui/components/Atomic/Badge'
import Table from '@shared-ui/components/Atomic/TableNew'
import TableSelectionPanel from '@shared-ui/components/Atomic/TableNew/TableSelectionPanel'
import StatusPill from '@shared-ui/components/Atomic/StatusPill'
import { states } from '@shared-ui/components/Atomic/StatusPill/constants'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'

import { devicesStatuses, DEVICES_DEFAULT_PAGE_SIZE, NO_DEVICE_NAME } from '../../constants'
import { messages as t } from '../../Devices.i18n'
import { Props, defaultProps } from './DevicesList.types'
import { isDeviceOnline } from '@/containers/Devices/utils'
import { AppContext } from '@/containers/App/AppContext'
import { IconShowPassword, IconTrash } from '@shared-ui/components/Atomic'

const { UNREGISTERED } = devicesStatuses

export const DevicesList: FC<Props> = (props) => {
    const { data, setSelectedDevices, selectedDevices, onDeleteClick, unselectRowsToken, isAllSelected, setIsAllSelected } = {
        ...defaultProps,
        ...props,
    }
    const { formatMessage: _ } = useIntl()
    const history = useHistory()
    const { collapsed } = useContext(AppContext)

    const { ref, height } = useResizeDetector()

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
                                    onClick: () => onDeleteClick(id),
                                    label: _(t.delete),
                                    icon: <IconTrash />,
                                },
                                {
                                    onClick: () => history.push(`/devices/${id}`),
                                    label: _(t.view),
                                    icon: <IconShowPassword />,
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

    const validData = (data: any) => (!data || data[0] === undefined ? [] : data)
    const selectedCount = useMemo(() => Object.keys(selectedDevices).length, [selectedDevices])

    return (
        <div
            ref={ref}
            style={{
                height: '100%',
            }}
        >
            <Table
                columns={columns}
                data={validData(data)}
                defaultPageSize={DEVICES_DEFAULT_PAGE_SIZE}
                defaultSortBy={[
                    {
                        id: 'name',
                        desc: false,
                    },
                ]}
                height={height}
                i18n={{
                    search: _(t.search),
                }}
                onRowsSelect={(isAllRowsSelected, selection) => {
                    isAllRowsSelected !== isAllSelected && setIsAllSelected && setIsAllSelected(isAllRowsSelected)
                    setSelectedDevices(selection)
                }}
                paginationPortalTargetId='paginationPortalTarget'
                primaryAttribute='id'
                unselectRowsToken={unselectRowsToken}
            />
            <TableSelectionPanel
                actionPrimary={
                    <Button onClick={() => onDeleteClick()} variant='primary'>
                        {_(t.delete)}
                    </Button>
                }
                leftPanelCollapsed={collapsed}
                selectionInfo={`${selectedCount} device${selectedCount > 1 ? 's' : ''} `}
                show={selectedCount > 0}
            />
        </div>
    )
}

DevicesList.displayName = 'DevicesList'
DevicesList.defaultProps = defaultProps

export default DevicesList
