import { ResourceContentType, ResourceType } from '@shared-ui/components/Organisms/ResourceToggleCreator/ResourceToggleCreator.types'

export type ConfigurationDataType = {
    id: string
    name: string
    owner: string
    resources?: ResourceType[] | null
    timestamp: string
    version: string
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
    timestamp: string
    version: string
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
