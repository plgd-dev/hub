import { useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { UseFormWatch } from 'react-hook-form'

import AppContext from '@shared-ui/app/share/AppContext'
import { useStreamApi } from '@shared-ui/common/hooks'
import { security } from '@shared-ui/common/services'
import { useStreamVersionData } from '@shared-ui/common/hooks/useStreamVersionData'
import { StreamApiReturnType } from '@shared-ui/common/types/API.types'

import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'
import { SnippetServiceApiEndpoints } from './constants'
import { devicesApiEndpoints } from '@/containers/Devices/constants'
import { getAppliedDeviceConfigStatus } from '@/containers/SnippetService/utils'
import {
    AppliedConfigurationDataEnhancedType,
    AppliedConfigurationDataType,
    ConditionDataEnhancedType,
    ConditionDataType,
    ConfigurationDataType,
} from '@/containers/SnippetService/ServiceSnippet.types'
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
    return useAppliedConfigurationsList(`?httpConfigurationIdFilter=${id}`, requestActive)
}

export const useConditionsList = (): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.snippetService || getConfig().httpGatewayAddress
    const [data, setData] = useState(null)

    const {
        data: conditionsData,
        loading: conditionsLoading,
        refresh: conditionsRefresh,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${url}${SnippetServiceApiEndpoints.CONDITIONS}`, {
        telemetryWebTracer,
        telemetrySpan: 'snippet-service-get-conditions',
        unauthorizedCallback,
    })

    const {
        data: configurationsData,
        loading: configurationsLoading,
        refresh: configurationRefresh,
    } = useStreamVersionData<ConfigurationDataType[]>({
        unauthorizedCallback,
        url: `${url}${SnippetServiceApiEndpoints.CONFIGURATIONS}`,
        ids: conditionsData ? conditionsData.map((config: ConditionDataType) => config.configurationId) : [],
        requestActive: !!conditionsData,
        telemetrySpan: 'snippet-service-get-configurations',
    })

    const loading = useMemo(() => conditionsLoading || configurationsLoading, [conditionsLoading, configurationsLoading])

    const refresh = useCallback(() => {
        configurationRefresh()
        conditionsRefresh()
    }, [configurationRefresh, conditionsRefresh])

    useEffect(() => {
        if (configurationsData && conditionsData) {
            setData(
                conditionsData.map((condition: ConditionDataType) => ({
                    ...condition,
                    configurationName: configurationsData.find((configuration: ConfigurationDataType) => configuration.id === condition.configurationId)?.name,
                }))
            )
        }
    }, [loading, configurationsData, conditionsData])

    return { data, refresh, loading, ...rest }
}

export const useConditionsDetail = (id: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.snippetService || getConfig().httpGatewayAddress
    const [data, setData] = useState<ConditionDataEnhancedType | null>(null)

    const {
        data: conditionsData,
        loading: conditionsLoading,
        refresh: conditionsRefresh,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${url}${SnippetServiceApiEndpoints.CONDITIONS}?httpIdFilter=${id}/all`, {
        telemetryWebTracer,
        telemetrySpan: `snippet-service-get-condition-${id}`,
        requestActive,
        unauthorizedCallback,
    })

    const {
        data: configurationsData,
        loading: configurationsLoading,
        refresh: configurationRefresh,
    } = useStreamVersionData<ConfigurationDataType[]>({
        unauthorizedCallback,
        url: `${url}${SnippetServiceApiEndpoints.CONFIGURATIONS}`,
        ids: conditionsData ? conditionsData.map((config: ConditionDataType) => config.configurationId) : [],
        requestActive: !!conditionsData,
        telemetrySpan: 'snippet-service-get-configurations',
    })

    const loading = useMemo(() => conditionsLoading || configurationsLoading, [conditionsLoading, configurationsLoading])

    const refresh = useCallback(() => {
        configurationRefresh()
        conditionsRefresh()
    }, [configurationRefresh, conditionsRefresh])

    useEffect(() => {
        if (configurationsData && conditionsData) {
            setData(
                conditionsData.map((condition: ConditionDataType) => ({
                    ...condition,
                    configurationName: configurationsData.find((configuration: ConfigurationDataType) => configuration.id === condition.configurationId)?.name,
                }))
            )
        }
    }, [loading, configurationsData, conditionsData])

    return { data, refresh, loading, ...rest }
}

