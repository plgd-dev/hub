import chunk from 'lodash/chunk'

import { fetchApi, security } from '@shared-ui/common/services'
import { withTelemetry } from '@shared-ui/common/services/opentelemetry'

import { DEVICE_AUTH_CODE_SESSION_KEY } from '@/constants'
import { devicesApiEndpoints, DEVICE_DELETE_CHUNK_SIZE } from './constants'
import { interfaceGetParam } from './utils'
import { SecurityConfig } from '@/containers/App/App.types'

/**
 * Get a single thing by its ID Rest Api endpoint
 * @param {*} params { deviceId }
 */
export const getDeviceApi = (deviceId: string) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()

    return withTelemetry(
        () => fetchApi(`${httpGatewayAddress}${devicesApiEndpoints.DEVICES}/${deviceId}`, { cancelRequestDeadlineTimeout, unauthorizedCallback }),
        'get-device'
    )
}

/**
 * Delete a set of devices by their IDs Rest Api endpoint
 * @param {*} params deviceIds
 * @param {*} data
 */
export const deleteDevicesApi = (deviceIds: string[]) => {
    // We split the fetch into multiple chunks due to the URL being too long for the browser to handle
    const chunks = chunk(deviceIds, DEVICE_DELETE_CHUNK_SIZE)
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()

    return Promise.all(
        chunks.map((ids) => {
            const idsString = ids.map((id) => `deviceIdFilter=${id}`).join('&')
            return withTelemetry(
                () =>
                    fetchApi(`${httpGatewayAddress}${devicesApiEndpoints.DEVICES}?${idsString}`, {
                        method: 'DELETE',
                        cancelRequestDeadlineTimeout,
                        unauthorizedCallback,
                    }),
                'delete-device'
            )
        })
    )
}

/**
 * Get devices RESOURCES Rest Api endpoint
 * @param {*} params { deviceId, href - resource href, currentInterface - interface }
 * @param {*} data
 */
export const getDevicesResourcesApi = ({ deviceId, href, currentInterface = '' }: { deviceId: string; href: string; currentInterface?: string }) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()

    return withTelemetry(
        () =>
            fetchApi(`${httpGatewayAddress}${devicesApiEndpoints.DEVICES}/${deviceId}/resources${href}${interfaceGetParam(currentInterface)}`, {
                cancelRequestDeadlineTimeout,
                unauthorizedCallback,
            }),
        'get-device-resource'
    )
}

/**
 * Update devices RESOURCE Rest Api endpoint
 * @param {*} params { deviceId, href - resource href, currentInterface - interface, ttl - timeToLive }
 * @param {*} data
 */
export const updateDevicesResourceApi = (
    {
        deviceId,
        href,
        currentInterface = '',
        ttl,
    }: {
        deviceId: string
        href: string
        currentInterface?: string
        ttl: any
    },
    data: any
) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()

    return withTelemetry(
        () =>
            fetchApi(
                `${httpGatewayAddress}${devicesApiEndpoints.DEVICES}/${deviceId}/resources${href}?timeToLive=${ttl}&${interfaceGetParam(currentInterface)}`,
                { method: 'PUT', body: data, timeToLive: ttl, cancelRequestDeadlineTimeout, unauthorizedCallback }
            ),
        'update-device-resource'
    )
}

/**
 * Create devices RESOURCE Rest Api endpoint
 * @param {*} params { deviceId, href - resource href, currentInterface - interface, ttl - timeToLive }
 * @param {*} data
 */
export const createDevicesResourceApi = (
    {
        deviceId,
        href,
        currentInterface = '',
        ttl,
    }: {
        deviceId: string
        href: string
        currentInterface?: string
        ttl: any
    },
    data: any
) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()

    return withTelemetry(
        () =>
            fetchApi(
                `${httpGatewayAddress}${devicesApiEndpoints.DEVICES}/${deviceId}/resource-links${href}?timeToLive=${ttl}&${interfaceGetParam(
                    currentInterface
                )}`,
                { method: 'POST', body: data, timeToLive: ttl, cancelRequestDeadlineTimeout, unauthorizedCallback }
            ),
        'create-device-resource'
    )
}

/**
 * Delete devices RESOURCE Rest Api endpoint
 * @param {*} params { deviceId, href - resource href, ttl - timeToLive }
 * @param {*} data
 */
export const deleteDevicesResourceApi = ({ deviceId, href, ttl }: { deviceId: string; href: string; ttl: any }) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()

    return withTelemetry(
        () =>
            fetchApi(`${httpGatewayAddress}${devicesApiEndpoints.DEVICES}/${deviceId}/resource-links${href}?timeToLive=${ttl}`, {
                method: 'DELETE',
                timeToLive: ttl,
                cancelRequestDeadlineTimeout,
                unauthorizedCallback,
            }),
        'delete-device-resource'
    )
}

/**
 * Update the twinEnabled of one Thing Rest Api endpoint
 * @param {*} deviceId
 * @param {*} twinEnabled
 */
export const updateDeviceTwinSynchronizationApi = (deviceId: string, twinEnabled: boolean) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()

    return withTelemetry(
        () =>
            fetchApi(`${httpGatewayAddress}${devicesApiEndpoints.DEVICES}/${deviceId}/metadata`, {
                method: 'PUT',
                body: { twinEnabled },
                cancelRequestDeadlineTimeout,
                unauthorizedCallback,
            }),
        'update-device-metadata'
    )
}

type DeviceOAuthConfigType = {
    clientId: string
    audience: string
    scopes: string[]
}

/**
 * Returns an async function which resolves with a authorization code gathered from a rendered iframe, used for onboarding of a device.
 * @param {*} deviceId
 */
export const getDeviceAuthCode = (deviceId: string) => {
    return new Promise((resolve, reject) => {
        const { clientId, audience, scopes = [] } = security.getDeviceOAuthConfig() as DeviceOAuthConfigType
        const AuthUserManager = security.getUserManager()

        if (!clientId) {
            return reject(new Error('clientId is missing from the deviceOauthClient configuration'))
        }

        AuthUserManager.metadataService.getAuthorizationEndpoint().then((authorizationEndpoint: string) => {
            let timeout: any = null
            const iframe = document.createElement('iframe')
            const audienceParam = audience ? `&audience=${audience}` : ''
            iframe.src = `${authorizationEndpoint}?response_type=code&client_id=${clientId}&scope=${scopes}${audienceParam}&redirect_uri=${window.location.origin}/devices&device_id=${deviceId}`

            const destroyIframe = () => {
                sessionStorage.removeItem(DEVICE_AUTH_CODE_SESSION_KEY)
                localStorage.removeItem(DEVICE_AUTH_CODE_SESSION_KEY)
                iframe.parentNode?.removeChild(iframe)
            }

            const doResolve = (value: string) => {
                destroyIframe()
                clearTimeout(timeout)
                resolve(value)
            }

            const doReject = () => {
                destroyIframe()
                clearTimeout(timeout)
                reject(new Error('Failed to get the device auth code.'))
            }

            iframe.onload = () => {
                let attempts = 0
                const maxAttempts = 40
                const getCode = () => {
                    attempts += 1
                    const code = sessionStorage.getItem(DEVICE_AUTH_CODE_SESSION_KEY) || localStorage.getItem(DEVICE_AUTH_CODE_SESSION_KEY)

                    if (code) {
                        return doResolve(code)
                    }

                    if (attempts > maxAttempts) {
                        return doReject()
                    }

                    timeout = setTimeout(getCode, 500)
                }

                getCode()
            }

            iframe.onerror = () => {
                doReject()
            }

            document.body.appendChild(iframe)
        })
    })
}
