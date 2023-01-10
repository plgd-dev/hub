import { useEffect, useState } from 'react'
import { useIntl } from 'react-intl'
import { useParams } from 'react-router-dom'
import classNames from 'classnames'

import { history } from '@/store'
import { ConfirmModal } from '@/components/confirm-modal'
import { Layout } from '@/components/layout'
import { NotFoundPage } from '@/containers/not-found-page'
import { useIsMounted } from '@/common/hooks'
import { messages as menuT } from '@/components/menu/menu-i18n'
import { showSuccessToast } from '@/components/toast'
import { useAppConfig } from '@/containers/App'
import { PendingCommandsExpandableList } from '@/containers/pending-commands'

import { DevicesDetails } from './_devices-details'
import { DevicesResources } from '../Resources/_devices-resources'
import { DevicesDetailsHeader } from './_devices-details-header'
import { DevicesDetailsTitle } from './_devices-details-title'
import { DevicesResourcesModal } from '../Resources/_devices-resources-modal'
import { CommanTimeoutControl } from '../_command-timeout-control'
import {
  devicesStatuses,
  defaultNewResource,
  resourceModalTypes,
  NO_DEVICE_NAME,
} from '../constants'
import {
  handleCreateResourceErrors,
  handleUpdateResourceErrors,
  handleFetchResourceErrors,
  handleDeleteResourceErrors,
  handleTwinSynchronizationErrors,
} from '../utils'
import {
  getDevicesResourcesApi,
  updateDevicesResourceApi,
  createDevicesResourceApi,
  deleteDevicesResourceApi,
  updateDeviceTwinSynchronizationApi,
} from '../rest'
import { useDeviceDetails, useDevicesResources } from '../hooks'
import { messages as t } from '../devices-i18n'

import './devices-details.scss'

