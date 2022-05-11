import { useEffect } from 'react'

import { deviceStatusListener } from '@/containers/devices/websockets'
import { WebSocketEventClient } from './websocket-event-client'
import { eventFilters } from './websocket-event-constants'

let initialized = false

export const InitServices = () => {
  useEffect(() => {
    // Register the default WS instances
    WebSocketEventClient._connect()
    WebSocketEventClient.onOpen = () => {
      if (!initialized) {
        WebSocketEventClient.subscribe(
          {
            eventFilter: [
              eventFilters.DEVICE_METADATA_UPDATED,
              eventFilters.REGISTERED,
              eventFilters.UNREGISTERED,
            ],
          },
          'device-status',
          deviceStatusListener
        )
      }
      initialized = true
    }

    return () => {
      WebSocketEventClient.unsubscribe('device-status')
    }
  }, [])

  return null
}
