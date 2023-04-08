import { FC, useMemo } from 'react'
import { useIntl } from 'react-intl'
import classNames from 'classnames'

import Table from '@shared-ui/components/new/TableNew'
import TableActionButton from '@shared-ui/components/organisms/TableActionButton'

import { RESOURCES_DEFAULT_PAGE_SIZE, devicesStatuses } from '../../constants'
import { messages as t } from '../../Devices.i18n'
import { Props } from './DevicesResourcesList.types'
import { canCreateResource } from '@/containers/Devices/utils'

const DevicesResourcesList: FC<Props> = (props) => {
    const { data, onUpdate, onCreate, onDelete, deviceStatus, isActiveTab, loading, pageSize } = props
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
                    const cleanHref = href.replace(/\/$/, '') // href without a trailing slash
                    return (
                        <TableActionButton
                            disabled={isUnregistered || loading}
                            items={[
                                {
                                    onClick: () => onCreate(cleanHref),
                                    label: _(t.create),
                                    icon: 'plus',
                                    hidden: !canCreateResource(interfaces),
                                },
                                {
                                    onClick: () => onUpdate({ deviceId, href: cleanHref }),
                                    label: _(t.update),
                                    icon: 'edit',
                                },
                                {
                                    onClick: () => onDelete(cleanHref),
                                    label: _(t.delete),
                                    icon: 'trash',
                                },
                            ]}
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
            height={pageSize.height}
            i18n={{
                search: _(t.search),
            }}
            paginationPortalTargetId={isActiveTab ? 'paginationPortalTarget' : undefined}
        />
    )
}

DevicesResourcesList.displayName = 'DevicesResourcesList'

export default DevicesResourcesList
