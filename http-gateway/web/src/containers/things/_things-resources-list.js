import { useMemo } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'

import { Badge } from '@/components/badge'
import { Table } from '@/components/table'

import { RESOURCES_DEFAULT_PAGE_SIZE } from './constants'
import { thingResourceShape } from './shapes'
import { messages as t } from './things-i18n'

export const ThingsResourcesList = ({ data, onClick }) => {
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
            <span className="link" onClick={() => onClick({ di, href })}>
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
    ],
    [onClick] //eslint-disable-line
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
  onClick: PropTypes.func.isRequired,
}

ThingsResourcesList.defaultProps = {
  data: null,
}
