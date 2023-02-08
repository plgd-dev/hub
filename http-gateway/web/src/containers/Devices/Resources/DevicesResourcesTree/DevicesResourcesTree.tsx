import { FC, useMemo } from 'react'
import { useIntl } from 'react-intl'
import classNames from 'classnames'

import TreeExpander from '@shared-ui/components/new/TreeExpander'
import { TreeTable } from '@shared-ui/components/new/Table'
import Badge from '@shared-ui/components/new/Badge'
import DevicesResourcesActionButton from '../DevicesResourcesActionButton'
import { devicesStatuses, RESOURCE_TREE_DEPTH_SIZE } from '../../constants'
import {
  createNestedResourceData,
  getLastPartOfAResourceHref,
} from '../../utils'
import { messages as t } from '../../Devices.i18n'
import { Props } from './DevicesResourcesTree.types'

const DevicesResourcesTree: FC<Props> = ({
  data: rawData,
  onUpdate,
  onCreate,
  onDelete,
  deviceStatus,
  loading,
}) => {
  const { formatMessage: _ } = useIntl()
  const isUnregistered = deviceStatus === devicesStatuses.UNREGISTERED
  const greyedOutClassName = classNames({ 'grayed-out': isUnregistered })
  const data = useMemo(() => createNestedResourceData(rawData), [rawData])

  const columns = useMemo(
    () => [
      {
        Header: _(t.href),
        accessor: 'href',
        Cell: ({ value, row }: { value: any; row: any }) => {
          const {
            original: { deviceId, href },
          } = row
          const lastValue = getLastPartOfAResourceHref(value)
          const onLinkClick = deviceId
            ? () => onUpdate({ deviceId, href: href.replace(/\/$/, '') })
            : () => {}

          if (isUnregistered) {
            return <span>{lastValue}</span>
          }

          if (row.canExpand) {
            return (
              <div className="tree-expander-container">
                <TreeExpander
                  {...row.getToggleRowExpandedProps({ title: null })}
                  expanded={!!row.isExpanded}
                  style={{
                    marginLeft: `${row.depth * RESOURCE_TREE_DEPTH_SIZE}px`,
                  }}
                />
                <span
                  className={deviceId ? 'link reveal-icon-on-hover' : ''}
                  onClick={onLinkClick}
                >
                  {`/${lastValue}/`}
                </span>
                {deviceId && <i className="fas fa-pen" />}
              </div>
            )
          }

          return (
            <div
              className="tree-expander-container"
              style={{
                marginLeft: `${
                  row.depth === 0
                    ? 0
                    : (row.depth + 1) * RESOURCE_TREE_DEPTH_SIZE
                }px`,
              }}
            >
              <span className="link reveal-icon-on-hover" onClick={onLinkClick}>
                {`/${lastValue}`}
              </span>
              <i className="fas fa-pen" />
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
            <div className="badges-box-horizontal">
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
            <DevicesResourcesActionButton
              disabled={isUnregistered || loading}
              href={cleanHref}
              deviceId={deviceId}
              interfaces={interfaces}
              onCreate={onCreate}
              onUpdate={onUpdate}
              onDelete={onDelete}
            />
          )
        },
      },
    ],
    [onUpdate, onCreate, onDelete, isUnregistered, loading] //eslint-disable-line
  )

  return (
    <TreeTable
      columns={columns}
      data={data || []}
      className={greyedOutClassName}
    />
  )
}

DevicesResourcesTree.displayName = 'DevicesResourcesTree'

export default DevicesResourcesTree
