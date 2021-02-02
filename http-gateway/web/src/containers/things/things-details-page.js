import { useIntl } from 'react-intl'
import { useParams } from 'react-router-dom'
import classNames from 'classnames'

import { Layout } from '@/components/layout'
import { NotFoundPage } from '@/containers/not-found-page'
import { useApi } from '@/common/hooks'
import { useAppConfig } from '@/containers/app'
import { messages as menuT } from '@/components/menu/menu-i18n'

import { ThingsDetails } from './_things-details'
import { ThingsResourcesList } from './_things-resources-list'
import { thingsApiEndpoints } from './constants'
import { messages as t } from './things-i18n'

export const ThingsDetailsPage = props => {
  const { formatMessage: _ } = useIntl()
  const { id } = useParams()
  const { audience, httpGatewayAddress } = useAppConfig()

  const { data, loading, error } = useApi(
    `${httpGatewayAddress}${thingsApiEndpoints.THINGS}/${id}`,
    { audience }
  )

  if (error) {
    return (
      <NotFoundPage
        title={_(t.thingNotFound)}
        message={_(t.thingNotFoundMessage, { id })}
      />
    )
  }

  const deviceName = data?.device?.n
  const breadcrumbs = [
    {
      to: '/',
      label: _(menuT.dashboard),
    },
    {
      to: '/things',
      label: _(menuT.things),
    },
  ]

  if (deviceName) {
    breadcrumbs.push({ label: deviceName })
  }

  return (
    <Layout
      title={`${deviceName ? deviceName + ' | ' : ''}${_(menuT.things)}`}
      breadcrumbs={breadcrumbs}
      loading={loading}
    >
      <h2 className={classNames({ shimmering: loading })}>{deviceName}</h2>
      <ThingsDetails data={data} loading={loading} />

      <h2>{_(t.resources)}</h2>
      <ThingsResourcesList data={data?.links} />
    </Layout>
  )
}
