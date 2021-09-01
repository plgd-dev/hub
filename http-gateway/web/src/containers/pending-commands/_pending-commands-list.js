import { useEffect, useMemo, useState } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'
// import classNames from 'classnames'
import { time } from 'units-converter'
import OverlayTrigger from 'react-bootstrap/OverlayTrigger'
import Tooltip from 'react-bootstrap/Tooltip'

import { Badge } from '@/components/badge'
import { Table } from '@/components/table'

// import { PendingCommandsListActionButton } from './_pending-commands-list-action-button'
import { PENDING_COMMANDS_DEFAULT_PAGE_SIZE } from './constants'
import { getPendingCommandStatusColorAndLabel } from './utils'
import { pendingCommandShape } from './shapes'
import { messages as t } from './pending-commands-i18n'

import './pending-commands.scss'

export const PendingCommandsList = ({ data, onCancelClick, onViewClick }) => {
  const { formatMessage: _, formatDate, formatTime } = useIntl()
  const [currentTime, setCurrentTime] = useState(Date.now())

  useEffect(() => {
    const timeout = setInterval(() => {
      setCurrentTime(Date.now())
    }, 5000)

    return () => {
      clearInterval(timeout)
    }
  }, [])

  const columns = useMemo(
    () => [
      {
        Header: _(t.type),
        accessor: 'commandType',
        disableSortBy: true,
        Cell: ({ value, row }) => {
          const {
            original: {
              auditContext: { correlationId },
              resourceId: { deviceId, href },
            },
          } = row

          return (
            <span
              className="no-wrap-text link"
              onClick={() => onViewClick(deviceId, href, correlationId)}
            >
              {_(t[value])}
            </span>
          )
        },
      },
      {
        Header: _(t.deviceId),
        accessor: 'resourceId.deviceId',
        disableSortBy: true,
        Cell: ({ value }) => {
          return <span className="no-wrap-text">{value}</span>
        },
      },
      {
        Header: _(t.resourceHref),
        accessor: 'resourceId.href',
        disableSortBy: true,
        Cell: ({ value }) => {
          return <span className="no-wrap-text">{value}</span>
        },
      },
      {
        Header: _(t.status),
        accessor: 'status',
        disableSortBy: true,
        Cell: ({ value, row }) => {
          const { validUntil } = row.original
          const validUntilMs = time(validUntil)
            .from('ns')
            .to('ms').value
          const { color, label } = getPendingCommandStatusColorAndLabel(
            value,
            validUntilMs,
            currentTime
          )
          return <Badge className={color}>{_(label)}</Badge>
        },
      },
      {
        Header: _(t.validUntil),
        accessor: 'validUntil',
        disableSortBy: true,
        Cell: ({ value }) => {
          const date = new Date(
            time(value)
              .from('ns')
              .to('ms').value
          )
          const visibleDate = `${formatDate(date)} ${formatTime(date)}`
          const tooltipDate = `${formatDate(date)} ${formatTime(date)}`

          return (
            <OverlayTrigger
              placement="top"
              overlay={
                <Tooltip className="plgd-tooltip">{tooltipDate}</Tooltip>
              }
            >
              <span className="no-wrap-text tooltiped-text">{visibleDate}</span>
            </OverlayTrigger>
          )
        },
      },
      {
        Header: _(t.actions),
        accessor: 'actions',
        disableSortBy: true,
        Cell: ({ row }) => {
          const {
            original: {
              auditContext: { correlationId },
              resourceId: { deviceId, href },
            },
          } = row

          return (
            <div
              className="dropdown action-button"
              onClick={() => onCancelClick(deviceId, href, correlationId)}
              title={_(t.cancel)}
            >
              <button className="dropdown-toggle btn btn-empty">
                <i className="fas fa-times" />
              </button>
            </div>
          )
        },
      },
    ],
    [currentTime] // eslint-disable-line
  )

  return (
    <Table
      columns={columns}
      data={data || []}
      defaultSortBy={[
        {
          id: 'validUntil',
          desc: false,
        },
      ]}
      autoFillEmptyRows
      defaultPageSize={PENDING_COMMANDS_DEFAULT_PAGE_SIZE}
      getColumnProps={column => {
        if (column.id === 'actions') {
          return { style: { textAlign: 'center' } }
        }

        return {}
      }}
    />
  )
}

PendingCommandsList.propTypes = {
  data: PropTypes.arrayOf(pendingCommandShape),
  onCancelClick: PropTypes.func.isRequired,
  onViewClick: PropTypes.func.isRequired,
}

PendingCommandsList.defaultProps = {
  data: [],
}
