import { ConditionDataEnhancedType } from '@/containers/SnippetService/ServiceSnippet.types'

export type Props = {
    defaultFormData: Partial<ConditionDataEnhancedType>
    resetIndex: number
    setFilterError?: (error: boolean) => void
}

export type Inputs = {
    jqExpressionFilter: string
    resourceHrefFilter: string[]
    resourceTypeFilter: string[]
    deviceIdFilter: string[]
}
