import { useEffect, useRef, useState } from 'react'
import { useIntl } from 'react-intl'
import { useSelector, useDispatch } from 'react-redux'
import classNames from 'classnames'
import PropTypes from 'prop-types'
import { useHistory } from 'react-router-dom'

import { showSuccessToast } from '@/components/toast'
import { ConfirmModal } from '@/components/confirm-modal'
import { Button } from '@/components/button'
import { WebSocketEventClient, eventFilters } from '@/common/services'
import { Switch } from '@/components/switch'
import { useIsMounted } from '@/common/hooks'
import {
  getThingNotificationKey,
  getResourceRegistrationNotificationKey,
  handleDeleteDevicesErrors,
} from './utils'
import { isNotificationActive, toggleActiveNotification } from './slice'
import { deviceResourceRegistrationListener } from './websockets'
import { deleteThingsApi } from './rest'
import { messages as t } from './things-i18n'

export const ThingsDetailsHeader = ({
  deviceId,
  deviceName,
  isUnregistered,
}) => {
  const { formatMessage: _ } = useIntl()
  const dispatch = useDispatch()
  const resourceRegistrationObservationWSKey =
    getResourceRegistrationNotificationKey(deviceId)
  const thingNotificationKey = getThingNotificationKey(deviceId)
  const notificationsEnabled = useRef(false)
  notificationsEnabled.current = useSelector(
    isNotificationActive(thingNotificationKey)
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
    setDeleteModalOpen(true)
  }

  const handleCloseDeleteDeviceModal = () => {
    setDeleteModalOpen(false)
  }

  const handleDeleteDevice = async () => {
    // api
    try {
      setDeleting(true)
      await deleteThingsApi([deviceId])

      if (isMounted.current) {
        showSuccessToast({
          title: _(t.thingDeleted),
          message: _(t.thingWasDeleted, { name: deviceName }),
        })

        // Unregister the WS when the device is deleted
        WebSocketEventClient.unsubscribe(resourceRegistrationObservationWSKey)
        setDeleting(false)
        setDeleteModalOpen(false)

        // Redirect to the things page after a deletion
        history.push(`/things`)
      }
    } catch (error) {
      setDeleting(false)
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

          dispatch(toggleActiveNotification(thingNotificationKey))
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

ThingsDetailsHeader.propTypes = {
  deviceId: PropTypes.string,
  deviceName: PropTypes.string,
  isUnregistered: PropTypes.bool.isRequired,
}

ThingsDetailsHeader.defaultProps = {
  deviceId: null,
  deviceName: null,
}
