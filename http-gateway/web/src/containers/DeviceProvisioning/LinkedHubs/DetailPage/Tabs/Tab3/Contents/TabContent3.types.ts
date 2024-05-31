import { RefObject } from 'react'
import { AuthorizationDataType, HubDataType } from '../../../LinkedHubsDetailPage.types'

export type Props = {
    contentRefs?: {
        ref1: RefObject<HTMLHeadingElement>
        ref2: RefObject<HTMLHeadingElement>
        ref3: RefObject<HTMLHeadingElement>
    }
    defaultFormData: HubDataType
    loading: boolean
}

export type Inputs = {
    authorization: AuthorizationDataType
}
