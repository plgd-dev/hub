import { DevicesResourcesModalType } from '@shared-ui/components/Organisms/DevicesResourcesModal/DevicesResourcesModal.types'

export type Props = {
    defaultActiveTab?: number
}

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
