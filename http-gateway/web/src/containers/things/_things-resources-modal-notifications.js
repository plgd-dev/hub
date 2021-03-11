import { useEffect } from 'react'
import { useIntl } from 'react-intl'
import { useSelector, useDispatch } from 'react-redux'
import PropTypes from 'prop-types'

import { WSManager } from '@/common/services/ws-manager'
import { Switch } from '@/components/switch'
import { thingsApiEndpoints } from './constants'
import { getResourceUpdateNotificationKey } from './utils'
import { isNotificationActive, toggleActiveNotification } from './slice'
import { deviceResourceUpdateListener } from './websockets'
import { messages as t } from './things-i18n'

export const ThingsResourcesModalNotifications = ({
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

  useEffect(
    () => {
      if (isUnregistered) {
        // Unregister the WS when the device is unregistered
        WSManager.removeWsClient(resourceUpdateObservationWSKey)
      }
    },
    [isUnregistered, resourceUpdateObservationWSKey]
  )

  const toggleNotifications = e => {
    if (e.target.checked) {
      // Request browser notifications
      // (browsers will explicitly disallow notification permission requests not triggered in response to a user gesture,
      // so we must call it to make sure the user has received a notification request)
      Notification.requestPermission()

      // Register the WS
      WSManager.addWsClient({
        name: resourceUpdateObservationWSKey,
        api: `${thingsApiEndpoints.THINGS_WS}/${deviceId}${href}`,
        listener: deviceResourceUpdateListener({ deviceId, href, deviceName }),
      })
    } else {
      WSManager.removeWsClient(resourceUpdateObservationWSKey)
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

ThingsResourcesModalNotifications.propTypes = {
  deviceId: PropTypes.string,
  deviceName: PropTypes.string,
  isUnregistered: PropTypes.bool.isRequired,
}

ThingsResourcesModalNotifications.defaultProps = {
  deviceId: null,
  deviceName: null,
}
