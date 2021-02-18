import { useState } from 'react'
import { useIntl } from 'react-intl'
import { useParams } from 'react-router-dom'
import classNames from 'classnames'

import { Layout } from '@/components/layout'
import { NotFoundPage } from '@/containers/not-found-page'
import { useApi, useIsMounted } from '@/common/hooks'
import { useAppConfig } from '@/containers/app'
import { messages as menuT } from '@/components/menu/menu-i18n'
import { fetchApi } from '@/common/services'
import { showSuccessToast } from '@/components/toast'

import { ThingsDetails } from './_things-details'
import { ThingsResourcesList } from './_things-resources-list'
import { ThingsResourcesModal } from './_things-resources-modal'
import {
  thingsApiEndpoints,
  thingsStatuses,
  defaultNewResource,
  resourceModalTypes,
  knownInterfaces,
} from './constants'
import {
  interfaceGetParam,
  handleUpdateResourceErrors,
  handleFetchResourceErrors,
} from './utils'
import { messages as t } from './things-i18n'

export const ThingsDetailsPage = () => {
  const { formatMessage: _ } = useIntl()
  const { id } = useParams()
  const { httpGatewayAddress } = useAppConfig()
  const [resourceModalData, setResourceModalData] = useState(null)
  const [loadingResource, setLoadingResource] = useState(false)
  const [savingResource, setSavingResource] = useState(false)
  const isMounted = useIsMounted()

  const { data, loading, error } = useApi(
    `${httpGatewayAddress}${thingsApiEndpoints.THINGS}/${id}`
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

  const fetchResourceAndOpenUpdateModal = async ({
    href,
    currentInterface = '',
  }) => {
    // If there is already a fetch for a resource, disable the next attempt for a fetch untill the previous fetch finishes
    if (loadingResource) {
      return
    }

    setLoadingResource(true)

    try {
      const {
        data: { if: ifs, rt, ...resourceData }, // exclude the if and rt
      } = await fetchApi(
        `${httpGatewayAddress}${
          thingsApiEndpoints.THINGS
        }/${id}${href}${interfaceGetParam(currentInterface)}`
      )

      if (isMounted.current) {
        setLoadingResource(false)

        // Retrieve the types and interfaces of this resource
        const { rt: types = [], if: interfaces = [] } =
          data?.links?.find?.(link => link.href === href) || {}

        setResourceModalData({
          data: {
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
        handleFetchResourceErrors(error)
      }
    }
  }

  const updateResource = async (
    { href, currentInterface = '' },
    resourceDataUpdate
  ) => {
    setSavingResource(true)

    try {
      await fetchApi(
        `${httpGatewayAddress}${
          thingsApiEndpoints.THINGS
        }/${id}${href}${interfaceGetParam(currentInterface)}`,
        { method: 'PUT', body: resourceDataUpdate }
      )

      if (isMounted.current) {
        showSuccessToast({
          title: _(t.resourceUpdateSuccess),
          message: _(t.resourceWasUpdated),
        })

        setSavingResource(false)
      }
    } catch (error) {
      if (error && isMounted.current) {
        handleUpdateResourceErrors(error, isOnline, _)
        setSavingResource(false)
      }
    }
  }

  const openCreateModal = async href => {
    // If there is already a fetch for a resource, disable the next attempt for a fetch untill the previous fetch finishes
    if (loadingResource) {
      return
    }

    setLoadingResource(true)

    try {
      const {
        data: { rts: supportedTypes },
      } = await fetchApi(
        `${httpGatewayAddress}${
          thingsApiEndpoints.THINGS
        }/${id}${href}${interfaceGetParam(knownInterfaces.OIC_IF_BASELINE)}`
      )

      if (isMounted.current) {
        setLoadingResource(false)

        setResourceModalData({
          data: {
            href,
            types: supportedTypes,
          },
          resourceData: {
            ...defaultNewResource,
            rt: supportedTypes,
          },
          type: resourceModalTypes.CREATE_RESOURCE,
        })
      }
    } catch (error) {
      if (error && isMounted.current) {
        setLoadingResource(false)
        handleFetchResourceErrors(error)
      }
    }
  }

  // const createResource = async (
  //   { href, currentInterface = '' },
  //   resourceDataUpdate
  // ) => {
  //   setSavingResource(true)

  //   try {
  //     await fetchApi(
  //       `${httpGatewayAddress}${
  //         thingsApiEndpoints.THINGS
  //       }/${id}${href}${interfaceGetParam(currentInterface)}`,
  //       { method: 'PUT', body: resourceDataUpdate }
  //     )

  //     if (isMounted.current) {
  //       showSuccessToast({
  //         title: _(t.resourceUpdateSuccess),
  //         message: _(t.resourceWasUpdated),
  //       })

  //       setSavingResource(false)
  //     }
  //   } catch (error) {
  //     if (error && isMounted.current) {
  //       handleUpdateResourceErrors(error, isOnline, _)
  //       setSavingResource(false)
  //     }
  //   }
  // }

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
        onUpdate={fetchResourceAndOpenUpdateModal}
        onCreate={openCreateModal}
      />

      <ThingsResourcesModal
        {...resourceModalData}
        onClose={() => setResourceModalData(null)}
        fetchResource={fetchResourceAndOpenUpdateModal}
        updateResource={updateResource}
        retrieving={loadingResource}
        loading={savingResource}
        isDeviceOnline={isOnline}
        deviceId={id}
      />
    </Layout>
  )
}
