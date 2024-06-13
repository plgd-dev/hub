import chunk from 'lodash/chunk'

import { fetchApi, security } from '@shared-ui/common/services'
import { withTelemetry } from '@shared-ui/common/services/opentelemetry'

import { SecurityConfig } from '@/containers/App/App.types'
import { DPS_DELETE_CHUNK_SIZE, dpsApiEndpoints } from './constants'
import { HubDataType } from '@/containers/DeviceProvisioning/LinkedHubs/DetailPage/LinkedHubsDetailPage.types'

export const deleteProvisioningRecordsApi = (provisioningRecordsIds: string[]) => {
    // We split the fetch into multiple chunks due to the URL being too long for the browser to handle
    const chunks = chunk(provisioningRecordsIds, DPS_DELETE_CHUNK_SIZE)
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()

    return Promise.all(
        chunks.map((ids) => {
            const idsString = ids.map((id) => `idFilter=${id}`).join('&')
            return withTelemetry(
                () =>
                    fetchApi(`${httpGatewayAddress}${dpsApiEndpoints.PROVISIONING_RECORDS}?${idsString}`, {
                        method: 'DELETE',
                        cancelRequestDeadlineTimeout,
                        unauthorizedCallback,
                    }),
                'delete-provisioning-records'
            )
        })
    )
}

export type UpdateProvisioningRecordNameBodyType = {
    owner: string
    attestationMechanism: {
        x509: {
            certificateChain: string
            leadCertificateName: string
            expiredCertificateEnabled: boolean
        }
    }
    hubId: string
    preSharedKey: string
    name: string
}

export const updateProvisioningRecordNameApi = (enrollmentGroupId: string, body: UpdateProvisioningRecordNameBodyType) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()

    return withTelemetry(
        () =>
            fetchApi(`${httpGatewayAddress}${dpsApiEndpoints.ENROLLMENT_GROUPS}/${enrollmentGroupId}`, {
                method: 'PUT',
                cancelRequestDeadlineTimeout,
                body,
                unauthorizedCallback,
            }),
        'edit-enrollment-group-name'
    )
}

export const deleteEnrollmentGroupsApi = (provisioningRecordsIds: string[]) => {
    // We split the fetch into multiple chunks due to the URL being too long for the browser to handle
    const chunks = chunk(provisioningRecordsIds, DPS_DELETE_CHUNK_SIZE)
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()

    return Promise.all(
        chunks.map((ids) => {
            const idsString = ids.map((id) => `idFilter=${id}`).join('&')
            return withTelemetry(
                () =>
                    fetchApi(`${httpGatewayAddress}${dpsApiEndpoints.ENROLLMENT_GROUPS}?${idsString}`, {
                        method: 'DELETE',
                        cancelRequestDeadlineTimeout,
                        unauthorizedCallback,
                    }),
                'delete-enrollment-groups'
            )
        })
    )
}

export const deleteLinkedHubsApi = (linkedHubsIds: string[]) => {
    // We split the fetch into multiple chunks due to the URL being too long for the browser to handle
    const chunks = chunk(linkedHubsIds, DPS_DELETE_CHUNK_SIZE)
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()

    return Promise.all(
        chunks.map((ids) => {
            const idsString = ids.map((id) => `idFilter=${id}`).join('&')
            return withTelemetry(
                () =>
                    fetchApi(`${httpGatewayAddress}${dpsApiEndpoints.HUBS}?${idsString}`, {
                        method: 'DELETE',
                        cancelRequestDeadlineTimeout,
                        unauthorizedCallback,
                    }),
                'delete-linked-hubs'
            )
        })
    )
}

export const updateLinkedHubData = (linkedHubsId: string, body: Omit<HubDataType, 'id'>) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()
    return withTelemetry(
        () =>
            fetchApi(`${httpGatewayAddress}${dpsApiEndpoints.HUBS}/${linkedHubsId}`, {
                method: 'PUT',
                cancelRequestDeadlineTimeout,
                body,
                unauthorizedCallback,
            }),
        'update-linked-hub'
    )
}

export const createLinkedHub = (body: Omit<HubDataType, 'id'>) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()
    return withTelemetry(
        () =>
            fetchApi(`${httpGatewayAddress}${dpsApiEndpoints.HUBS}`, {
                method: 'POST',
                cancelRequestDeadlineTimeout,
                body,
                unauthorizedCallback,
            }),
        'update-linked-hub'
    )
}

export const updateEnrollmentGroup = (enrollmentGroupId: string, body: any) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()
    return withTelemetry(
        () =>
            fetchApi(`${httpGatewayAddress}${dpsApiEndpoints.ENROLLMENT_GROUPS}/${enrollmentGroupId}`, {
                method: 'PUT',
                cancelRequestDeadlineTimeout,
                body,
                unauthorizedCallback,
            }),
        'update-enrollment-group'
    )
}

export const createEnrollmentGroup = (body: any) => {
    const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
    const { unauthorizedCallback } = security.getWellKnownConfig()
    return withTelemetry(
        () =>
            fetchApi(`${httpGatewayAddress}${dpsApiEndpoints.ENROLLMENT_GROUPS}`, {
                method: 'POST',
                cancelRequestDeadlineTimeout,
                body,
                unauthorizedCallback,
            }),
        'update-enrollment-group'
    )
}
