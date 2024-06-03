import { useContext } from 'react'

import AppContext from '@shared-ui/app/share/AppContext'
import { useStreamApi } from '@shared-ui/common/hooks'
import { security } from '@shared-ui/common/services'

import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'
import { certificatesEndpoints } from './constants'

const getConfig = () => security.getGeneralConfig() as SecurityConfig
const getWellKnow = () => security.getWellKnownConfig()

export const useCertificatesList = (): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.certificateAuthority || getWellKnow()?.ui?.deviceProvisioningService || getConfig().httpGatewayAddress

    return useStreamApi(`${url}${certificatesEndpoints.CERTIFICATES}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-certificates',
        unauthorizedCallback,
    })
}

export const useCertificatesDetail = (id: string): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.certificateAuthority || getWellKnow()?.ui?.deviceProvisioningService || getConfig().httpGatewayAddress

    const { data, ...rest } = useStreamApi(`${url}${certificatesEndpoints.CERTIFICATES}?idFilter=${id}`, {
        telemetryWebTracer,
        telemetrySpan: `get-certificate-${id}`,
        unauthorizedCallback,
    })

    if (data && Array.isArray(data)) {
        return {
            data: data[0],
            ...rest,
        }
    }

    return { data, ...rest }
}
