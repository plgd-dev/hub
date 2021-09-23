import PropTypes from 'prop-types'
import { useMemo } from 'react'
import { useIntl } from 'react-intl'
import { Link } from 'react-router-dom'
import classNames from 'classnames'
import { useHistory } from 'react-router-dom'

import { Button } from '@/components/button'
import { Badge } from '@/components/badge'
import { Table } from '@/components/table'
import { IndeterminateCheckbox } from '@/components/checkbox'

import { ThingsListActionButton } from './_things-list-action-button'
import {
  thingsStatuses,
  THINGS_DEFAULT_PAGE_SIZE,
  NO_DEVICE_NAME,
} from './constants'
import { thingShape } from './shapes'
import { shadowSynchronizationEnabled } from './utils'
import { messages as t } from './things-i18n'

const { ONLINE, UNREGISTERED } = thingsStatuses

export const ThingsList = ({
  data,
  loading,
  setSelectedDevices,
  selectedDevices,
  onDeleteClick,
  unselectRowsToken,
}) => {
  const { formatMessage: _ } = useIntl()
  const history = useHistory()

  const columns = useMemo(
    () => [
      {
        id: 'selection',
        accessor: 'selection',
        disableSortBy: true,
        style: { width: '36px' },
        Header: ({ getToggleAllRowsSelectedProps }) => (
          <IndeterminateCheckbox {...getToggleAllRowsSelectedProps()} />
        ),
        Cell: ({ row }) => {
          if (row.original?.metadata?.status?.value === UNREGISTERED) {
            return null
          }

          return <IndeterminateCheckbox {...row.getToggleRowSelectedProps()} />
        },
      },
      {
        Header: _(t.name),
        accessor: 'name',
        Cell: ({ value, row }) => {
          const deviceName = value || NO_DEVICE_NAME

          if (row.original?.metadata?.status?.value === UNREGISTERED) {
            return <span>{deviceName}</span>
          }
          return (
            <Link to={`/things/${row.original?.id}`}>
              <span className="no-wrap-text">{deviceName}</span>
            </Link>
          )
        },
        style: { width: '100%' },
      },
      {
        Header: 'ID',
        accessor: 'id',
        style: { maxWidth: '350px', width: '100%' },
        Cell: ({ value }) => {
          return <span className="no-wrap-text">{value}</span>
        },
      },
      {
        Header: _(t.shadowSynchronization),
        accessor: 'metadata.shadowSynchronization',
        Cell: ({ value }) => {
          const isShadowSynchronizationEnabled =
            shadowSynchronizationEnabled(value)
          return (
            <Badge className={isShadowSynchronizationEnabled ? 'green' : 'red'}>
              {isShadowSynchronizationEnabled ? _(t.enabled) : _(t.disabled)}
            </Badge>
          )
        },
      },
      {
        Header: _(t.status),
        accessor: 'metadata.status.value',
        style: { width: '120px' },
        Cell: ({ value }) => {
          const isOnline = ONLINE === value
          return (
            <Badge className={isOnline ? 'green' : 'red'}>
              {isOnline ? _(t.online) : _(t.offline)}
            </Badge>
          )
        },
      },
      {
        Header: _(t.actions),
        accessor: 'actions',
        disableSortBy: true,
        Cell: ({ row }) => {
          const {
            original: { id },
          } = row
          return (
            <ThingsListActionButton
              deviceId={id}
              onView={deviceId => history.push(`/things/${deviceId}`)}
              onDelete={onDeleteClick}
            />
          )
        },
        className: 'actions',
      },
    ],
    [] // eslint-disable-line
  )

  return (
    <Table
      className="with-selectable-rows"
      columns={columns}
      data={data || []}
      defaultSortBy={[
        {
          id: 'name',
          desc: false,
        },
      ]}
      autoFillEmptyRows
      defaultPageSize={THINGS_DEFAULT_PAGE_SIZE}
      getRowProps={row => ({
        className: classNames({
          'grayed-out': row.original?.status === UNREGISTERED,
        }),
      })}
      getColumnProps={column => {
        if (column.id === 'actions') {
          return { style: { textAlign: 'center' } }
        }

        return {}
      }}
      primaryAttribute="id"
      onRowsSelect={setSelectedDevices}
      bottomControls={
        <Button
          onClick={onDeleteClick}
          variant="secondary"
          icon="fa-trash-alt"
          disabled={selectedDevices.length === 0 || loading}
        >
          {_(t.delete)}
        </Button>
      }
      unselectRowsToken={unselectRowsToken}
    />
  )
}

ThingsList.propTypes = {
  data: PropTypes.arrayOf(thingShape),
  selectedDevices: PropTypes.arrayOf(PropTypes.string).isRequired,
  setSelectedDevices: PropTypes.func.isRequired,
  loading: PropTypes.bool.isRequired,
  onDeleteClick: PropTypes.func.isRequired,
  unselectRowsToken: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
}

ThingsList.defaultProps = {
  data: [],
  unselectRowsToken: null,
}
