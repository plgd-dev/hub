import { useMemo } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'

import { Badge } from '@/components/badge'
import { Table } from '@/components/table'
import { ActionButton } from '@/components/action-button'

import { RESOURCES_DEFAULT_PAGE_SIZE } from './constants'
import { canCreateResource } from './utils'
import { thingResourceShape } from './shapes'
import { messages as t } from './things-i18n'

export const ThingsResourcesList = ({ data, onUpdate, onCreate }) => {
  const { formatMessage: _ } = useIntl()

  const columns = useMemo(
    () => [
      {
        Header: _(t.location),
        accessor: 'href',
        Cell: ({ value, row }) => {
          const {
            original: { di, href },
          } = row
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
              menuProps={{ align: 'right' }}
              items={[
                {
                  onClick: () => onCreate(href),
                  label: _(t.create),
                  icon: 'fa-plus',
                  hidden: !canCreateResource(interfaces) || true, // temporary disabled
                },
                {
                  onClick: () => onUpdate({ di, href }),
                  label: _(t.update),
                  icon: 'fa-pen',
                },
                {
                  onClick: () => console.log('helo'),
                  label: _(t.delete),
                  icon: 'fa-trash-alt',
                  hidden: true,
                },
              ]}
            >
              <i className="fas fa-ellipsis-h" />
            </ActionButton>
          )
        },
      },
    ],
    [onUpdate, onCreate] //eslint-disable-line
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
    />
  )
}

ThingsResourcesList.propTypes = {
  data: PropTypes.arrayOf(thingResourceShape),
  onUpdate: PropTypes.func.isRequired,
}

ThingsResourcesList.defaultProps = {
  data: null,
}
