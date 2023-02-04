import { DevicesResourcesModalType } from '../../Resources/DevicesResourcesModal/DevicesResourcesModal.types'

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
