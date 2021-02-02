import { useEffect, useRef } from 'react'

// Hook for checking if the component is mounted
export const useIsMounted = () => {
  const componentIsMounted = useRef(true)
  useEffect(
    () => () => {
      componentIsMounted.current = false
    },
    []
  )
  return componentIsMounted
}
