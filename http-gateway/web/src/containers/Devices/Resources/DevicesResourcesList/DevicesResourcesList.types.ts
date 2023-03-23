import { DeviceResourcesCrudType } from '@/containers/Devices/Devices.types'
import { DevicesResourcesDeviceStatusType } from '@/containers/Devices/Resources/DevicesResources/DevicesResources.types'

export type Props = {
    data: {
        deviceId?: string
        href?: string
        interfaces: string[]
        resourceTypes: string[]
    }
    deviceStatus: DevicesResourcesDeviceStatusType
    loading: boolean
    onCreate: (href: string) => void
    onDelete: (href: string) => void
    onUpdate: ({ deviceId, href }: { deviceId: string; href: string }) => void
    pageSize: {
        height?: number
        width?: number
    }
} & DeviceResourcesCrudType
