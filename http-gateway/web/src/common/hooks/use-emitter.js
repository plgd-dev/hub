import { useEffect } from 'react'

import { Emitter } from '@/common/services/emitter'

// This hook automatically registers an event listener to the Emitter and cleans it up when props are changed.
export const useEmitter = (eventKey, listener) => {
  useEffect(
    () => {
      Emitter.on(eventKey, listener)

      return () => Emitter.off(eventKey, listener)
    },
    [eventKey, listener]
  )
}
