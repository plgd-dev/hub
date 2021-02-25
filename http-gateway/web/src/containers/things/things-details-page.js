import { useState } from 'react'
import { useIntl } from 'react-intl'
import { useParams } from 'react-router-dom'
import classNames from 'classnames'

import { Layout } from '@/components/layout'
import { NotFoundPage } from '@/containers/not-found-page'
import { useIsMounted } from '@/common/hooks'
import { messages as menuT } from '@/components/menu/menu-i18n'
import { showSuccessToast } from '@/components/toast'

import { ThingsDetails } from './_things-details'
import { ThingsResourcesList } from './_things-resources-list'
import { ThingsResourcesModal } from './_things-resources-modal'
import {
  thingsStatuses,
  defaultNewResource,
  resourceModalTypes,
  knownInterfaces,
} from './constants'
import {
  handleCreateResourceErrors,
  handleUpdateResourceErrors,
  handleFetchResourceErrors,
} from './utils'
import {
  getThingsResourcesApi,
  updateThingsResourceApi,
  createThingsResourceApi,
} from './rest'
import { useThingDetails } from './hooks'
import { messages as t } from './things-i18n'

export const ThingsDetailsPage = () => {
  const { formatMessage: _ } = useIntl()
  const { id } = useParams()
  const [resourceModalData, setResourceModalData] = useState(null)
  const [loadingResource, setLoadingResource] = useState(false)
  const [savingResource, setSavingResource] = useState(false)
  const isMounted = useIsMounted()
  const { data, loading, error } = useThingDetails(id)

  if (error) {
    return (
      <NotFoundPage
        title={_(t.thingNotFound)}
        message={_(t.thingNotFoundMessage, { id })}
      />
    )
  }

  const deviceStatus = data?.status
  const isOnline = thingsStatuses.ONLINE === deviceStatus
  const greyedOutClassName = classNames({
    'grayed-out': thingsStatuses.UNREGISTERED === deviceStatus,
  })
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

  // Fetches the resource and sets its values to the modal data, which opens the modal.
  const openUpdateModal = async ({ href, currentInterface = '' }) => {
    // If there is already a fetch for a resource, disable the next attempt for a fetch untill the previous fetch finishes
    if (loadingResource) {
      return
    }

    setLoadingResource(true)

    try {
      const {
        data: { if: ifs, rt, ...resourceData }, // exclude the if and rt
      } = await getThingsResourcesApi({ deviceId: id, href, currentInterface })

      if (isMounted.current) {
        setLoadingResource(false)

        // Retrieve the types and interfaces of this resource
        const { rt: types = [], if: interfaces = [] } =
          data?.links?.find?.(link => link.href === href) || {}

        // Setting the data and opening the modal
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

  // Fetches the resources supported types and sets its values to the modal data, which opens the modal.
  const openCreateModal = async href => {
    // If there is already a fetch for a resource, disable the next attempt for a fetch untill the previous fetch finishes
    if (loadingResource) {
      return
    }

    setLoadingResource(true)

    try {
      const {
        data: { rts: supportedTypes },
      } = await getThingsResourcesApi({
        deviceId: id,
        href,
        currentInterface: knownInterfaces.OIC_IF_BASELINE,
      })

      if (isMounted.current) {
        setLoadingResource(false)

        // Setting the data and opening the modal
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

  // Updates the resource through rest API
  const updateResource = async (
    { href, currentInterface = '' },
    resourceDataUpdate
  ) => {
    setSavingResource(true)

    try {
      await updateThingsResourceApi(
        { deviceId: id, href, currentInterface },
        resourceDataUpdate
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

  // Created a new resource through rest API
  const createResource = async (
    { href, currentInterface = '' },
    resourceDataCreate
  ) => {
    setSavingResource(true)

    try {
      await createThingsResourceApi(
        { deviceId: id, href, currentInterface },
        resourceDataCreate
      )

      if (isMounted.current) {
        showSuccessToast({
          title: _(t.resourceCreateSuccess),
          message: _(t.resourceWasCreated),
        })

        setSavingResource(false)
      }
    } catch (error) {
      if (error && isMounted.current) {
        handleCreateResourceErrors(error, isOnline, _)
        setSavingResource(false)
      }
    }
  }

  return (
    <Layout
      title={`${deviceName ? deviceName + ' | ' : ''}${_(menuT.things)}`}
      breadcrumbs={breadcrumbs}
      loading={loading || (!resourceModalData && loadingResource)}
    >
      <h2
        className={classNames(
          {
            shimmering: loading,
          },
          greyedOutClassName
        )}
      >
        {deviceName}
      </h2>
      <ThingsDetails data={data} loading={loading} />

      <h2 className={classNames(greyedOutClassName)}>{_(t.resources)}</h2>
      <ThingsResourcesList
        data={data?.links}
        onUpdate={openUpdateModal}
        onCreate={openCreateModal}
        deviceStatus={deviceStatus}
      />

      <ThingsResourcesModal
        {...resourceModalData}
        onClose={() => setResourceModalData(null)}
        fetchResource={openUpdateModal}
        updateResource={updateResource}
        createResource={createResource}
        retrieving={loadingResource}
        loading={savingResource}
        isDeviceOnline={isOnline}
        deviceId={id}
      />
    </Layout>
  )
}
