// import { DevicesResourcesModalType } from '../../Resources/DevicesResourcesModal/DevicesResourcesModal.types'
import { DevicesResourcesModalType } from '@shared-ui/components/organisms/DevicesResourcesModal/DevicesResourcesModal.types'

export type DevicesDetailsResourceModalData = {
    data: {
        deviceId?: string
        href?: string
        interfaces?: string[]
        types: string[]
    }
    resourceData: any
    type?: DevicesResourcesModalType
}
