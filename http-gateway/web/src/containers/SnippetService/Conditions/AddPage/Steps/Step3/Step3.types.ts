import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'

export type Props = {
    defaultFormData: any
    isActivePage: boolean
    onFinish: () => void
}

export type Inputs = {
    configurationId: OptionType
}
