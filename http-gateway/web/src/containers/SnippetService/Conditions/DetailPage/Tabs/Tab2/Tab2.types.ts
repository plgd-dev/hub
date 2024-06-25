import { ConditionDataType } from '@/containers/SnippetService/ServiceSnippet.types'

export type Props = {
    defaultFormData: Partial<ConditionDataType>
    resetIndex: number
    setFilterError?: (error: boolean) => void
}

export type Inputs = {
    jqExpressionFilter: string
    resourceHrefFilter: string[]
    resourceTypeFilter: string[]
    deviceIdFilter: string[]
}
