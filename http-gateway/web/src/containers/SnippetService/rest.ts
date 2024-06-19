import { fetchApi, security } from '@shared-ui/common/services'
import { deleteByChunks } from '@shared-ui/common/services/api-utils'
import { withTelemetry } from '@shared-ui/common/services/opentelemetry'

import { SecurityConfig } from '@/containers/App/App.types'
import { DELETE_CHUNK_SIZE, SnippetServiceApiEndpoints } from '@/containers/SnippetService/constants'

const getWellKnow = () => security.getWellKnownConfig()

export const createConfigurationApi = (body: any) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const url = getWellKnow()?.ui?.snippetService || httpGatewayAddress

    return withTelemetry(
        () =>
            fetchApi(`${url}${SnippetServiceApiEndpoints.CONFIGURATIONS}`, {
                method: 'POST',
                cancelRequestDeadlineTimeout,
                body,
            }),
        'create-resource-config'
    )
}

export const updateResourceConfigApi = (id: string, body: any) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const url = getWellKnow()?.ui?.snippetService || httpGatewayAddress

    return withTelemetry(
        () =>
            fetchApi(`${url}${SnippetServiceApiEndpoints.CONFIGURATIONS}/${id}`, {
                method: 'PUT',
                cancelRequestDeadlineTimeout,
                body,
            }),
        `update-resource-config-${id}`
    )
}

export const createConditionApi = (body: any) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const url = getWellKnow()?.ui?.snippetService || httpGatewayAddress

    return withTelemetry(
        () =>
            fetchApi(`${url}${SnippetServiceApiEndpoints.CONDITIONS}`, {
                method: 'POST',
                cancelRequestDeadlineTimeout,
                body,
            }),
        'create-condition'
    )
}

export const updateConditionApi = (id: string, body: any) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const url = getWellKnow()?.ui?.snippetService || httpGatewayAddress

    return withTelemetry(
        () =>
            fetchApi(`${url}${SnippetServiceApiEndpoints.CONDITIONS}/${id}`, {
                method: 'PUT',
                cancelRequestDeadlineTimeout,
                body,
            }),
        `update-condition-${id}`
    )
}

export const deleteConfigurationsApi = (ids: string[]) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const url = getWellKnow()?.ui?.snippetService || httpGatewayAddress

    return deleteByChunks(
        `${url}${SnippetServiceApiEndpoints.CONFIGURATIONS}`,
        ids,
        cancelRequestDeadlineTimeout,
        'snippet-service-delete-configurations',
        'httpIdFilter',
        '/all',
        DELETE_CHUNK_SIZE
    )
}

export const deleteConditionsApi = (ids: string[]) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const url = getWellKnow()?.ui?.snippetService || httpGatewayAddress

    return deleteByChunks(
        `${url}${SnippetServiceApiEndpoints.CONDITIONS}`,
        ids,
        cancelRequestDeadlineTimeout,
        'snippet-service-delete-conditions',
        'httpIdFilter',
        '/all',
        DELETE_CHUNK_SIZE
    )
}

export const deleteAppliedDeviceConfigApi = (ids: string[]) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const url = getWellKnow()?.ui?.snippetService || httpGatewayAddress

    return deleteByChunks(
        `${url}${SnippetServiceApiEndpoints.CONFIGURATIONS_APPLIED}`,
        ids,
        cancelRequestDeadlineTimeout,
        'snippet-service-delete-applied-devices-config',
        'httpIdFilter',
        '/all',
        DELETE_CHUNK_SIZE
    )
}
