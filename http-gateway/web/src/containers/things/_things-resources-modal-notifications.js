import { useEffect, useRef } from 'react'
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
  href,
  isUnregistered,
}) => {
  const { formatMessage: _ } = useIntl()
  const dispatch = useDispatch()
  const resourceUpdateObservationWSKey = getResourceUpdateNotificationKey(
    deviceId,
    href
  )
  const notificationsEnabled = useRef(false)
  notificationsEnabled.current = useSelector(
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

  const toggleNotifications = () => {
    if (notificationsEnabled.current) {
      WSManager.removeWsClient(resourceUpdateObservationWSKey)
    } else {
      // Register the WS
      WSManager.addWsClient({
        name: resourceUpdateObservationWSKey,
        api: `${thingsApiEndpoints.THINGS_WS}/${deviceId}${href}`,
        listener: deviceResourceUpdateListener(deviceId, href),
      })
    }

    dispatch(toggleActiveNotification(resourceUpdateObservationWSKey))
  }

  return (
    <Switch
      disabled={isUnregistered}
      id="resource-update-notifications"
      label={_(t.notifications)}
      checked={notificationsEnabled.current}
      onChange={toggleNotifications}
    />
  )
}

ThingsResourcesModalNotifications.propTypes = {
  deviceId: PropTypes.string,
  isUnregistered: PropTypes.bool.isRequired,
}

ThingsResourcesModalNotifications.defaultProps = {
  deviceId: null,
}
