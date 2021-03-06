import { useEffect, useRef } from 'react'
import { useIntl } from 'react-intl'
import { useSelector, useDispatch } from 'react-redux'
import classNames from 'classnames'
import PropTypes from 'prop-types'

import { WSManager } from '@/common/services/ws-manager'
import { Switch } from '@/components/switch'
import { thingsApiEndpoints } from './constants'
import {
  getThingNotificationKey,
  getResourceRegistrationNotificationKey,
} from './utils'
import { isNotificationActive, toggleActiveNotification } from './slice'
import { deviceResourceRegistrationListener } from './websockets'
import { messages as t } from './things-i18n'

export const ThingsDetailsHeader = ({ deviceId, isUnregistered }) => {
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
        WSManager.addWsClient({
          name: resourceRegistrationObservationWSKey,
          api: `${thingsApiEndpoints.THINGS_WS}/${deviceId}`,
          listener: deviceResourceRegistrationListener(deviceId),
        })
      }

      return () => {
        // Unregister the WS if notification is off
        if (!notificationsEnabled.current) {
          WSManager.removeWsClient(resourceRegistrationObservationWSKey)
        }
      }
    },
    [deviceId, resourceRegistrationObservationWSKey]
  )

  useEffect(
    () => {
      if (isUnregistered) {
        // Unregister the WS when the device is unregistered
        WSManager.removeWsClient(resourceRegistrationObservationWSKey)
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
      onChange={() => dispatch(toggleActiveNotification(thingNotificationKey))}
    />
  )
}

ThingsDetailsHeader.propTypes = {
  deviceId: PropTypes.string,
  isUnregistered: PropTypes.bool.isRequired,
}

ThingsDetailsHeader.defaultProps = {
  deviceId: null,
}
