import { useCallback, useContext, useEffect, useMemo, useState } from 'react'

import AppContext from '@shared-ui/app/share/AppContext'
import { useStreamApi } from '@shared-ui/common/hooks'
import { security } from '@shared-ui/common/services'

import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'
import { SnippetServiceApiEndpoints } from './constants'
import { devicesApiEndpoints } from '@/containers/Devices/constants'
import { getAppliedDeviceConfigStatus } from '@/containers/SnippetService/utils'

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
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const [data, setData] = useState(null)

    const {
        data: appliedConfigData,
        loading: appliedConfigLoading,
        refresh: appliedConfigRefresh,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${SnippetServiceApiEndpoints.CONFIGURATIONS_APPLIED}`, {
        telemetryWebTracer,
        telemetrySpan: 'snippet-service-get-applied-devices-config',
    })

    const {
        data: devicesData,
        loading: devicesLoading,
        refresh: deviceRefresh,
    }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${devicesApiEndpoints.DEVICES}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-devices',
        unauthorizedCallback,
    })

    useEffect(() => {
        if (!devicesLoading && !appliedConfigLoading) {
            const appliedDeviceConfig = appliedConfigData.map((config: any) => {
                const device = devicesData.find((d: any) => d.id === config.deviceId)
                return {
                    ...config,
                    name: device?.name,
                    status: getAppliedDeviceConfigStatus(config),
                }
            })

            setData(appliedDeviceConfig)
        }
    }, [appliedConfigLoading, devicesData, devicesLoading, appliedConfigData])

    const refresh = useCallback(() => {
        appliedConfigRefresh()
        deviceRefresh()
    }, [appliedConfigRefresh, deviceRefresh])

    const loading = useMemo(() => devicesLoading || appliedConfigLoading, [appliedConfigLoading, devicesLoading])

    return { data, refresh, loading, ...rest }
}

export const useAppliedDeviceConfigDetail = (id: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const [data, setData] = useState(null)

    const {
        data: appliedConfigData,
        loading: appliedConfigLoading,
        refresh: appliedConfigRefresh,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${SnippetServiceApiEndpoints.CONFIGURATIONS_APPLIED}?httpIdFilter=${id}/latest`, {
        telemetryWebTracer,
        telemetrySpan: `snippet-service-get-applied-devices-config-${id}`,
        requestActive,
    })

    const {
        data: devicesData,
        loading: devicesLoading,
        refresh: deviceRefresh,
    }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${devicesApiEndpoints.DEVICES}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-devices',
        unauthorizedCallback,
    })

    useEffect(() => {
        if (!devicesLoading && !appliedConfigLoading) {
            const detailData = appliedConfigData && appliedConfigData[0]

            setData({
                ...detailData,
                name: devicesData.find((d: any) => d.id === detailData.deviceId)?.name,
                status: getAppliedDeviceConfigStatus(detailData),
            })
        }
    }, [appliedConfigLoading, devicesData, devicesLoading, appliedConfigData])

    const refresh = useCallback(() => {
        appliedConfigRefresh()
        deviceRefresh()
    }, [appliedConfigRefresh, deviceRefresh])

    const loading = useMemo(() => devicesLoading || appliedConfigLoading, [appliedConfigLoading, devicesLoading])

    return { data, refresh, loading, ...rest }
}
