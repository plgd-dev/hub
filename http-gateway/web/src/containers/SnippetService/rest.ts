import { fetchApi, security } from '@shared-ui/common/services'
import { deleteByChunks } from '@shared-ui/common/services/api-utils'
import { withTelemetry } from '@shared-ui/common/services/opentelemetry'

import { SecurityConfig } from '@/containers/App/App.types'
import { DELETE_CHUNK_SIZE, SnippetServiceApiEndpoints } from '@/containers/SnippetService/constants'

export const createResourcesConfigApi = (body: any) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig

    return withTelemetry(
        () =>
            fetchApi(`${httpGatewayAddress}${SnippetServiceApiEndpoints.CONFIGURATIONS}`, {
                method: 'POST',
                cancelRequestDeadlineTimeout,
                body,
            }),
        'create-resource-config'
    )
}

export const updateResourceConfigApi = (id: string, body: any) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig

    return withTelemetry(
        () =>
            fetchApi(`${httpGatewayAddress}${SnippetServiceApiEndpoints.CONFIGURATIONS}/${id}`, {
                method: 'PUT',
                cancelRequestDeadlineTimeout,
                body,
            }),
        `update-resource-config-${id}`
    )
}

export const createConditionApi = (body: any) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig

    return withTelemetry(
        () =>
            fetchApi(`${httpGatewayAddress}${SnippetServiceApiEndpoints.CONDITIONS}`, {
                method: 'POST',
                cancelRequestDeadlineTimeout,
                body,
            }),
        'create-condition'
    )
}

export const updateConditionApi = (id: string, body: any) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig

    return withTelemetry(
        () =>
            fetchApi(`${httpGatewayAddress}${SnippetServiceApiEndpoints.CONDITIONS}/${id}`, {
                method: 'PUT',
                cancelRequestDeadlineTimeout,
                body,
            }),
        `update-condition-${id}`
    )
}

export const deleteResourcesConfigApi = (ids: string[]) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig

    return deleteByChunks(
        `${httpGatewayAddress}${SnippetServiceApiEndpoints.CONFIGURATIONS}`,
        ids,
        cancelRequestDeadlineTimeout,
        'snippet-service-delete-resources-config',
        DELETE_CHUNK_SIZE
    )
}

export const deleteConditionsApi = (ids: string[]) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig

    return deleteByChunks(
        `${httpGatewayAddress}${SnippetServiceApiEndpoints.CONDITIONS}`,
        ids,
        cancelRequestDeadlineTimeout,
        'snippet-service-delete-conditions',
        DELETE_CHUNK_SIZE
    )
}

export const deleteAppliedDeviceConfigApi = (ids: string[]) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig

    return deleteByChunks(
        `${httpGatewayAddress}${SnippetServiceApiEndpoints.CONFIGURATIONS_APPLIED}`,
        ids,
        cancelRequestDeadlineTimeout,
        'snippet-service-delete-applied-devices-config',
        DELETE_CHUNK_SIZE
    )
}
