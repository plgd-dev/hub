import { ResourcesType } from '@/containers/Devices/Devices.types'

export type Props = {
    deviceId: string
    deviceName: string
    handleOpenEditDeviceNameModal: () => void
    isOnline: boolean
    isUnregistered: boolean
    links: ResourcesType[]
}
