import { ResourceType } from '@shared-ui/components/Organisms/ResourceToggleCreator/ResourceToggleCreator.types'

export type Props = {
    defaultFormData: any
    isActiveTab: boolean
    loading: boolean
    resetIndex?: number
    setResourcesError?: (error: boolean) => void
}

export type Inputs = {
    name: string
    resources: ResourceType[]
}

export type ResourceTypeEnhanced = ResourceType & { id: number }
