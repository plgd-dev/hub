import { useEffect, useState } from 'react'
import { useIntl } from 'react-intl'
import { toast } from 'react-toastify'

import { showSuccessToast } from '@/components/toast'
import { ConfirmModal } from '@/components/confirm-modal'
import { Layout } from '@/components/layout'
import { getApiErrorMessage } from '@/common/utils'
import { useIsMounted } from '@/common/hooks'
import { Emitter } from '@/common/services/emitter'
import { PendingCommandsExpandableList } from '@/containers/pending-commands'
import { messages as menuT } from '@/components/menu/menu-i18n'

import {
  DEVICES_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY,
  RESET_COUNTER,
} from './constants'
import { useDevicesList } from './hooks'
import { DevicesList } from './_devices-list'
import { DevicesListHeader } from './_devices-list-header'
import { deleteDevicesApi } from './rest'
import { handleDeleteDevicesErrors, sleep } from './utils'
import { messages as t } from './devices-i18n'

export const DevicesListPage = () => {
  const { formatMessage: _ } = useIntl()
  const { data, loading, error: deviceError, refresh } = useDevicesList()
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  const [selectedDevices, setSelectedDevices] = useState([])
  const [singleDevice, setSingleDevice] = useState(null)
  const [deleting, setDeleting] = useState(false)
  const [unselectRowsToken, setUnselectRowsToken] = useState(1)
  const isMounted = useIsMounted()
  const combinedSelectedDevices = singleDevice
    ? [singleDevice]
    : selectedDevices

  useEffect(() => {
    if (deviceError) {
      toast.error(getApiErrorMessage(deviceError))
    }
  }, [deviceError])

  const handleOpenDeleteModal = deviceId => {
    if (typeof deviceId === 'string') {
      setSingleDevice(deviceId)
    }

    setDeleteModalOpen(true)
  }

  const handleCloseDeleteModal = () => {
    setSingleDevice(null)
    setDeleteModalOpen(false)
  }

  const handleRefresh = () => {
    refresh()

    // Unselect all rows from the table
    setUnselectRowsToken(prevValue => prevValue + 1)

    // Reset the counter on the Refresh button
    Emitter.emit(DEVICES_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY, RESET_COUNTER)
  }

  const deleteDevices = async () => {
    try {
      setDeleting(true)
      await deleteDevicesApi(combinedSelectedDevices)
      await sleep(200)

      if (isMounted.current) {
        showSuccessToast({
          title: _(t.devicesDeleted),
          message: _(t.devicesDeletedMessage),
        })

        setDeleting(false)
        setDeleteModalOpen(false)
        setSingleDevice(null)
        setSelectedDevices([])
        handleCloseDeleteModal()
        handleRefresh()
      }
    } catch (error) {
      setDeleting(false)
      handleDeleteDevicesErrors(error, _)
    }
  }

  const selectedDevicesCount = combinedSelectedDevices.length
  const selectedDeviceName =
    selectedDevicesCount === 1 && data
      ? data?.find?.(d => d.id === combinedSelectedDevices[0])?.name
      : null

  const loadingOrDeleting = loading || deleting

  return (
    <Layout
      title={_(menuT.devices)}
      breadcrumbs={[
        {
          label: _(menuT.devices),
        },
      ]}
      loading={loading}
      header={<DevicesListHeader loading={loading} refresh={handleRefresh} />}
    >
      <DevicesList
        data={data}
        selectedDevices={selectedDevices}
        setSelectedDevices={setSelectedDevices}
        loading={loadingOrDeleting}
        onDeleteClick={handleOpenDeleteModal}
        unselectRowsToken={unselectRowsToken}
      />

      <PendingCommandsExpandableList />

      <ConfirmModal
        onConfirm={deleteDevices}
        show={deleteModalOpen}
        title={
          <>
            <i className="fas fa-trash-alt" />
            {`${_(t.delete)} ${
              selectedDevicesCount > 1
                ? `${selectedDevicesCount} ${_(menuT.devices)}`
                : selectedDeviceName
            }`}
          </>
        }
        body={
          selectedDevicesCount > 1
            ? _(t.deleteDevicesMessage, { count: selectedDevicesCount })
            : _(t.deleteDeviceMessage)
        }
        confirmButtonText={_(t.delete)}
        loading={loadingOrDeleting}
        onClose={handleCloseDeleteModal}
      >
        {_(t.delete)}
      </ConfirmModal>
    </Layout>
  )
}
