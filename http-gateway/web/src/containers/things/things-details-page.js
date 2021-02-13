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
import { getApiErrorMessage } from '@/common/utils'
import {
  showSuccessToast,
  showErrorToast,
  showWarningToast,
} from '@/components/toast'

import { ThingsDetails } from './_things-details'
import { ThingsResourcesList } from './_things-resources-list'
import { ThingsResourcesUpdateModal } from './_things-resources-update-modal'
import { thingsApiEndpoints, thingsStatuses, errorCodes } from './constants'
import { interfaceGetParam } from './utils'
import { messages as t } from './things-i18n'

export const ThingsDetailsPage = () => {
  const { formatMessage: _ } = useIntl()
  const { id } = useParams()
  const { httpGatewayAddress } = useAppConfig()
  const [resourceModalData, setResourceModalData] = useState(null)
  const [loadingResource, setLoadingResource] = useState(false)
  const [updatingResource, setUpdatingResource] = useState(false)
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

  const fetchResourceAndOpenModal = async ({ href, currentInterface = '' }) => {
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

        updateResourceData({
          href,
          resourceData: resourceData,
        })
      }
    } catch (error) {
      if (error && isMounted.current) {
        setLoadingResource(false)
        showErrorToast({
          title: _(t.resourceRetrieveError),
          message: getApiErrorMessage(error),
        })
      }
    }
  }

  const updateResource = async (
    { href, currentInterface = '' },
    resourceDataUpdate
  ) => {
    setUpdatingResource(true)

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

        setUpdatingResource(false)
      }
    } catch (error) {
      if (error && isMounted.current) {
        const errorMessage = getApiErrorMessage(error)

        if (
          !isOnline &&
          errorMessage?.includes?.(errorCodes.DEADLINE_EXCEEDED)
        ) {
          // Device update went through, but it will be applied once the device comes online
          showWarningToast({
            title: _(t.resourceUpdateSuccess),
            message: _(t.resourceWasUpdatedOffline),
          })
        } else if (errorMessage?.includes?.(errorCodes.INVALID_ARGUMENT)) {
          // JSON validation error
          showErrorToast({
            title: _(t.resourceUpdateError),
            message: _(t.invalidArgument),
          })
        } else {
          showErrorToast({
            title: _(t.resourceUpdateError),
            message: errorMessage,
          })
        }

        setUpdatingResource(false)
      }
    }
  }

  const updateResourceData = ({ href, resourceData }) => {
    // Retrieve the types and interfaces of this resource
    const { rt: types, if: interfaces } =
      data?.links?.find?.(link => link.di === id) || {}

    setResourceModalData({
      data: {
        di: id,
        href,
        types,
        interfaces,
      },
      resourceData,
    })
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
        updateResource={updateResource}
        retrieving={loadingResource}
        updating={updatingResource}
        isDeviceOnline={isOnline}
      />
    </Layout>
  )
}
