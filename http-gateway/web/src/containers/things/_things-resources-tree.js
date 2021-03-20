import { useMemo } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'
import classNames from 'classnames'

import { TreeExpander } from '@/components/tree-expander'
import { TreeTable } from '@/components/table'
import { Badge } from '@/components/badge'
import { ActionButton } from '@/components/action-button'

import { thingsStatuses } from './constants'
import { canCreateResource, createNestedResourceData } from './utils'
import { thingResourceShape } from './shapes'
import { messages as t } from './things-i18n'

const DEPTH_SIZE = 15

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
            original: { di, href },
          } = row
          if (isUnregistered) {
            return <span>{value}</span>
          }

          if (row.canExpand) {
            return (
              <span
                {...row.getToggleRowExpandedProps({
                  style: {
                    paddingLeft: `${row.depth * DEPTH_SIZE}px`,
                  },
                  className: 'tree-expander-container',
                  title: null,
                })}
              >
                <TreeExpander expanded={!!row.isExpanded} />
                {value}
              </span>
            )
          }

          return (
            <span
              className="link tree-expander-container"
              onClick={() => onUpdate({ di, href })}
              style={{
                paddingLeft: `${
                  row.depth === 0 ? 0 : (row.depth + 1) * DEPTH_SIZE
                }px`,
              }}
            >
              {value}
              <i className="fas fa-pen m-l-10" />
            </span>
          )
        },
        style: { width: '50%' },
      },
      {
        Header: _(t.types),
        accessor: 'rt',
        Cell: ({ value, row }) => {
          if (row.canExpand) {
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
          if (row.canExpand) {
            return null
          }

          const {
            original: { di, href, if: interfaces },
          } = row
          return (
            <ActionButton
              disabled={isUnregistered || loading}
              menuProps={{
                align: 'right',
              }}
              items={[
                {
                  onClick: () => onCreate(href),
                  label: _(t.create),
                  icon: 'fa-plus',
                  hidden: !canCreateResource(interfaces),
                },
                {
                  onClick: () => onUpdate({ di, href }),
                  label: _(t.update),
                  icon: 'fa-pen',
                },
                {
                  onClick: () => onDelete(href),
                  label: _(t.delete),
                  icon: 'fa-trash-alt',
                },
              ]}
            >
              <i className="fas fa-ellipsis-h" />
            </ActionButton>
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
