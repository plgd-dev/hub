import { RefObject } from 'react'
import { GRPCData, HubDataType } from '../../../LinkedHubsDetailPage.types'

export type Props = {
    defaultFormData: HubDataType
    loading: boolean
    contentRefs?: {
        ref1: RefObject<HTMLHeadingElement>
        ref2: RefObject<HTMLHeadingElement>
        ref3: RefObject<HTMLHeadingElement>
    }
}

export type Inputs = {
    certificateAuthority: {
        grpc: GRPCData
    }
}
