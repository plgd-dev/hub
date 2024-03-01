import { useContext } from 'react'

import AppContext from '@shared-ui/app/share/AppContext'
import { useStreamApi } from '@shared-ui/common/hooks'
import { security } from '@shared-ui/common/services'

import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'
import { certificatesEndpoints } from './constants'

const getConfig = () => security.getGeneralConfig() as SecurityConfig

export const useCertificatesList = (): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    return useStreamApi(`${getConfig().httpGatewayAddress}${certificatesEndpoints.CERTIFICATES}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-certificates',
    })
}

export const useCertificatesDetail = (id: string): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    const { data, ...rest } = useStreamApi(`${getConfig().httpGatewayAddress}${certificatesEndpoints.CERTIFICATES}?idFilter=${id}`, {
        telemetryWebTracer,
        telemetrySpan: `get-certificate-${id}`,
    })

    if (data && Array.isArray(data)) {
        return {
            data: data[0],
            ...rest,
        }
    }

    return { data, ...rest }
}
