import { ConditionDataEnhancedType } from '@/containers/SnippetService/ServiceSnippet.types'

export type Props = {
    defaultFormData: Partial<ConditionDataEnhancedType>
    resetIndex: number
}

export type Inputs = {
    name: string
    enabled: boolean
}
