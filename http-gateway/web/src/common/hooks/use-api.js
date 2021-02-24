import { useEffect, useState } from 'react'

import { useIsMounted } from '@/common/hooks'
import { fetchApi } from '@/common/services'

export const useApi = (url, options = {}) => {
  const isMounted = useIsMounted()
  const [state, setState] = useState({
    error: null,
    loading: true,
    data: null,
  })
  const [refreshIndex, setRefreshIndex] = useState(0)

  useEffect(
    () => {
      ;(async () => {
        try {
          const { data } = await fetchApi(url, options)

          if (isMounted.current) {
            setState({
              ...state,
              data,
              error: null,
              loading: false,
            })
          }
        } catch (error) {
          if (isMounted.current) {
            setState({
              ...state,
              data: null,
              error,
              loading: false,
            })
          }
        }
      })()
    },
    [url, refreshIndex] // eslint-disable-line
  )

  return {
    ...state,
    refresh: () => setRefreshIndex(refreshIndex + 1),
  }
}
