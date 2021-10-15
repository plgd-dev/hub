import { useEffect, useMemo, useState } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'
import { time } from 'units-converter'
import OverlayTrigger from 'react-bootstrap/OverlayTrigger'
import Tooltip from 'react-bootstrap/Tooltip'
import { toast } from 'react-toastify'

import { ConfirmModal } from '@/components/confirm-modal'
import { Badge } from '@/components/badge'
import { Table } from '@/components/table'
import { useIsMounted } from '@/common/hooks'
import { getApiErrorMessage } from '@/common/utils'
import { WebSocketEventClient, eventFilters } from '@/common/services'

import { PendingCommandDetailsModal } from './_pending-command-details-modal'
import {
  PENDING_COMMANDS_DEFAULT_PAGE_SIZE,
  EMBEDDED_PENDING_COMMANDS_DEFAULT_PAGE_SIZE,
  PENDING_COMMANDS_LIST_REFRESH_INTERVAL_MS,
  NEW_PENDING_COMMAND_WS_KEY,
  UPDATE_PENDING_COMMANDS_WS_KEY,
  dateFormat,
  timeFormat,
  timeFormatLong,
} from './constants'
import {
  getPendingCommandStatusColorAndLabel,
  hasCommandExpired,
  handleEmitNewPendingCommand,
  handleEmitUpdatedCommandEvents,
} from './utils'
import { usePendingCommandsList } from './hooks'
import { cancelPendingCommandApi } from './rest'
import { messages as t } from './pending-commands-i18n'

import './pending-commands.scss'

const DateTooltip = ({ value }) => {
  const { formatDate, formatTime } = useIntl()
  const date = new Date(time(value).from('ns').to('ms').value)
  const visibleDate = `${formatDate(date, dateFormat)} ${formatTime(
    date,
    timeFormat
  )}`
  const tooltipDate = `${formatDate(date, dateFormat)} ${formatTime(
    date,
    timeFormatLong
  )}`

  return (
    <OverlayTrigger
      placement="top"
      overlay={<Tooltip className="plgd-tooltip">{tooltipDate}</Tooltip>}
    >
      <span className="no-wrap-text tooltiped-text">{visibleDate}</span>
    </OverlayTrigger>
  )
}

