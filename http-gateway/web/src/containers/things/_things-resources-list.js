import { useMemo } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'
import classNames from 'classnames'

import { Badge } from '@/components/badge'
import { Table } from '@/components/table'
import { ThingsResourcesActionButton } from './_things-resources-action-button'
import { RESOURCES_DEFAULT_PAGE_SIZE, thingsStatuses } from './constants'
import { thingResourceShape } from './shapes'
import { messages as t } from './things-i18n'

export const ThingsResourcesList = ({
  data,
  onUpdate,
  onCreate,
  onDelete,
  deviceStatus,
  loading,
}) => {
  const { formatMessage: _ } = useIntl()

  const isUnregistered = deviceStatus === thingsStatuses.UNREGISTERED
  const greyedOutClassName = classNames({ 'grayed-out': isUnregistered })

  const columns = useMemo(
    () => [
      {
        Header: _(t.location),
        accessor: 'href',
        Cell: ({ value, row }) => {
          const {
            original: { deviceId, href },
          } = row
          if (isUnregistered) {
            return <span>{value}</span>
          }
          return (
            <div className="tree-expander-container">
              <span
                className="link reveal-icon-on-hover"
                onClick={() => onUpdate({ deviceId, href })}
              >
                {value}
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
        Cell: ({ value }) => {
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
          const {
            original: { deviceId, href, interfaces },
          } = row
          return (
            <ThingsResourcesActionButton
              disabled={isUnregistered || loading}
              href={href}
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
    <Table
      columns={columns}
      data={data || []}
      defaultSortBy={[
        {
          id: 'href',
          desc: false,
        },
      ]}
      defaultPageSize={RESOURCES_DEFAULT_PAGE_SIZE}
      autoFillEmptyRows
      className={greyedOutClassName}
      paginationProps={{
        className: greyedOutClassName,
        disabled: isUnregistered,
      }}
    />
  )
}

ThingsResourcesList.propTypes = {
  data: PropTypes.arrayOf(thingResourceShape),
  onCreate: PropTypes.func.isRequired,
  onUpdate: PropTypes.func.isRequired,
  onDelete: PropTypes.func.isRequired,
  loading: PropTypes.bool.isRequired,
  deviceStatus: PropTypes.oneOf(Object.values(thingsStatuses)),
}

ThingsResourcesList.defaultProps = {
  data: null,
  deviceStatus: null,
}
