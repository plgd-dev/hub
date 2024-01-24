import { RefObject } from 'react'
import { HubDataType } from '@/containers/DeviceProvisioning/LinkedHubs/DetailPage/LinkedHubsDetailPage.types'

export type Props = {
    defaultFormData: HubDataType
    loading: boolean
    contentRefs: {
        ref1: RefObject<HTMLHeadingElement>
        ref2: RefObject<HTMLHeadingElement>
        ref3: RefObject<HTMLHeadingElement>
        ref4: RefObject<HTMLHeadingElement>
    }
}
