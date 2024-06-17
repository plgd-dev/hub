import { useContext, useEffect, useState } from 'react'

import AppContext from '@shared-ui/app/share/AppContext'
import { useStreamApi } from '@shared-ui/common/hooks'
import { security } from '@shared-ui/common/services'

import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'
import { SnippetServiceApiEndpoints } from './constants'

const getConfig = () => security.getGeneralConfig() as SecurityConfig

export const useResourcesConfigList = (requestActive = true): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    return useStreamApi(`${getConfig().httpGatewayAddress}${SnippetServiceApiEndpoints.CONFIGURATIONS}`, {
        telemetryWebTracer,
        telemetrySpan: 'snippet-service-get-resources-configurations',
        requestActive,
    })
}

export const useResourcesConfigDetail = (id: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)
    // const [data, setData] = useState(null)

    return useStreamApi(`${getConfig().httpGatewayAddress}${SnippetServiceApiEndpoints.CONFIGURATIONS}?httpIdFilter=${id}/latest`, {
        telemetryWebTracer,
        telemetrySpan: `snippet-service-get-resources-configuration-${id}`,
        requestActive,
    })

    // useEffect(() => {
    //     console.log(resData)
    //     if (resData && Array.isArray(resData)) {
    //         setData({
    //             ...resData[0],
    //             // inject id
    //             resources: resData[0].resources.map((r: any, i: number) => ({ ...r, id: i })),
    //         })
    //     }
    // }, [resData])
    //
    // return { data, ...rest }
}

export const useResourcesConfigConditions = (id: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    return useStreamApi(`${getConfig().httpGatewayAddress}${SnippetServiceApiEndpoints.CONDITIONS}?configurationIdFilter=${id}`, {
        telemetryWebTracer,
        telemetrySpan: `snippet-service-get-resources-configurations-conditions-${id}`,
        requestActive,
    })
}

export const useResourcesConfigApplied = (id: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    return useStreamApi(`${getConfig().httpGatewayAddress}${SnippetServiceApiEndpoints.CONFIGURATIONS_APPLIED}?configurationIdFilter=${id}`, {
        telemetryWebTracer,
        telemetrySpan: `snippet-service-get-resources-configurations-applied-${id}`,
        requestActive,
    })
}

export const useConditionsList = (): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    return useStreamApi(`${getConfig().httpGatewayAddress}${SnippetServiceApiEndpoints.CONDITIONS}`, {
        telemetryWebTracer,
        telemetrySpan: 'snippet-service-get-conditions',
    })
}

export const useConditionsDetail = (id: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)
    const [data, setData] = useState(null)

    const { data: resData, ...rest }: StreamApiPropsType = useStreamApi(
        `${getConfig().httpGatewayAddress}${SnippetServiceApiEndpoints.CONDITIONS}?httpIdFilter=${id}/latest`,
        {
            telemetryWebTracer,
            telemetrySpan: `snippet-service-get-condition-${id}`,
            requestActive,
        }
    )

    useEffect(() => {
        if (resData && Array.isArray(resData)) {
            setData(resData[0])
        }
    }, [resData])

    return { data, ...rest }
}

export const useAppliedDeviceConfigList = (): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    return useStreamApi(`${getConfig().httpGatewayAddress}${SnippetServiceApiEndpoints.CONFIGURATIONS_APPLIED}`, {
        telemetryWebTracer,
        telemetrySpan: 'snippet-service-get-applied-devices-config',
    })
}
