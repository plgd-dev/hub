import { ConditionDataType } from '@/containers/SnippetService/ServiceSnippet.types'

export type Props = {
    defaultFormData: Partial<ConditionDataType>
    resetIndex: number
}

export type Inputs = {
    apiAccessToken: string
}
