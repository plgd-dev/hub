import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'

export type Props = {
    defaultFormData: any
    isActivePage: boolean
}

export type Inputs = {
    deviceIds: OptionType[]
    resourceHrefFilter: string[]
    resourceTypeFilter: string[]
    jqExpressionFilter: string
}
