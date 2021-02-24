import { useEffect } from 'react'
import { useIntl } from 'react-intl'
import { toast } from 'react-toastify'

import { Layout } from '@/components/layout'
import { getApiErrorMessage } from '@/common/utils'
import { messages as menuT } from '@/components/menu/menu-i18n'

import { useThingsList } from './hooks'
import { ThingsList } from './_things-list'

export const ThingsListPage = () => {
  const { formatMessage: _ } = useIntl()
  const { data, loading, error } = useThingsList()

  useEffect(
    () => {
      if (error) {
        toast.error(getApiErrorMessage(error))
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
      <ThingsList data={data} />
    </Layout>
  )
}
