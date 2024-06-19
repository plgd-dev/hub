import { ResourceType } from '@shared-ui/components/Organisms/ResourceToggleCreator/ResourceToggleCreator.types'

export type Props = {
    defaultFormData: any
    isActiveTab: boolean
    loading: boolean
    resetIndex?: number
}

export type Inputs = {
    name: string
}

export type ResourceTypeEnhanced = ResourceType & { id: number }
