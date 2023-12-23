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
