import React, { FC, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { Link, useHistory } from 'react-router-dom'
import Button from '@shared-ui/components/new/Button'
import Badge from '@shared-ui/components/new/Badge'
import Table from '@shared-ui/components/new/TableNew'
import DevicesListActionButton from '../DevicesListActionButton'
import { devicesStatuses, DEVICES_DEFAULT_PAGE_SIZE, NO_DEVICE_NAME } from '../../constants'
import { messages as t } from '../../Devices.i18n'
import { Props, defaultProps } from './DevicesList.types'
import { isDeviceOnline } from '@/containers/Devices/utils'
import TableSelectionPanel from '@plgd/shared-ui/src/components/new/TableNew/TableSelectionPanel'
import StatusPill from '@shared-ui/components/new/StatusPill'
import { states } from '@shared-ui/components/new/StatusPill/constants'
import Tag from '@shared-ui/components/new/Tag'

const { UNREGISTERED } = devicesStatuses

export const DevicesList: FC<Props> = (props) => {
    const { data, setSelectedDevices, selectedDevices, onDeleteClick, unselectRowsToken, isAllSelected, setIsAllSelected } = {
        ...defaultProps,
        ...props,
    }
    const { formatMessage: _ } = useIntl()
    const history = useHistory()

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

                    return (
                        <StatusPill
                            label={isOnline ? 'Online' : 'Offline'}
                            // pending={value.pending.show ? { text: `${value.pending.number} pending commands`, onClick: console.log } : undefined}
                            status={isOnline ? states.ONLINE : states.OFFLINE}
                            tooltipText={
                                isOnline ? (
                                    'Connected at: xxxxx'
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
            {
                Header: 'Protocol',
                accessor: 'protocol',
                Cell: ({ row }: { row: any }) => {
                    return 'Protocol'
                },
            },
            {
                Header: 'Shared',
                accessor: 'shared',
                style: { width: '120px' },
                Cell: ({ row }: { row: any }) => (
                    <Tag icon='link' onClick={console.log}>
                        Yes
                    </Tag>
                ),
            },
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
                    return <DevicesListActionButton deviceId={id} onDelete={onDeleteClick} onView={(deviceId) => history.push(`/devices/${deviceId}`)} />
                },
                className: 'actions',
            },
        ],
        [] // eslint-disable-line
    )

    const validData = (data: any) => (!data || data[0] === undefined ? [] : data)
    const selectedCount = useMemo(() => Object.keys(selectedDevices).length, [selectedDevices])

    return (
        <>
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
                onRowsSelect={(isAllRowsSelected, selection) => {
                    isAllRowsSelected !== isAllSelected && setIsAllSelected && setIsAllSelected(isAllRowsSelected)
                    setSelectedDevices(selection)
                }}
                paginationPortalTarget={document.getElementById('paginationPortalTarget')}
                primaryAttribute='id'
                unselectRowsToken={unselectRowsToken}
            />
            <TableSelectionPanel
                actionPrimary={<Button variant='primary'>Main Action</Button>}
                actionSecondary={
                    <Button htmlType='button' key={1}>
                        Secondary Action
                    </Button>
                }
                selectionInfo={`${selectedCount} device${selectedCount > 1 ? 's' : ''} `}
                show={selectedCount > 0}
            />
        </>
    )
}

DevicesList.displayName = 'DevicesList'
DevicesList.defaultProps = defaultProps

export default DevicesList
