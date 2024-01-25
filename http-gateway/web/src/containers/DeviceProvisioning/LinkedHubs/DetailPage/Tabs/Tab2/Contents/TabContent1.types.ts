import { GRPCData, HubDataType } from '@/containers/DeviceProvisioning/LinkedHubs/DetailPage/LinkedHubsDetailPage.types'

export type Props = { defaultFormData: HubDataType; loading: boolean }

export type Inputs = {
    certificateAuthority: {
        grpc: GRPCData
    }
}
