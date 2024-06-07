import { security } from '@shared-ui/common/services'
import { SecurityConfig } from '@/containers/App/App.types'
import { deleteByChunks } from '@shared-ui/common/services/api-utils'
import { DELETE_CHUNK_SIZE, SnippetServiceApiEndpoints } from '@/containers/SnippetService/constants'

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
