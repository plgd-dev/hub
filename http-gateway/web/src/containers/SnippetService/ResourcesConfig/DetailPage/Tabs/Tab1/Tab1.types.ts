export type Props = {
    defaultFormData: any
    isActiveTab: boolean
    loading: boolean
    resetIndex?: number
}

export type Inputs = {
    name: string
}

export type ResourceType = {
    href: string
    timeToLive: string
    content: ResourceContentType
}

export type ResourceTypeEnhanced = ResourceType & { id: number }

export type ResourceContentType = object | string | number | boolean
