import { useEffect, useState } from 'react'

import { Emitter } from '@/common/services/emitter'

// Returns the last emitted data from a given eventKey.
export const useEmitterData = eventKey => {
  const [data, setData] = useState(null)

  useEffect(
    () => {
      Emitter.on(eventKey, setData)

      return () => Emitter.off(eventKey, setData)
    },
    [eventKey]
  )

  return data
}
