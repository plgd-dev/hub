import { FC, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { Link, useHistory } from 'react-router-dom'
import classNames from 'classnames'
import Button from '@shared-ui/components/new/Button'
import Badge from '@shared-ui/components/new/Badge'
import Table from '@shared-ui/components/new/Table'
import { IndeterminateCheckbox } from '@/components/checkbox'
import DevicesListActionButton from '../DevicesListActionButton'
import {
  devicesStatuses,
  DEVICES_DEFAULT_PAGE_SIZE,
  NO_DEVICE_NAME,
} from '../../constants'
import { messages as t } from '../../Devices.i18n'
import { Props, defaultProps } from './DevicesList.types'

const { ONLINE, UNREGISTERED } = devicesStatuses

export const DevicesList: FC<Props> = props => {
  const {
    data,
    loading,
    setSelectedDevices,
    selectedDevices,
    onDeleteClick,
    unselectRowsToken,
  } = { ...defaultProps, ...props }
  const { formatMessage: _ } = useIntl()
  const history = useHistory()

  const columns = useMemo(
    () => [
      {
        id: 'selection',
        accessor: 'selection',
        disableSortBy: true,
        style: { width: '36px' },
        Header: ({ getToggleAllRowsSelectedProps }: any) => (
          <IndeterminateCheckbox {...getToggleAllRowsSelectedProps()} />
        ),
        Cell: ({ row }: any) => {
          if (row.original?.metadata?.connection?.status === UNREGISTERED) {
            return null
          }

          return <IndeterminateCheckbox {...row.getToggleRowSelectedProps()} />
        },
      },
      {
        Header: _(t.name),
        accessor: 'name',
        Cell: ({ value, row }: any) => {
          const deviceName = value || NO_DEVICE_NAME

          if (row.original?.metadata?.connection?.status === UNREGISTERED) {
            return <span>{deviceName}</span>
          }
          return (
            <Link to={`/devices/${row.original?.id}`}>
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
        Cell: ({ value }: { value: string | number }) => {
          return <span className="no-wrap-text">{value}</span>
        },
      },
      {
        Header: _(t.twinSynchronization),
        accessor: 'metadata.twinEnabled',
        Cell: ({ value }: { value: string | number }) => {
          const isTwinEnabled = value
          return (
            <Badge className={isTwinEnabled ? 'green' : 'red'}>
              {isTwinEnabled ? _(t.enabled) : _(t.disabled)}
            </Badge>
          )
        },
      },
      {
        Header: _(t.status),
        accessor: 'metadata.connection.status',
        style: { width: '120px' },
        Cell: ({ value }: { value: string | number }) => {
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
        Cell: ({ row }: any) => {
          const {
            original: { id },
          } = row
          return (
            <DevicesListActionButton
              deviceId={id}
              onView={deviceId => history.push(`/devices/${deviceId}`)}
              onDelete={onDeleteClick}
            />
          )
        },
        className: 'actions',
      },
    ],
    [] // eslint-disable-line
  )

  const validData = (data: any) => (!data || data[0] === undefined ? [] : data)

  return (
    <Table
      className="with-selectable-rows"
      columns={columns}
      data={validData(data)}
      defaultSortBy={[
        {
          id: 'name',
          desc: false,
        },
      ]}
      autoFillEmptyRows
      defaultPageSize={DEVICES_DEFAULT_PAGE_SIZE}
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
          onClick={() => onDeleteClick()}
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

DevicesList.displayName = 'DevicesList'
DevicesList.defaultProps = defaultProps

export default DevicesList
