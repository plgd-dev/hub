import { FC, memo, useMemo } from 'react'
import { useIntl } from 'react-intl'
import classNames from 'classnames'

import Switch from '@shared-ui/components/Atomic/Switch'
import { useLocalStorage } from '@shared-ui/common/hooks'
import DevicesResourcesList from '@shared-ui/components/Organisms/DevicesResourcesList'
import DevicesResourcesTree from '@shared-ui/components/Organisms/DevicesResourcesTree'
import TreeExpander from '@shared-ui/components/Atomic/TreeExpander'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { canCreateResource } from '@shared-ui/common/utils'
import { IconPlus, IconEdit, IconTrash } from '@shared-ui/components/Atomic/Icon'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import TagGroup from '@shared-ui/components/Atomic/TagGroup'
import { messages as app } from '@shared-ui/app/clientApp/App/App.i18n'
import Tag from '@shared-ui/components/Atomic/Tag'

import { devicesStatuses, RESOURCE_TREE_DEPTH_SIZE } from '../../constants'
import { messages as t } from '../../Devices.i18n'
import { GetColumnsType, Props } from './DevicesResources.types'
import { getLastPartOfAResourceHref } from '@/containers/Devices/utils'

const getTableAction = ({ _, isUnregistered, loading, onCreate, cleanHref, interfaces, onUpdate, deviceId, onDelete }: any) => (
    <TableActionButton
        disabled={isUnregistered || loading}
        items={[
            {
                onClick: () => onCreate(cleanHref),
                label: _(t.create),
                icon: <IconPlus />,
                hidden: !canCreateResource(interfaces),
            },
            {
                onClick: () => onUpdate({ deviceId, href: cleanHref }),
                label: _(t.update),
                icon: <IconEdit />,
            },
            {
                onClick: () => onDelete(cleanHref),
                label: _(t.delete),
                icon: <IconTrash />,
            },
        ]}
    />
)

const getColumns = ({ _, onUpdate, loading, isUnregistered, onCreate, onDelete }: GetColumnsType) => [
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
            return getTableAction({
                _,
                isUnregistered,
                loading,
                onCreate,
                cleanHref: href.replace(/\/$/, ''),
                interfaces,
                onUpdate,
                deviceId,
                onDelete,
            })
        },
        className: 'actions',
    },
]

const getTreeColumns = ({ _, onUpdate, onCreate, onDelete, isUnregistered, loading }: GetColumnsType) => [
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
                            expanded={row.isExpanded}
                            style={{
                                marginLeft: `${row.depth * RESOURCE_TREE_DEPTH_SIZE}px`,
                            }}
                        />
                        <span className={classNames(deviceId && 'link')} onClick={onLinkClick}>
                            {`/${lastValue}/`}
                        </span>
                    </div>
                )
            }

            return (
                <div
                    className='tree-expander-container'
                    style={{
                        marginLeft: `${row.depth === 0 ? 0 : row.depth * RESOURCE_TREE_DEPTH_SIZE}px`,
                    }}
                >
                    {row.depth > 0 && (
                        <span
                            style={{
                                display: 'block',
                                width: 15,
                            }}
                        ></span>
                    )}
                    <span className='link' onClick={onLinkClick}>
                        {`/${lastValue}`}
                    </span>
                </div>
            )
        },
        style: { width: '40%' },
    },
    {
        Header: _(t.types),
        accessor: 'resourceTypes',
        Cell: ({ value, row }: { value: any; row: any }) => {
            if (!row.original.deviceId) {
                return null
            }

            return (
                <TagGroup
                    i18n={{
                        more: _(app.more),
                        types: _(app.types),
                    }}
                >
                    {value?.map?.((type: string) => (
                        <Tag className='tree-custom-tag' key={type} variant={tagVariants.DEFAULT}>
                            {type}
                        </Tag>
                    ))}
                </TagGroup>
            )
        },
    },
    {
        Header: _(t.actions),
        accessor: 'actions',
        className: 'actions',
        disableSortBy: true,
        Cell: ({ row }: { row: any }) => {
            const {
                original: { deviceId, href, interfaces },
            } = row
            return getTableAction({
                _,
                isUnregistered,
                loading,
                onCreate,
                cleanHref: href.replace(/\/$/, ''),
                interfaces,
                onUpdate,
                deviceId,
                onDelete,
            })
        },
    },
]

const DevicesResources: FC<Props> = memo((props) => {
    const { data, onUpdate, onCreate, onDelete, deviceStatus, isActiveTab, loading, pageSize } = props
    const { formatMessage: _ } = useIntl()
    const [treeViewActive, setTreeViewActive] = useLocalStorage('treeViewActive', false)
    const isUnregistered = devicesStatuses.UNREGISTERED === deviceStatus
    const greyedOutClassName = classNames({
        'grayed-out': isUnregistered,
    })

    const columns = useMemo(
        () => getColumns({ _, onUpdate, loading, isUnregistered, onCreate, onDelete }),
        [onUpdate, onCreate, onDelete, isUnregistered, loading] //eslint-disable-line
    )

    const treeColumns = useMemo(
        () => getTreeColumns({ _, onUpdate, onCreate, onDelete, isUnregistered, loading }),
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
})

DevicesResources.displayName = 'DevicesResources'

export default DevicesResources
