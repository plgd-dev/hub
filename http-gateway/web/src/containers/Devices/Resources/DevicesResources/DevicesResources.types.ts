import { devicesStatuses } from '@/containers/Devices/constants'
import { DeviceResourcesCrudType } from '@/containers/Devices/Devices.types'

export type DevicesResourcesDeviceStatusType = typeof devicesStatuses[keyof typeof devicesStatuses]

export type Props = {
    data: {
        deviceId?: string
        href?: string
        interfaces: string[]
        resourceTypes: string[]
    }
    deviceStatus: DevicesResourcesDeviceStatusType
    loading: boolean
    isActiveTab: boolean
    onCreate: (href: string) => void
    onUpdate: (data: { href: string; currentInterface?: string }) => void
    onDelete: (href: string) => void
    pageSize: {
        height?: number
        width?: number
    }
} & DeviceResourcesCrudType

export type GetColumnsType = {
    _: any
    loading: boolean
    isUnregistered: boolean
} & DeviceResourcesCrudType
