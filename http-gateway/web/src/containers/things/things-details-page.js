import { useState } from 'react'
import { useIntl } from 'react-intl'
import { useParams } from 'react-router-dom'
import classNames from 'classnames'
import { toast } from 'react-toastify'

import { Layout } from '@/components/layout'
import { NotFoundPage } from '@/containers/not-found-page'
import { useApi, useIsMounted } from '@/common/hooks'
import { useAppConfig } from '@/containers/app'
import { messages as menuT } from '@/components/menu/menu-i18n'
import { fetchApi } from '@/common/services'

import { ThingsDetails } from './_things-details'
import { ThingsResourcesList } from './_things-resources-list'
import { ThingsResourcesUpdateModal } from './_things-resources-update-modal'
import { thingsApiEndpoints, thingsStatuses } from './constants'
import { messages as t } from './things-i18n'

export const ThingsDetailsPage = () => {
  const { formatMessage: _ } = useIntl()
  const { id } = useParams()
  const { audience, httpGatewayAddress } = useAppConfig()
  const [resourceModalData, setResourceModalData] = useState(null)
  const [loadingResource, setLoadingResource] = useState(false)
  const isMounted = useIsMounted()

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

  const isOnline = thingsStatuses.ONLINE === data?.status
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

  const fetchResourceAndOpenModal = async ({
    di,
    href,
    currentInterface = '',
  }) => {
    // If there is already a fetch for a resource, disable the next attempt for a fetch untill the previous fetch finishes
    if (loadingResource) {
      return
    }

    setLoadingResource(true)

    try {
      const interfaceGetParam = currentInterface
        ? `interface=${currentInterface}`
        : ''
      const {
        data: { if: ifs, rt, ...resourceData }, // exclude the if and rt
      } = await fetchApi(
        `${httpGatewayAddress}${
          thingsApiEndpoints.THINGS
        }/${id}${href}?${interfaceGetParam}`,
        { audience }
      )

      if (isMounted.current) {
        setLoadingResource(false)

        // Retrieve the types and interfaces of this resource
        const { rt: types, if: interfaces } =
          data?.links?.find?.(link => link.di === di) || {}

        setResourceModalData({
          data: {
            di,
            href,
            types,
            interfaces,
          },
          resourceData,
        })
      }
    } catch (error) {
      if (error && isMounted.current) {
        setLoadingResource(false)
        toast.error(error?.response?.data?.err || error?.message)
      }
    }
  }

  return (
    <Layout
      title={`${deviceName ? deviceName + ' | ' : ''}${_(menuT.things)}`}
      breadcrumbs={breadcrumbs}
      loading={loading || (!resourceModalData && loadingResource)}
    >
      <h2 className={classNames({ shimmering: loading })}>{deviceName}</h2>
      <ThingsDetails data={data} loading={loading} />

      <h2>{_(t.resources)}</h2>
      <ThingsResourcesList
        data={data?.links}
        onClick={fetchResourceAndOpenModal}
      />

      <ThingsResourcesUpdateModal
        {...resourceModalData}
        onClose={() => setResourceModalData(null)}
        fetchResource={fetchResourceAndOpenModal}
        retrieving={loadingResource}
        isDeviceOnline={isOnline}
      />
    </Layout>
  )
}
