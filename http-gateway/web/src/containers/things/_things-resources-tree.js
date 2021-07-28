import { useMemo } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'
import classNames from 'classnames'

import { TreeExpander } from '@/components/tree-expander'
import { TreeTable } from '@/components/table'
import { Badge } from '@/components/badge'
import { ThingsResourcesActionButton } from './_things-resources-action-button'
import { thingsStatuses, RESOURCE_TREE_DEPTH_SIZE } from './constants'
import { createNestedResourceData, getLastPartOfAResourceHref } from './utils'
import { thingResourceShape } from './shapes'
import { messages as t } from './things-i18n'

export const ThingsResourcesTree = ({
  data: rawData,
  onUpdate,
  onCreate,
  onDelete,
  deviceStatus,
  loading,
}) => {
  const { formatMessage: _ } = useIntl()

  const isUnregistered = deviceStatus === thingsStatuses.UNREGISTERED
  const greyedOutClassName = classNames({ 'grayed-out': isUnregistered })
  const data = useMemo(() => createNestedResourceData(rawData), [rawData])

  const columns = useMemo(
    () => [
      {
        Header: _(t.location),
        accessor: 'href',
        Cell: ({ value, row }) => {
          const {
            original: { deviceId, href },
          } = row
          const lastValue = getLastPartOfAResourceHref(value)
          const onLinkClick = deviceId
            ? () => onUpdate({ deviceId, href: href.replace(/\/$/, '') })
            : null

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
        Cell: ({ value, row }) => {
          if (!row.original.deviceId) {
            return null
          }

          return (
            <div className="badges-box-horizontal">
              {value?.map?.(type => <Badge key={type}>{type}</Badge>)}
            </div>
          )
        },
      },
      {
        Header: _(t.actions),
        accessor: 'actions',
        disableSortBy: true,
        Cell: ({ row }) => {
          if (!row.original.deviceId) {
            return null
          }

          const {
            original: { deviceId, href, interfaces },
          } = row
          const cleanHref = href.replace(/\/$/, '') // href without a trailing slash
          return (
            <ThingsResourcesActionButton
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

ThingsResourcesTree.propTypes = {
  data: PropTypes.arrayOf(thingResourceShape),
  onCreate: PropTypes.func.isRequired,
  onUpdate: PropTypes.func.isRequired,
  onDelete: PropTypes.func.isRequired,
  loading: PropTypes.bool.isRequired,
  deviceStatus: PropTypes.oneOf(Object.values(thingsStatuses)),
}

ThingsResourcesTree.defaultProps = {
  data: null,
  deviceStatus: null,
}
