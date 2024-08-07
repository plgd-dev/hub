import { AuthorizationDataType } from '@/containers/DeviceProvisioning/LinkedHubs/DetailPage/LinkedHubsDetailPage.types'

export type Props = {
    defaultFormData: any
    onSubmit?: () => void
}

export type Inputs = {
    authorization: AuthorizationDataType
}
