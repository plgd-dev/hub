import { GRPCData } from '@/containers/DeviceProvisioning/LinkedHubs/DetailPage/LinkedHubsDetailPage.types'

export type Props = {
    defaultFormData: any
}

export type Inputs = {
    certificateAuthority: {
        grpc: GRPCData
    }
}
