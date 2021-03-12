import { useMemo } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'
import classNames from 'classnames'

import { Badge } from '@/components/badge'
import { Table } from '@/components/table'
import { ActionButton } from '@/components/action-button'

import { RESOURCES_DEFAULT_PAGE_SIZE, thingsStatuses } from './constants'
import { canCreateResource } from './utils'
import { thingResourceShape } from './shapes'
import { messages as t } from './things-i18n'

const { ONLINE, OFFLINE, REGISTERED, UNREGISTERED } = thingsStatuses

export const ThingsResourcesList = ({
  data,
  onUpdate,
  onCreate,
  onDelete,
  deviceStatus,
  loading,
}) => {
  const { formatMessage: _ } = useIntl()

  const isUnregistered = deviceStatus === UNREGISTERED
  const greyedOutClassName = classNames({ 'grayed-out': isUnregistered })

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
          return (
            <span className="link" onClick={() => onUpdate({ di, href })}>
              {value}
            </span>
          )
        },
        style: { width: '50%' },
      },
      {
        Header: _(t.types),
        accessor: 'rt',
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
  deviceStatus: PropTypes.oneOf([ONLINE, OFFLINE, REGISTERED, UNREGISTERED]),
}

ThingsResourcesList.defaultProps = {
  data: null,
  deviceStatus: null,
}
