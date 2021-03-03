import { useEffect } from 'react'

import { WSManager } from './ws-manager'

import { thingsWSClient } from '@/containers/things/websockets'

export const InitServices = () => {
  useEffect(() => {
    // Register the default WS instances
    if (!WSManager.isInitialized) {
      WSManager.addWsClient(thingsWSClient)
      WSManager.registerWSClients()
    }

    return () => {
      // WSManager.unregisterWSClients()
    }
  }, [])

  return null
}
