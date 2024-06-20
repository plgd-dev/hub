import { ResourceType } from '@shared-ui/components/Organisms/ResourceToggleCreator/ResourceToggleCreator.types'

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
    configurationId: {
        id: string
        version: string
    }
    conditionId: {
        id: string
        version: string
    }
    deviceId: string
    id: string
    status: number
    timestamp: string
    version: string
    resources: {
        correlationId: string
        href: string
        resourceUpdated: {
            auditContext: {
                correlationId: string
                owner: string
            }
            content: string
            status: string
        }
        status: string
    }[]
}
