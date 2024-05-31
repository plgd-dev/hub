import { HubDataType } from '../../LinkedHubsDetailPage.types'

export type Props = {
    defaultFormData: HubDataType
    resetIndex: number
}

export type Inputs = {
    name: string
    gateways: { value: string; id?: string }[]
}
