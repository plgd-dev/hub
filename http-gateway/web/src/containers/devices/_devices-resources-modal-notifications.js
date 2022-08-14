import { useEffect } from 'react'
import { useIntl } from 'react-intl'
import { useSelector, useDispatch } from 'react-redux'
import PropTypes from 'prop-types'

import { WebSocketEventClient, eventFilters } from '@shared-ui/common/services'
import { Switch } from '@shared-ui/components/old/switch'
import { getResourceUpdateNotificationKey } from './utils'
import { isNotificationActive, toggleActiveNotification } from './slice'
import { deviceResourceUpdateListener } from './websockets'
import { messages as t } from './devices-i18n'

export const DevicesResourcesModalNotifications = ({
  deviceId,
  deviceName,
  href,
  isUnregistered,
}) => {
  const { formatMessage: _ } = useIntl()
  const dispatch = useDispatch()
  const resourceUpdateObservationWSKey = getResourceUpdateNotificationKey(
    deviceId,
    href
  )
  const notificationsEnabled = useSelector(
    isNotificationActive(resourceUpdateObservationWSKey)
  )

  useEffect(() => {
    if (isUnregistered) {
      // Unregister the WS when the device is unregistered
      WebSocketEventClient.unsubscribe(resourceUpdateObservationWSKey)
    }
  }, [isUnregistered, resourceUpdateObservationWSKey])

  const toggleNotifications = e => {
    if (e.target.checked) {
      // Request browser notifications
      // (browsers will explicitly disallow notification permission requests not triggered in response to a user gesture,
      // so we must call it to make sure the user has received a notification request)
      Notification?.requestPermission?.()

      // Register the WS
      WebSocketEventClient.subscribe(
        {
          eventFilter: [eventFilters.RESOURCE_CHANGED],
          resourceIdFilter: [`${deviceId}${href}`],
        },
        resourceUpdateObservationWSKey,
        deviceResourceUpdateListener({ deviceId, href, deviceName })
      )
    } else {
      WebSocketEventClient.unsubscribe(resourceUpdateObservationWSKey)
    }

    dispatch(toggleActiveNotification(resourceUpdateObservationWSKey))
  }

  return (
    <Switch
      disabled={isUnregistered}
      id="resource-update-notifications"
      label={_(t.notifications)}
      checked={notificationsEnabled}
      onChange={toggleNotifications}
    />
  )
}

DevicesResourcesModalNotifications.propTypes = {
  deviceId: PropTypes.string,
  deviceName: PropTypes.string,
  isUnregistered: PropTypes.bool.isRequired,
}

DevicesResourcesModalNotifications.defaultProps = {
  deviceId: null,
  deviceName: null,
}
