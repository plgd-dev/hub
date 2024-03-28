import { HubDataType } from '../../../LinkedHubsDetailPage.types'

export type Props = {
    loading: boolean
    defaultFormData: HubDataType
}

export type Inputs = {
    authorization: {
        provider: {
            authority: string
            clientId: string
            clientSecret: string
            name: string
            scopes: string[]
        }
    }
}
