import { ResourceType } from '@shared-ui/components/Organisms/ResourceToggleCreator/ResourceToggleCreator.types'
import { APPLIED_CONFIGURATIONS_STATUS } from '@/containers/SnippetService/constants'

export type ConfigurationDataType = {
    id?: string
    name: string
    owner: string
    resources: ResourceType[]
    timestamp: string
    version: string
    timeToLive?: string
}

export type ConditionDataType = {
    apiAccessToken: string
    configurationId: string
    deviceIdFilter?: string[]
    enabled: boolean
    id: string
    jqExpressionFilter: string
    name: string
    owner: string
    resourceHrefFilter?: string[]
    resourceTypeFilter?: string[]
    timestamp: string
    version: string
}

export type ConditionDataEnhancedType = ConditionDataType & {
    configurationName?: string
}

export type AppliedConfigurationDataType = {
    conditionId?: {
        id: string
        version: string
    }
    configurationId: {
        id: string
        version: string
    }
    deviceId: string
    id: string
    onDemand?: boolean
    resources: ResourceType[]
    status: number
    timestamp: string
    version: string
}

export type AppliedConfigurationDataEnhancedType = AppliedConfigurationDataType & {
    name: string
    status: string
    configurationName?: string
    conditionName?: string | number
}

export type AppliedConfigurationStatusType = (typeof APPLIED_CONFIGURATIONS_STATUS)[keyof typeof APPLIED_CONFIGURATIONS_STATUS]
