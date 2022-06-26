import { useEffect, useState } from 'react'
import { context, trace } from '@opentelemetry/api'
import get from 'lodash/get'

import { useIsMounted } from '@/common/hooks'
import { fetchApi, streamApi } from '@/common/services'
import { useAppConfig } from '@/containers/app'

const getData = async (method, url, options, telemetryWebTracer) => {
  const { telemetrySpan } = options
  let dataToReturn = undefined

  if (telemetryWebTracer && telemetrySpan) {
    const singleSpan = telemetryWebTracer.startSpan(telemetrySpan)
    await context.with(
      trace.setSpan(context.active(), singleSpan),
      async () => {
        dataToReturn = await method(url, options).then(result => {
          trace
            .getSpan(context.active())
            .addEvent('fetching-single-span-completed')
          singleSpan.end()

          return result.data
        })
      }
    )
  } else {
    const { data } = await method(url, options)
    return data
  }

  return dataToReturn
}

export const useStreamApi = (url, options = {}) => {
  const isMounted = useIsMounted()
  const [state, setState] = useState({
    error: null,
    loading: true,
    data: null,
  })
  const [refreshIndex, setRefreshIndex] = useState(0)
  const { telemetryWebTracer } = useAppConfig()
  const apiMethod = get(options, 'streamApi', true) ? streamApi : fetchApi

  useEffect(
    () => {
      ;(async () => {
        try {
          // Set loading to true
          setState({ ...state, loading: true })
          const data = await getData(
            apiMethod,
            url,
            options,
            telemetryWebTracer
          )

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

            trace
              .getSpan(context.active())
              .addEvent('fetching-single-span-completed')
          }
        }
      })()
    },
    [url, refreshIndex] // eslint-disable-line
  )

  return {
    ...state,
    updateData: updatedData => setState({ ...state, data: updatedData }),
    refresh: () => setRefreshIndex(Math.random),
  }
}
