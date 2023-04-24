import { FC, useMemo } from 'react'
import { useIntl } from 'react-intl'
import classNames from 'classnames'

import Switch from '@shared-ui/components/new/Switch'
import { useLocalStorage } from '@shared-ui/common/hooks'
import DevicesResourcesList from '@shared-ui/components/organisms/DevicesResourcesList'
import DevicesResourcesTree from '@shared-ui/components/organisms/DevicesResourcesTree'
import TreeExpander from '@shared-ui/components/new/TreeExpander'
import Badge from '@shared-ui/components/new/Badge'
import TableActionButton from '@shared-ui/components/organisms/TableActionButton'
import { canCreateResource } from '@shared-ui/common/utils'

import { devicesStatuses, RESOURCE_TREE_DEPTH_SIZE } from '../../constants'
import { messages as t } from '../../Devices.i18n'
import { Props } from './DevicesResources.types'
import { getLastPartOfAResourceHref } from '@/containers/Devices/utils'

const DevicesResources: FC<Props> = (props) => {
    const { data, onUpdate, onCreate, onDelete, deviceStatus, isActiveTab, loading, pageSize } = props
    const { formatMessage: _ } = useIntl()
    const [treeViewActive, setTreeViewActive] = useLocalStorage('treeViewActive', false)
    const isUnregistered = devicesStatuses.UNREGISTERED === deviceStatus
    const greyedOutClassName = classNames({
        'grayed-out': isUnregistered,
    })

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

    const treeColumns = useMemo(
        () => [
            {
                Header: _(t.href),
                accessor: 'href',
                Cell: ({ value, row }: { value: any; row: any }) => {
                    const {
                        original: { deviceId, href },
                    } = row
                    const lastValue = getLastPartOfAResourceHref(value)
                    const onLinkClick = deviceId ? () => onUpdate({ deviceId, href: href.replace(/\/$/, '') }) : () => {}

                    if (isUnregistered) {
                        return <span>{lastValue}</span>
                    }

                    if (row.canExpand) {
                        return (
                            <div className='tree-expander-container'>
                                <TreeExpander
                                    {...row.getToggleRowExpandedProps({ title: null })}
                                    expanded={!!row.isExpanded}
                                    style={{
                                        marginLeft: `${row.depth * RESOURCE_TREE_DEPTH_SIZE}px`,
                                    }}
                                />
                                <span className={deviceId ? 'link reveal-icon-on-hover' : ''} onClick={onLinkClick}>
                                    {`/${lastValue}/`}
                                </span>
                                {deviceId && <i className='fas fa-pen' />}
                            </div>
                        )
                    }

                    return (
                        <div
                            className='tree-expander-container'
                            style={{
                                marginLeft: `${row.depth === 0 ? 0 : (row.depth + 1) * RESOURCE_TREE_DEPTH_SIZE}px`,
                            }}
                        >
                            <span className='link reveal-icon-on-hover' onClick={onLinkClick}>
                                {`/${lastValue}`}
                            </span>
                            <i className='fas fa-pen' />
                        </div>
                    )
                },
                style: { width: '100%' },
            },
            {
                Header: _(t.types),
                accessor: 'resourceTypes',
                Cell: ({ value, row }: { value: any; row: any }) => {
                    if (!row.original.deviceId) {
                        return null
                    }

                    return (
                        <div className='badges-box-horizontal'>
                            {value?.map?.((type: string) => (
                                <Badge key={type}>{type}</Badge>
                            ))}
                        </div>
                    )
                },
            },
            {
                Header: _(t.actions),
                accessor: 'actions',
                disableSortBy: true,
                Cell: ({ row }: { row: any }) => {
                    if (!row.original.deviceId) {
                        return null
                    }

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
            },
        ],
        [onUpdate, onCreate, onDelete, isUnregistered, loading] //eslint-disable-line
    )

    return (
        <>
            <div
                className={classNames('d-flex justify-content-between align-items-center', greyedOutClassName)}
                style={{
                    paddingBottom: 12,
                }}
            >
                <div></div>
                <div className='d-flex justify-content-end align-items-center'>
                    <Switch
                        checked={treeViewActive}
                        disabled={isUnregistered}
                        id='toggle-tree-view'
                        label={_(t.treeView)}
                        onChange={() => setTreeViewActive(!treeViewActive)}
                    />
                </div>
            </div>

            {treeViewActive ? (
                <DevicesResourcesTree columns={treeColumns} data={data} deviceStatus={deviceStatus} />
            ) : (
                <DevicesResourcesList
                    columns={columns}
                    data={data}
                    i18n={{
                        search: _(t.search),
                    }}
                    isActiveTab={isActiveTab}
                    pageSize={pageSize}
                />
            )}
        </>
    )
}

DevicesResources.displayName = 'DevicesResources'

export default DevicesResources