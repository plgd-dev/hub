import chunk from 'lodash/chunk'

import { withTelemetry } from '@shared-ui/common/services/opentelemetry'
import { fetchApi, security } from '@shared-ui/common/services'

import { DEVICE_DELETE_CHUNK_SIZE } from '@/containers/Devices/constants'
import { SecurityConfig } from '@/containers/App/App.types'
import { certificatesEndpoints } from '@/containers/Certificates/constants'

const getConfig = () => security.getGeneralConfig() as SecurityConfig
const getWellKnow = () => security.getWellKnownConfig()

export const deleteCertificatesApi = (deviceIds: string[]) => {
    // We split the fetch into multiple chunks due to the URL being too long for the browser to handle
    const chunks = chunk(deviceIds, DEVICE_DELETE_CHUNK_SIZE)
    const { cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig

    const url = getWellKnow()?.certificateAuthority || getWellKnow()?.ui?.deviceProvisioningService || getConfig().httpGatewayAddress

    return Promise.all(
        chunks.map((ids) => {
            const idFilter = ids.map((id) => `idFilter=${id}`).join('&')
            return withTelemetry(
                () =>
                    fetchApi(`${url}${certificatesEndpoints.CERTIFICATES}?${idFilter}`, {
                        method: 'DELETE',
                        cancelRequestDeadlineTimeout,
                    }),
                'delete-certificate'
            )
        })
    )
}
