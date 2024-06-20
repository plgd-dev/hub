import { useCallback, useContext, useEffect, useMemo, useState } from 'react'

import AppContext from '@shared-ui/app/share/AppContext'
import { useStreamApi } from '@shared-ui/common/hooks'
import { security } from '@shared-ui/common/services'

import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'
import { SnippetServiceApiEndpoints } from './constants'
import { devicesApiEndpoints } from '@/containers/Devices/constants'
import { getAppliedDeviceConfigStatus } from '@/containers/SnippetService/utils'
import { AppliedConfigurationDataType, ConditionDataType, ConfigurationDataType } from '@/containers/SnippetService/ServiceSnippet.types'
import { DeviceDataType } from '@/containers/Devices/Devices.types'

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

    return useStreamApi(`${url}${SnippetServiceApiEndpoints.CONDITIONS}?httpIdFilter=${id}/all`, {
        telemetryWebTracer,
        telemetrySpan: `snippet-service-get-condition-${id}`,
        requestActive,
        unauthorizedCallback,
    })
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

    const { data: configurationsData, loading: configurationsLoading } = useConfigurationList()
    const { data: conditionsData, loading: conditionsLoading } = useConditionsList()

    const loading = useMemo(
        () => devicesLoading || appliedConfigLoading || configurationsLoading || conditionsLoading,
        [appliedConfigLoading, configurationsLoading, devicesLoading, conditionsLoading]
    )

    useEffect(() => {
        if (!loading && devicesData && appliedConfigData && configurationsData && conditionsData) {
            const appliedDeviceConfig = appliedConfigData.map((appliedConfig: AppliedConfigurationDataType) => {
                const device = devicesData.find((d: DeviceDataType) => d.id === appliedConfig.deviceId)
                return {
                    ...appliedConfig,
                    name: device?.name,
                    status: getAppliedDeviceConfigStatus(appliedConfig),
                    configurationName: configurationsData.find((configuration: ConfigurationDataType) => configuration.id === appliedConfig.configurationId.id)
                        ?.name,
                    conditionName: conditionsData.find((condition: ConditionDataType) => condition.id === appliedConfig.conditionId.id)?.name,
                }
            })

            setData(appliedDeviceConfig)
        }
    }, [loading, devicesData, appliedConfigData, configurationsData, conditionsData])

    const refresh = useCallback(() => {
        appliedConfigRefresh()
        deviceRefresh()
    }, [appliedConfigRefresh, deviceRefresh])

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
