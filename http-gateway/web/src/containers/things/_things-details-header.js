import { useEffect, useRef } from 'react'
import { useIntl } from 'react-intl'
import { useSelector, useDispatch } from 'react-redux'
import classNames from 'classnames'
import PropTypes from 'prop-types'

import { WebSocketEventClient, eventFilters } from '@/common/services'
import { Switch } from '@/components/switch'
import {
  getThingNotificationKey,
  getResourceRegistrationNotificationKey,
} from './utils'
import { isNotificationActive, toggleActiveNotification } from './slice'
import { deviceResourceRegistrationListener } from './websockets'
import { messages as t } from './things-i18n'

export const ThingsDetailsHeader = ({
  deviceId,
  deviceName,
  isUnregistered,
}) => {
  const { formatMessage: _ } = useIntl()
  const dispatch = useDispatch()
  const resourceRegistrationObservationWSKey = getResourceRegistrationNotificationKey(
    deviceId
  )
  const thingNotificationKey = getThingNotificationKey(deviceId)
  const notificationsEnabled = useRef(false)
  notificationsEnabled.current = useSelector(
    isNotificationActive(thingNotificationKey)
  )

  useEffect(
    () => {
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
    },
    [deviceId, deviceName, resourceRegistrationObservationWSKey]
  )

  useEffect(
    () => {
      if (isUnregistered) {
        // Unregister the WS when the device is unregistered
        WebSocketEventClient.unsubscribe(resourceRegistrationObservationWSKey)
      }
    },
    [isUnregistered, resourceRegistrationObservationWSKey]
  )

  return (
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
