import { useMemo } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'
import { Link } from 'react-router-dom'

import { Badge } from '@/components/badge'
import { Table } from '@/components/table'

import { thingResourceShape } from './shapes'
import { messages as t } from './things-i18n'

export const ThingsResourcesList = ({ data }) => {
  const { formatMessage: _ } = useIntl()

  const columns = useMemo(
    () => [
      {
        Header: _(t.location),
        accessor: 'href',
        Cell: ({ value, row }) => (
          <Link to={`/things/${row.original?.di}${row.original?.href}`}>
            {value}
          </Link>
        ),
        style: { width: '50%' },
      },
      {
        Header: _(t.type),
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
    [] //eslint-disable-line
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
      defaultPageSize={5}
      autoFillEmptyRows
    />
  )
}

ThingsResourcesList.propTypes = {
  data: PropTypes.arrayOf(thingResourceShape),
}

ThingsResourcesList.defaultProps = {
  data: null,
}
