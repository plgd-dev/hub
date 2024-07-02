export type Props = {
    formData: any
    resetIndex: number
    setFilterError?: (error: boolean) => void
}

export type Inputs = {
    name: string
    enabled: boolean
    jqExpressionFilter: string
    resourceHrefFilter: string[]
    resourceTypeFilter: string[]
    deviceIdFilter: string[]
    apiAccessToken: string
}
