import { FC, useMemo } from 'react'
import { useIntl } from 'react-intl'
import classNames from 'classnames'

import Badge from '@shared-ui/components/new/Badge'
import Table from '@shared-ui/components/new/TableNew'
import DevicesResourcesActionButton from '../DevicesResourcesActionButton'
import { RESOURCES_DEFAULT_PAGE_SIZE, devicesStatuses } from '../../constants'
import { messages as t } from '../../Devices.i18n'
import { Props } from './DevicesResourcesList.types'

const DevicesResourcesList: FC<Props> = ({ data, onUpdate, onCreate, onDelete, deviceStatus, loading }) => {
    const { formatMessage: _ } = useIntl()

    const isUnregistered = deviceStatus === devicesStatuses.UNREGISTERED
    const greyedOutClassName = classNames({ 'grayed-out': isUnregistered })

    const columns = useMemo(
        () => [
            {
                Header: _(t.href),
                accessor: 'href',
                Cell: ({ value, row }: { value: any; row: any }) => {
                    const {
                        original: { deviceId, href },
                    } = row
                    if (isUnregistered) {
                        return <span>{value}</span>
                    }
                    return (
                        <div className='tree-expander-container'>
                            <span className='link reveal-icon-on-hover' onClick={() => onUpdate({ deviceId, href })}>
                                {value}
                            </span>
                        </div>
                    )
                },
                style: { width: '300px' },
            },
            {
                Header: _(t.types),
                accessor: 'resourceTypes',
                style: { width: '100%' },
                Cell: ({ value }: { value: any }) => value.join(', '),
            },
            {
                Header: _(t.actions),
                accessor: 'actions',
                disableSortBy: true,
                Cell: ({ row }: { row: any }) => {
                    const {
                        original: { deviceId, href, interfaces },
                    } = row
                    return (
                        <DevicesResourcesActionButton
                            deviceId={deviceId}
                            disabled={isUnregistered || loading}
                            href={href}
                            interfaces={interfaces}
                            onCreate={onCreate}
                            onDelete={onDelete}
                            onUpdate={onUpdate}
                        />
                    )
                },
                className: 'actions',
            },
        ],
        [onUpdate, onCreate, onDelete, isUnregistered, loading] //eslint-disable-line
    )

    return (
        <Table
            className={greyedOutClassName}
            columns={columns}
            data={data || []}
            defaultPageSize={RESOURCES_DEFAULT_PAGE_SIZE}
            defaultSortBy={[
                {
                    id: 'href',
                    desc: false,
                },
            ]}
            i18n={{
                search: _(t.search),
            }}
        />
    )
}

DevicesResourcesList.displayName = 'DevicesResourcesList'

export default DevicesResourcesList
