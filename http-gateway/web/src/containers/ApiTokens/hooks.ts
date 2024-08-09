import { useContext, useEffect, useState } from 'react'

import AppContext from '@shared-ui/app/share/AppContext'
import { useStreamApi } from '@shared-ui/common/hooks'

import { getApiTokenUrlApi } from '@/containers/ApiTokens/rest'
import { ApiTokensApiEndpoints } from '@/containers/ApiTokens/constants'

export const useApiTokensList = (requestActive = true) => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)

    const [url, setUrl] = useState('')

    useEffect(() => {
        if (requestActive) {
            getApiTokenUrlApi().then((result: string) => {
                setUrl(`${result}${ApiTokensApiEndpoints.API_TOKENS}`)
            })
        }
    }, [requestActive])

    return useStreamApi(url, {
        telemetryWebTracer,
        telemetrySpan: 'api-tokens-get-tokens',
        unauthorizedCallback,
        requestActive: url !== '' && requestActive,
    })
}

export const useApiTokenDetail = (id: string, requestActive = true) => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)

    const [url, setUrl] = useState('')
    const [data, setData] = useState<any>({})
    const [loading, setLoading] = useState<boolean>(true)

    useEffect(() => {
        if (requestActive) {
            getApiTokenUrlApi().then((result: string) => {
                setUrl(`${result}${ApiTokensApiEndpoints.API_TOKENS}?idFilter=${id}`)
            })
        }
    }, [id, requestActive])

    const {
        data: tokenData,
        loading: tokenLoading,
        ...rest
    } = useStreamApi(url, {
        telemetryWebTracer,
        telemetrySpan: `api-tokens-get-token-${id}`,
        unauthorizedCallback,
        requestActive: url !== '' && requestActive,
    })

    useEffect(() => {
        if (tokenData) {
            setData(tokenData[0])
            setLoading(false)
        }
    }, [tokenData])

    return { data, loading, ...rest }
}