export const useAppliedConfigurationsList = (filter = '', requestActive = true): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.snippetService || getConfig().httpGatewayAddress

    const [data, setData] = useState(null)

    const {
        data: appliedConfigData,
        loading: appliedConfigLoading,
        refresh: appliedConfigRefresh,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${url}${SnippetServiceApiEndpoints.CONFIGURATIONS_APPLIED}${filter}`, {
        telemetryWebTracer,
        telemetrySpan: 'snippet-service-get-applied-devices-config',
        unauthorizedCallback,
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

    const {
        data: configurationsData,
        loading: configurationsLoading,
        refresh: configurationRefresh,
    } = useStreamVersionData<ConfigurationDataType[]>({
        unauthorizedCallback,
        url: `${url}${SnippetServiceApiEndpoints.CONFIGURATIONS}`,
        ids: appliedConfigData ? appliedConfigData.map((config: AppliedConfigurationDataType) => config.configurationId) : [],
        requestActive: !!appliedConfigData,
        telemetrySpan: 'snippet-service-get-configurations',
    })

    const {
        data: conditionsData,
        loading: conditionsLoading,
        refresh: conditionsRefresh,
    } = useStreamVersionData<ConditionDataType[]>({
        unauthorizedCallback,
        url: `${url}${SnippetServiceApiEndpoints.CONDITIONS}`,
        ids: appliedConfigData ? appliedConfigData.map((config: AppliedConfigurationDataType) => config.conditionId) : [],
        requestActive: !!appliedConfigData,
        telemetrySpan: 'snippet-service-get-conditions',
    })

    const allOnDemand = useMemo(() => appliedConfigData?.every((d: AppliedConfigurationDataType) => d.onDemand === true), [appliedConfigData])

    const loading = useMemo(
        () => devicesLoading || appliedConfigLoading || configurationsLoading || conditionsLoading,
        [appliedConfigLoading, configurationsLoading, devicesLoading, conditionsLoading]
    )

    useEffect(() => {
        if (devicesData && appliedConfigData && configurationsData && (conditionsData || allOnDemand)) {
            setData(
                appliedConfigData.map((appliedConfig: AppliedConfigurationDataType) => ({
                    ...appliedConfig,
                    name: devicesData.find((d: DeviceDataType) => d.id === appliedConfig.deviceId)?.name,
                    status: getAppliedDeviceConfigStatus(appliedConfig),
                    configurationName: configurationsData.find((configuration: ConfigurationDataType) => configuration.id === appliedConfig.configurationId.id)
                        ?.name,
                    conditionName: appliedConfig.conditionId
                        ? conditionsData?.find((condition: ConditionDataType) => condition.id === appliedConfig.conditionId?.id)?.name
                        : -1,
                }))
            )
        }
    }, [loading, devicesData, appliedConfigData, configurationsData, conditionsData, allOnDemand])

    const refresh = useCallback(() => {
        appliedConfigRefresh()
        deviceRefresh()
        configurationRefresh()
        conditionsRefresh()
    }, [appliedConfigRefresh, deviceRefresh, configurationRefresh, conditionsRefresh])

    return { data, refresh, loading, ...rest }
}

export const useAppliedConfigurationDetail = (id: string, requestActive = false): StreamApiReturnType<AppliedConfigurationDataEnhancedType> => {
    const [data, setData] = useState(null)
    const [loading, setLoading] = useState(true)

    const { data: listData, loading: listDataLoading, ...rest } = useAppliedConfigurationsList(`?idFilter=${id}`, requestActive)

    useEffect(() => {
        if (listData) {
            if (listData.length > 0) {
                setData(listData[0])
            }
            setLoading(false)
        }
    }, [listData])

    return { data, loading, ...rest }
}

export const useConditionFilterValidation = ({ watch }: { watch: UseFormWatch<any> }) => {
    const deviceIdFilterVal: string[] = watch('deviceIdFilter')
    const resourceHrefFilterVal: string[] = watch('resourceHrefFilter')
    const resourceTypeFilterVal: string[] = watch('resourceTypeFilter')

    const deviceIdFilter: string[] = useMemo(() => deviceIdFilterVal || [], [deviceIdFilterVal])
    const resourceHrefFilter: string[] = useMemo(() => resourceHrefFilterVal || [], [resourceHrefFilterVal])
    const resourceTypeFilter: string[] = useMemo(() => resourceTypeFilterVal || [], [resourceTypeFilterVal])

    return useMemo(
        () => !(deviceIdFilter.length > 0 || resourceHrefFilter.length > 0 || resourceTypeFilter.length > 0),
        [deviceIdFilter.length, resourceHrefFilter.length, resourceTypeFilter.length]
    )
}
