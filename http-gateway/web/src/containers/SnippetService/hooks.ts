import { useCallback, useContext, useEffect, useMemo, useState } from 'react'

import AppContext from '@shared-ui/app/share/AppContext'
import { useStreamApi } from '@shared-ui/common/hooks'
import { security } from '@shared-ui/common/services'

import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'
import { SnippetServiceApiEndpoints } from './constants'
import { devicesApiEndpoints } from '@/containers/Devices/constants'
import { getAppliedDeviceConfigStatus } from '@/containers/SnippetService/utils'

const getConfig = () => security.getGeneralConfig() as SecurityConfig
const getWellKnow = () => security.getWellKnownConfig()

export const useConfigurationList = (requestActive = true): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.snippetService || getConfig().httpGatewayAddress

    return useStreamApi(`${url}${SnippetServiceApiEndpoints.CONFIGURATIONS}`, {
        telemetryWebTracer,
        telemetrySpan: 'snippet-service-get-configurations',
        requestActive,
        unauthorizedCallback,
    })
}

export const useConfigurationDetail = (id: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.snippetService || getConfig().httpGatewayAddress

    return useStreamApi(`${url}${SnippetServiceApiEndpoints.CONFIGURATIONS}?httpIdFilter=${id}/all`, {
        telemetryWebTracer,
        telemetrySpan: `snippet-service-get-configuration-${id}`,
        requestActive,
        unauthorizedCallback,
    })
}

export const useConfigurationConditions = (id: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.snippetService || getConfig().httpGatewayAddress

    return useStreamApi(`${url}${SnippetServiceApiEndpoints.CONDITIONS}?configurationIdFilter=${id}`, {
        telemetryWebTracer,
        telemetrySpan: `snippet-service-get-configurations-conditions-${id}`,
        requestActive,
        unauthorizedCallback,
    })
}

export const useConfigurationAppliedConfigurations = (id: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.snippetService || getConfig().httpGatewayAddress

    return useStreamApi(`${url}${SnippetServiceApiEndpoints.CONFIGURATIONS_APPLIED}?httpConfigurationIdFilter=${id}`, {
        telemetryWebTracer,
        telemetrySpan: `snippet-service-get-configuration-applied-configurations-${id}`,
        requestActive,
        unauthorizedCallback,
    })
}

export const useConditionsList = (): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.snippetService || getConfig().httpGatewayAddress

    return useStreamApi(`${url}${SnippetServiceApiEndpoints.CONDITIONS}`, {
        telemetryWebTracer,
        telemetrySpan: 'snippet-service-get-conditions',
        unauthorizedCallback,
    })
}

export const useConditionsDetail = (id: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.snippetService || getConfig().httpGatewayAddress

    const [data, setData] = useState(null)

    const { data: resData, ...rest }: StreamApiPropsType = useStreamApi(`${url}${SnippetServiceApiEndpoints.CONDITIONS}?httpIdFilter=${id}/latest`, {
        telemetryWebTracer,
        telemetrySpan: `snippet-service-get-condition-${id}`,
        requestActive,
        unauthorizedCallback,
    })

    useEffect(() => {
        if (resData && Array.isArray(resData)) {
            setData(resData[0])
        }
    }, [resData])

    return { data, ...rest }
}

export const useAppliedConfigurationsList = (): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.snippetService || getConfig().httpGatewayAddress

    const [data, setData] = useState(null)

    const {
        data: appliedConfigData,
        loading: appliedConfigLoading,
        refresh: appliedConfigRefresh,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${url}${SnippetServiceApiEndpoints.CONFIGURATIONS_APPLIED}`, {
        telemetryWebTracer,
        telemetrySpan: 'snippet-service-get-applied-devices-config',
        unauthorizedCallback,
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
        if (!devicesLoading && !appliedConfigLoading && devicesData && appliedConfigData) {
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

export const useAppliedConfigurationDetail = (id: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.snippetService || getConfig().httpGatewayAddress

    const [data, setData] = useState(null)

    const {
        data: appliedConfigData,
        loading: appliedConfigLoading,
        refresh: appliedConfigRefresh,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${url}${SnippetServiceApiEndpoints.CONFIGURATIONS_APPLIED}?httpIdFilter=${id}/latest`, {
        telemetryWebTracer,
        telemetrySpan: `snippet-service-get-applied-configuration-${id}`,
        requestActive,
        unauthorizedCallback,
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

            if (detailData) {
                setData({
                    ...detailData,
                    name: devicesData.find((d: any) => d.id === detailData.deviceId)?.name,
                    status: getAppliedDeviceConfigStatus(detailData),
                })
            }
        }
    }, [appliedConfigLoading, devicesData, devicesLoading, appliedConfigData])

    const refresh = useCallback(() => {
        appliedConfigRefresh()
        deviceRefresh()
    }, [appliedConfigRefresh, deviceRefresh])

    const loading = useMemo(() => devicesLoading || appliedConfigLoading, [appliedConfigLoading, devicesLoading])

    return { data, refresh, loading, ...rest }
}
