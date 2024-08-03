import chunk from 'lodash/chunk'

import { withTelemetry } from '@shared-ui/common/services/opentelemetry'
import { fetchApi, security } from '@shared-ui/common/services'
import { FetchApiReturnType } from '@shared-ui/common/types/API.types'

import { OpenidConfigurationReturnType } from '@/containers/ApiTokens/ApiTokens.types'
import { ApiTokensApiEndpoints } from '@/containers/ApiTokens/constants'
import { SecurityConfig } from '@/containers/App/App.types'

export const getApiTokenUrlApi: () => Promise<string> = () => {
    const { m2mOauthClient } = security.getWellKnownConfig()

    return new Promise((resolve, reject) => {
        withTelemetry(
            () =>
                fetchApi(`${m2mOauthClient.authority}/.well-known/openid-configuration`, {
                    useToken: false,
                }),
            'get-m2mOauthClient-wellKnow-configuration'
        )
            .then((result: FetchApiReturnType<OpenidConfigurationReturnType>) => {
                if (result.data) {
                    const url = result?.data?.token_endpoint.split('m2m-oauth-server')[0]
                    resolve(url.endsWith('/') ? url.slice(0, -1) : url)
                } else {
                    reject(false)
                }
            })
            .catch((error: any) => {
                reject(error)
            })
    })
}

export const createApiTokenApi = async (body: any) => {
    const { cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig

    return getApiTokenUrlApi().then((url) =>
        withTelemetry(
            () =>
                fetchApi(`${url}${ApiTokensApiEndpoints.API_TOKENS}`, {
                    method: 'POST',
                    body,
                    cancelRequestDeadlineTimeout,
                }),
            'create-api-token'
        )
    )
}

export const removeApiTokenApi = (ids: string[], chunkSize = 50) => {
    const { cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig

    const chunks = chunk(ids, chunkSize)
    return Promise.all(
        chunks.map(async (ids) => {
            const idsString = ids.map((id) => `idFilter=${id}`).join('&')
            const url = await getApiTokenUrlApi()
            return withTelemetry(
                () =>
                    fetchApi(`${url}${ApiTokensApiEndpoints.API_TOKENS}?${idsString}`, {
                        method: 'DELETE',
                        cancelRequestDeadlineTimeout,
                        body: {
                            idFilter: ids,
                        },
                    }),
                `api-tokens-blacklist-${ids.join('-')}`
            )
        })
    )
}