export const DevicesDetailsPage = () => {
  const { formatMessage: _ } = useIntl()
  const { id, href: hrefParam } = useParams()
  const [twinSyncLoading, setTwinSyncLoading] = useState(false)
  const [resourceModalData, setResourceModalData] = useState(null)
  const [loadingResource, setLoadingResource] = useState(false)
  const [savingResource, setSavingResource] = useState(false)
  const [deleteResourceHref, setDeleteResourceHref] = useState()
  const {
    wellKnownConfig: { defaultCommandTimeToLive },
  } = useAppConfig()
  const [ttl, setTtl] = useState(defaultCommandTimeToLive)
  const [ttlHasError, setTtlHasError] = useState(false)
  const isMounted = useIsMounted()
  const { data, updateData, loading, error: deviceError } = useDeviceDetails(id)
  const {
    data: resourcesData,
    loading: loadingResources,
    error: resourcesError,
  } = useDevicesResources(id)

  const isTwinEnabled = data?.metadata?.twinEnabled
  const resources = resourcesData?.[0]?.resources || []

  // Open the resource modal when href is present
  useEffect(
    () => {
      if (hrefParam && !loading && !loadingResources) {
        openUpdateModal({ href: `/${hrefParam}` })
      }
    },
    [hrefParam, loading, loadingResources] // eslint-disable-line
  )

  if (deviceError) {
    return (
      <NotFoundPage
        title={_(t.deviceNotFound)}
        message={_(t.deviceNotFoundMessage, { id })}
      />
    )
  }

  if (resourcesError) {
    return (
      <NotFoundPage
        title={_(t.deviceResourcesNotFound)}
        message={_(t.deviceResourcesNotFoundMessage, { id })}
      />
    )
  }

  const deviceStatus = data?.metadata?.connection?.status
  const isOnline = devicesStatuses.ONLINE === deviceStatus
  const isUnregistered = devicesStatuses.UNREGISTERED === deviceStatus
  const greyedOutClassName = classNames({
    'grayed-out': isUnregistered,
  })
  const deviceName = data?.name || NO_DEVICE_NAME
  const breadcrumbs = [
    {
      to: '/',
      label: _(menuT.devices),
    },
  ]

  if (deviceName) {
    breadcrumbs.push({ label: deviceName })
  }

  // Fetches the resource and sets its values to the modal data, which opens the modal.
  const openUpdateModal = async ({ href, currentInterface = '' }) => {
    // If there is already a fetch for a resource, disable the next attempt for a fetch until the previous fetch finishes
    if (loadingResource) {
      return
    }

    setLoadingResource(true)

    try {
      const {
        data: { data: { content: { if: ifs, rt, ...resourceData } = {} } = {} }, // exclude the if and rt
      } = await getDevicesResourcesApi({ deviceId: id, href, currentInterface })

      if (isMounted.current) {
        setLoadingResource(false)

        // Retrieve the types and interfaces of this resource
        const { resourceTypes: types = [], interfaces = [] } =
          resources?.find?.(link => link.href === href) || {}

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
        handleFetchResourceErrors(error, _)
      }
    }
  }

  // Fetches the resources supported types and sets its values to the modal data, which opens the modal.
  const openCreateModal = async href => {
    // If there is already a fetch for a resource, disable the next attempt for a fetch until the previous fetch finishes
    if (loadingResource) {
      return
    }

    setLoadingResource(true)

    try {
      const { data: deviceData } = await getDevicesResourcesApi({
        deviceId: id,
        href,
      })
      const supportedTypes = deviceData?.data?.content?.rts || []

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
        handleFetchResourceErrors(error, _)
      }
    }
  }

  const openDeleteModal = href => {
    setDeleteResourceHref(href)
  }

  const closeDeleteModal = () => {
    setDeleteResourceHref(null)
  }

  // Updates the resource through rest API
  const updateResource = async (
    { href, currentInterface = '' },
    resourceDataUpdate
  ) => {
    setSavingResource(true)

    try {
      await updateDevicesResourceApi(
        { deviceId: id, href, currentInterface, ttl },
        resourceDataUpdate
      )

      if (isMounted.current) {
        showSuccessToast({
          title: _(t.resourceUpdateSuccess),
          message: _(t.resourceWasUpdated),
        })

        handleCloseUpdateModal()
        setSavingResource(false)
      }
    } catch (error) {
      if (error && isMounted.current) {
        handleUpdateResourceErrors(error, { id, href }, _)
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
      await createDevicesResourceApi(
        { deviceId: id, href, currentInterface, ttl },
        resourceDataCreate
      )

      if (isMounted.current) {
        showSuccessToast({
          title: _(t.resourceCreateSuccess),
          message: _(t.resourceWasCreated),
        })

        setResourceModalData(null) // close modal
        setSavingResource(false)
      }
    } catch (error) {
      if (error && isMounted.current) {
        handleCreateResourceErrors(error, { id, href }, _)
        setSavingResource(false)
      }
    }
  }

  const deleteResource = async () => {
    setLoadingResource(true)

    try {
      await deleteDevicesResourceApi({
        deviceId: id,
        href: deleteResourceHref,
        ttl,
      })

      if (isMounted.current) {
        showSuccessToast({
          title: _(t.resourceDeleteSuccess),
          message: _(t.resourceWasDeleted),
        })

        setLoadingResource(false)
        closeDeleteModal()
      }
    } catch (error) {
      if (error && isMounted.current) {
        handleDeleteResourceErrors(error, { id, href: deleteResourceHref }, _)
        setLoadingResource(false)
        closeDeleteModal()
      }
    }
  }

  // Handler which cleans up the resource modal data and updates the URL
  const handleCloseUpdateModal = () => {
    setResourceModalData(null)

    if (hrefParam) {
      // Remove the href from the URL when the update modal is closed
      history.replace(window.location.pathname.replace(`/${hrefParam}`, ''))
    }
  }

  // Update the device name in the data object
  const updateDeviceNameInData = name => {
    updateData({
      ...data,
      name,
    })
  }

  // Handler for setting the twin synchronization on a device
  const setTwinSynchronization = async () => {
    setTwinSyncLoading(true)

    try {
      const setSync = isTwinEnabled ? false : true
      await updateDeviceTwinSynchronizationApi(id, setSync)

      if (isMounted.current) {
        setTwinSyncLoading(false)
      }
    } catch (error) {
      if (error && isMounted.current) {
        handleTwinSynchronizationErrors(error, _)
        setTwinSyncLoading(false)
      }
    }
  }

  return (
    <Layout
      title={`${deviceName ? deviceName + ' | ' : ''}${_(menuT.devices)}`}
      breadcrumbs={breadcrumbs}
      loading={
        loading || (!resourceModalData && loadingResource) || twinSyncLoading
      }
      header={
        <DevicesDetailsHeader
          deviceId={id}
          deviceName={deviceName}
          isUnregistered={isUnregistered}
        />
      }
    >
      <DevicesDetailsTitle
        className={classNames(
          {
            shimmering: loading,
          },
          greyedOutClassName
        )}
        updateDeviceName={updateDeviceNameInData}
        loading={loading}
        isOnline={isOnline}
        deviceName={deviceName}
        deviceId={id}
        links={resources}
        ttl={ttl}
      />
      <DevicesDetails
        data={data}
        loading={loading}
        twinSyncLoading={twinSyncLoading}
        setTwinSynchronization={setTwinSynchronization}
      />

      <PendingCommandsExpandableList deviceId={id} />

      <DevicesResources
        data={resources}
        onUpdate={openUpdateModal}
        onCreate={openCreateModal}
        onDelete={openDeleteModal}
        deviceStatus={deviceStatus}
        loading={loadingResource}
        deviceId={id}
      />

      <DevicesResourcesModal
        {...resourceModalData}
        onClose={handleCloseUpdateModal}
        fetchResource={openUpdateModal}
        updateResource={updateResource}
        createResource={createResource}
        retrieving={loadingResource}
        loading={savingResource}
        isDeviceOnline={isOnline}
        isUnregistered={isUnregistered}
        deviceId={id}
        deviceName={deviceName}
        confirmDisabled={ttlHasError}
        ttlControl={
          <CommanTimeoutControl
            defaultValue={ttl}
            defaultTtlValue={defaultCommandTimeToLive}
            onChange={setTtl}
            disabled={loadingResource || savingResource}
            ttlHasError={ttlHasError}
            onTtlHasError={setTtlHasError}
          />
        }
      />

      <ConfirmModal
        onConfirm={deleteResource}
        show={!!deleteResourceHref}
        title={
          <>
            <i className="fas fa-trash-alt" />
            {`${_(t.delete)} ${deleteResourceHref}`}
          </>
        }
        body={
          <>
            {_(t.deleteResourceMessage)}
            <CommanTimeoutControl
              defaultValue={ttl}
              defaultTtlValue={defaultCommandTimeToLive}
              onChange={setTtl}
              disabled={loadingResource}
              ttlHasError={ttlHasError}
              onTtlHasError={setTtlHasError}
              isDelete
            />
          </>
        }
        confirmButtonText={_(t.delete)}
        loading={loadingResource}
        onClose={closeDeleteModal}
        confirmDisabled={ttlHasError}
      >
        {_(t.delete)}
      </ConfirmModal>
    </Layout>
  )
}
