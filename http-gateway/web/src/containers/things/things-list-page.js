import { useEffect } from 'react'
import { useIntl } from 'react-intl'
import { toast } from 'react-toastify'

import { Layout } from '@/components/layout'
import { useApi } from '@/common/hooks'
import { useAppConfig } from '@/containers/app'
import { messages as menuT } from '@/components/menu/menu-i18n'

import { thingsApiEndpoints } from './constants'
import { ThingsList } from './_things-list'

export const ThingsListPage = () => {
  const { formatMessage: _ } = useIntl()
  const { audience, apiEndpointUrl } = useAppConfig()

  const { data, loading, error } = useApi(
    `${apiEndpointUrl}${thingsApiEndpoints.THINGS}`,
    { audience }
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
      <ThingsList data={data} />
    </Layout>
  )
}
