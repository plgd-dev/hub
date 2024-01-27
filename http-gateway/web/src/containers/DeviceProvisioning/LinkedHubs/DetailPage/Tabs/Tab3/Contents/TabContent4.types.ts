import { AuthorizationDataType, HubDataType } from '../../../LinkedHubsDetailPage.types'

export type Props = {
    loading: boolean
    defaultFormData: HubDataType
}

export type Inputs = {
    authorization: AuthorizationDataType
}
