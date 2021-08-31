import { useEffect, useState } from 'react'
import { useIntl } from 'react-intl'
import { toast } from 'react-toastify'

import { showSuccessToast } from '@/components/toast'
import { ConfirmModal } from '@/components/confirm-modal'
import { Layout } from '@/components/layout'
import { getApiErrorMessage } from '@/common/utils'
import { useIsMounted } from '@/common/hooks'
import { Emitter } from '@/common/services/emitter'
import { messages as menuT } from '@/components/menu/menu-i18n'

import {
  THINGS_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY,
  RESET_COUNTER,
} from './constants'
import { useThingsList } from './hooks'
import { ThingsList } from './_things-list'
import { ThingsListHeader } from './_things-list-header'
import { deleteThingsApi } from './rest'
import { handleDeleteDevicesErrors } from './utils'
import { messages as t } from './things-i18n'

export const ThingsListPage = () => {
  const { formatMessage: _ } = useIntl()
  const { data, loading, error, refresh } = useThingsList()
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
    if (error) {
      toast.error(getApiErrorMessage(error))
    }
  }, [error])

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
    Emitter.emit(THINGS_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY, RESET_COUNTER)
  }

  const deleteDevices = async () => {
    try {
      setDeleting(true)
      await deleteThingsApi(combinedSelectedDevices)

      if (isMounted.current) {
        showSuccessToast({
          title: _(t.thingsDeleted),
          message: _(t.thingsDeletedMessage),
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
      header={<ThingsListHeader loading={loading} refresh={handleRefresh} />}
    >
      <ThingsList
        data={data}
        selectedDevices={selectedDevices}
        setSelectedDevices={setSelectedDevices}
        loading={loadingOrDeleting}
        onDeleteClick={handleOpenDeleteModal}
        unselectRowsToken={unselectRowsToken}
      />

      <ConfirmModal
        onConfirm={deleteDevices}
        show={deleteModalOpen}
        title={
          <>
            <i className="fas fa-trash-alt" />
            {`${_(t.delete)} ${
              selectedDevicesCount > 1
                ? `${selectedDevicesCount} ${_(menuT.things)}`
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
