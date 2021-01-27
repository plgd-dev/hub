import { useEffect, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { Link } from 'react-router-dom'
import { toast } from 'react-toastify'

import { Layout } from '@/components/layout'
import { Badge } from '@/components/badge'
import { Table } from '@/components/table'
import { useApi } from '@/common/hooks'
import { messages as menuT } from '@/components/menu/menu-i18n'

import { thingsStatuses } from './constants'
import { messages as t } from './things-i18n'

export const ThingsListPage = () => {
  const { formatMessage: _ } = useIntl()

  const { data, loading, error } = useApi(
    'https://api.try.plgd.cloud/api/v1/devices',
    { audience: 'https://try.plgd.cloud' }
  )

  const columns = useMemo(
    () => [
      {
        Header: _(t.name),
        accessor: 'device.n',
        Cell: ({ value, row }) => (
          <Link to={`/things/${row.values?.['device.di']}`}>{value}</Link>
        ),
      },
      {
        Header: 'ID',
        accessor: 'device.di',
      },
      {
        Header: _(t.status),
        accessor: 'status',
        Cell: ({ value }) => {
          const isOnline = thingsStatuses.ONLINE === value
          return (
            <Badge className={isOnline ? 'green' : 'red'}>
              {isOnline ? _(t.online) : _(t.offline)}
            </Badge>
          )
        },
      },
    ],
    [] //eslint-disable-line
  )

  useEffect(
    () => {
      if (error) {
        toast.error(error?.response?.data?.err || error?.message)
      }
    },
    [error]
  )

  return (
    <Layout
      title={_(menuT.things)}
      breadcrumbs={[
        {
          to: '/',
          label: _(menuT.dashboard),
        },
        {
          label: _(menuT.things),
        },
      ]}
      loading={loading}
      header={<div />}
    >
      <Table
        columns={columns}
        data={data || []}
        defaultSortBy={[
          {
            id: 'device.n',
            desc: false,
          },
        ]}
        autoFillEmptyRows
      />
    </Layout>
  )
}
