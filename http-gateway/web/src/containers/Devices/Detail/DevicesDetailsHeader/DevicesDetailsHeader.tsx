import { FC, useEffect, useRef, useState } from 'react'
import { useIntl } from 'react-intl'
import { useSelector, useDispatch } from 'react-redux'
import classNames from 'classnames'
import { useHistory } from 'react-router-dom'
import { Props } from './DevicesDetailsHeader.types'

import { showSuccessToast } from '@shared-ui/components/new/Toast'
import ConfirmModal from '@shared-ui/components/new/ConfirmModal'
import Button from '@shared-ui/components/new/Button'
import { WebSocketEventClient, eventFilters } from '@shared-ui/common/services'
import Switch from '@shared-ui/components/new/Switch'
import { useIsMounted } from '@shared-ui/common/hooks'
import {
  getDeviceNotificationKey,
  getResourceRegistrationNotificationKey,
  handleDeleteDevicesErrors,
  sleep,
} from '../../utils'
import { isNotificationActive, toggleActiveNotification } from '../../slice'
import { deviceResourceRegistrationListener } from '../../websockets'
import { deleteDevicesApi } from '../../rest'
import { messages as t } from '../../devices-i18n'

const DevicesDetailsHeader: FC<Props> = ({
  deviceId,
  deviceName,
  isUnregistered,
}) => {
  const { formatMessage: _ } = useIntl()
  const dispatch = useDispatch()
  const resourceRegistrationObservationWSKey =
    getResourceRegistrationNotificationKey(deviceId)
  const deviceNotificationKey = getDeviceNotificationKey(deviceId)
  const notificationsEnabled = useRef(false)
  notificationsEnabled.current = useSelector(
    isNotificationActive(deviceNotificationKey)
  )
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const isMounted = useIsMounted()
  const history = useHistory()

  const greyedOutClassName = classNames({
    'grayed-out': isUnregistered,
  })

  useEffect(() => {
    if (deviceId) {
      // Register the WS if not already registered
      WebSocketEventClient.subscribe(
        {
          eventFilter: [
            eventFilters.RESOURCE_PUBLISHED,
            eventFilters.RESOURCE_UNPUBLISHED,
          ],
          deviceIdFilter: [deviceId],
        },
        resourceRegistrationObservationWSKey,
        deviceResourceRegistrationListener({
          deviceId,
          deviceName,
        })
      )
    }

    return () => {
      // Unregister the WS if notification is off
      if (!notificationsEnabled.current) {
        WebSocketEventClient.unsubscribe(resourceRegistrationObservationWSKey)
      }
    }
  }, [deviceId, deviceName, resourceRegistrationObservationWSKey])

  useEffect(() => {
    if (isUnregistered) {
      // Unregister the WS when the device is unregistered
      WebSocketEventClient.unsubscribe(resourceRegistrationObservationWSKey)
    }
  }, [isUnregistered, resourceRegistrationObservationWSKey])

  const handleOpenDeleteDeviceModal = () => {
    if (isMounted.current) {
      setDeleteModalOpen(true)
    }
  }

  const handleCloseDeleteDeviceModal = () => {
    if (isMounted.current) {
      setDeleteModalOpen(false)
    }
  }

  const handleDeleteDevice = async () => {
    // api
    try {
      setDeleting(true)
      await deleteDevicesApi([deviceId])
      await sleep(200)

      if (isMounted.current) {
        showSuccessToast({
          title: _(t.deviceDeleted),
          message: _(t.deviceWasDeleted, { name: deviceName }),
        })

        // Unregister the WS when the device is deleted
        WebSocketEventClient.unsubscribe(resourceRegistrationObservationWSKey)

        // Redirect to the device page after a deletion
        history.push(`/device`)
      }
    } catch (error) {
      if (isMounted.current) {
        setDeleting(false)
      }
      handleDeleteDevicesErrors(error, _, true)
    }
  }

  return (
    <div
      className={classNames('d-flex align-items-center', greyedOutClassName)}
    >
      <Button
        className="m-r-30"
        variant="secondary"
        icon="fa-trash-alt"
        onClick={handleOpenDeleteDeviceModal}
        disabled={isUnregistered}
      >
        {_(t.delete)}
      </Button>

      <Switch
        disabled={isUnregistered}
        className={classNames({ shimmering: !deviceId })}
        id="status-notifications"
        label={_(t.notifications)}
        checked={notificationsEnabled.current}
        onChange={e => {
          if (e.target.checked) {
            // Request browser notifications
            // (browsers will explicitly disallow notification permission requests not triggered in response to a user gesture,
            // so we must call it to make sure the user has received a notification request)
            Notification?.requestPermission?.()
          }

          dispatch(toggleActiveNotification(deviceNotificationKey))
        }}
      />

      <ConfirmModal
        onConfirm={handleDeleteDevice}
        show={deleteModalOpen}
        title={
          <>
            <i className="fas fa-trash-alt" />
            {`${_(t.delete)} ${deviceName}`}
          </>
        }
        body={_(t.deleteDeviceMessage)}
        confirmButtonText={_(t.delete)}
        loading={deleting}
        onClose={handleCloseDeleteDeviceModal}
      >
        {_(t.delete)}
      </ConfirmModal>
    </div>
  )
}

DevicesDetailsHeader.displayName = 'DevicesDetailsHeader'

export default DevicesDetailsHeader