// This component contains also all the modals and websocket connections, used for
// interacting with pending commands because it is reused on three different places.
export const PendingCommandsList = ({ onLoading, embedded, deviceId }) => {
  const { formatMessage: _ } = useIntl()
  const [currentTime, setCurrentTime] = useState(Date.now())

  const { data, loading, error } = usePendingCommandsList(deviceId)
  const [canceling, setCanceling] = useState(false)
  const [confirmModalData, setConfirmModalData] = useState(null)
  const [detailsModalData, setDetailsModalData] = useState(null)
  const isMounted = useIsMounted()
  const deviceIdWsFilters = useMemo(
    () => (deviceId ? { deviceIdFilter: [deviceId] } : {}),
    [deviceId]
  )

  useEffect(() => {
    if (error) {
      toast.error(getApiErrorMessage(error))
    }
  }, [error])

  useEffect(() => {
    // WS for adding a new pending command to the list
    WebSocketEventClient.subscribe(
      {
        eventFilter: [
          eventFilters.RESOURCE_CREATE_PENDING,
          eventFilters.RESOURCE_DELETE_PENDING,
          eventFilters.RESOURCE_UPDATE_PENDING,
          eventFilters.RESOURCE_RETRIEVE_PENDING,
          eventFilters.DEVICE_METADATA_UPDATE_PENDING,
        ],
        ...deviceIdWsFilters,
      },
      NEW_PENDING_COMMAND_WS_KEY,
      handleEmitNewPendingCommand
    )

    // WS for updating the status of a pending command
    WebSocketEventClient.subscribe(
      {
        eventFilter: [
          eventFilters.RESOURCE_CREATED,
          eventFilters.RESOURCE_DELETED,
          eventFilters.RESOURCE_UPDATED,
          eventFilters.RESOURCE_RETRIEVED,
          eventFilters.DEVICE_METADATA_UPDATED,
        ],
        ...deviceIdWsFilters,
      },
      UPDATE_PENDING_COMMANDS_WS_KEY,
      handleEmitUpdatedCommandEvents
    )

    return () => {
      WebSocketEventClient.unsubscribe(NEW_PENDING_COMMAND_WS_KEY)
      WebSocketEventClient.unsubscribe(UPDATE_PENDING_COMMANDS_WS_KEY)
    }
  }, [deviceIdWsFilters])

  const onViewClick = ({ content, commandType }) => {
    setDetailsModalData({ content, commandType })
  }

  const onCloseViewModal = () => {
    setDetailsModalData(null)
  }

  const onCancelClick = data => {
    setConfirmModalData(data)
  }

  const onCloseCancelModal = () => {
    setConfirmModalData(null)
  }

  const cancelCommand = async () => {
    try {
      setCanceling(true)
      await cancelPendingCommandApi(confirmModalData)

      if (isMounted.current) {
        setCanceling(false)
        onCloseCancelModal()
      }
    } catch (error) {
      onCloseCancelModal()
      toast.error(getApiErrorMessage(error))
    }
  }

  const loadingPendingCommands = loading || canceling

  useEffect(() => {
    if (onLoading) {
      onLoading(loadingPendingCommands)
    }
  }, [loadingPendingCommands]) // eslint-disable-line

  useEffect(() => {
    const timeout = setInterval(() => {
      setCurrentTime(Date.now())
    }, PENDING_COMMANDS_LIST_REFRESH_INTERVAL_MS)

    return () => {
      clearInterval(timeout)
    }
  }, [])

  const columns = useMemo(
    () => {
      const cols = [
        {
          Header: _(t.created),
          accessor: 'eventMetadata.timestamp',
          disableSortBy: true,
          Cell: ({ value }) => <DateTooltip value={value} />,
        },
        {
          Header: _(t.type),
          accessor: 'commandType',
          disableSortBy: true,
          Cell: ({ value, row }) => {
            const {
              original: {
                auditContext: { correlationId },
                resourceId: { href } = {},
                content,
              },
            } = row
            const rowDeviceId =
              row?.original?.resourceId?.deviceId || row?.original?.deviceId

            if (!content && !href) {
              return <span className="no-wrap-text">{_(t[value])}</span>
            }

            return (
              <span
                className="no-wrap-text link"
                onClick={() =>
                  onViewClick({
                    deviceId: rowDeviceId,
                    href,
                    correlationId,
                    content,
                    commandType: value,
                  })
                }
              >
                {_(t[value])}
              </span>
            )
          },
        },
        {
          Header: _(t.resourceHref),
          accessor: 'resourceId.href',
          disableSortBy: true,
          Cell: ({ value }) => {
            return <span className="no-wrap-text">{value || '-'}</span>
          },
        },
        {
          Header: _(t.status),
          accessor: 'status',
          disableSortBy: true,
          Cell: ({ value, row }) => {
            const { validUntil } = row.original
            const { color, label } = getPendingCommandStatusColorAndLabel(
              value,
              validUntil,
              currentTime
            )

            if (!value) {
              return <Badge className={color}>{_(label)}</Badge>
            }

            return (
              <OverlayTrigger
                placement="top"
                overlay={<Tooltip className="plgd-tooltip">{value}</Tooltip>}
              >
                <Badge className={color}>{_(label)}</Badge>
              </OverlayTrigger>
            )
          },
        },
        {
          Header: _(t.validUntil),
          accessor: 'validUntil',
          disableSortBy: true,
          Cell: ({ value }) => {
            if (value === '0') return _(t.forever)

            return <DateTooltip value={value} />
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
                resourceId: { href } = {},
                status,
                validUntil,
              },
            } = row

            const rowDeviceId =
              row?.original?.resourceId?.deviceId || row?.original?.deviceId

            if (status || hasCommandExpired(validUntil, currentTime)) {
              return <div className="no-action" />
            }

            return (
              <div
                className="dropdown action-button"
                onClick={() =>
                  onCancelClick({ deviceId: rowDeviceId, href, correlationId })
                }
                title={_(t.cancel)}
              >
                <button className="dropdown-toggle btn btn-empty">
                  <i className="fas fa-times" />
                </button>
              </div>
            )
          },
          className: 'actions',
        },
      ]

      // Only show device id column when not on the device details
      if (!deviceId) {
        cols.splice(2, 0, {
          Header: _(t.deviceId),
          accessor: 'resourceId.deviceId',
          disableSortBy: true,
          Cell: ({ row }) => {
            return (
              <span className="no-wrap-text">
                {row?.original?.resourceId?.deviceId || row?.original?.deviceId}
              </span>
            )
          },
        })
      }

      return cols
    },
    [currentTime] // eslint-disable-line
  )

  return (
    <>
      <Table
        columns={columns}
        data={data || []}
        defaultSortBy={[
          {
            id: 'eventMetadata.timestamp',
            desc: true,
          },
        ]}
        autoFillEmptyRows
        defaultPageSize={
          embedded
            ? EMBEDDED_PENDING_COMMANDS_DEFAULT_PAGE_SIZE
            : PENDING_COMMANDS_DEFAULT_PAGE_SIZE
        }
        getColumnProps={column => {
          if (column.id === 'actions') {
            return { style: { textAlign: 'center' } }
          }

          return {}
        }}
        enablePagination={!embedded}
      />

      <PendingCommandDetailsModal
        {...detailsModalData}
        onClose={onCloseViewModal}
      />

      <ConfirmModal
        onConfirm={cancelCommand}
        show={!!confirmModalData}
        title={
          <>
            <i className="fas fa-times" />
            {_(t.cancelPendingCommand)}
          </>
        }
        body={_(t.doYouWantToCancelThisCommand)}
        confirmButtonText={_(t.yes)}
        cancelButtonText={_(t.no)}
        loading={canceling}
        onClose={onCloseCancelModal}
      />
    </>
  )
}

PendingCommandsList.propTypes = {
  onLoading: PropTypes.func,
  embedded: PropTypes.bool,
  deviceId: PropTypes.string,
}

PendingCommandsList.defaultProps = {
  onLoading: null,
  embedded: false,
  deviceId: null,
}
