import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'

export type Props = {
    formData: any
    refs: {
        accessToken: (element: HTMLDivElement) => void
        filterDeviceId: (element: HTMLDivElement) => void
        filterJqExpression: (element: HTMLDivElement) => void
        filterResourceHref: (element: HTMLDivElement) => void
        filterResourceType: (element: HTMLDivElement) => void
        general: (element: HTMLDivElement) => void
    }
    resetIndex: number
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
