import { devicesStatuses } from '@/containers/Devices/constants'

export type DevicesDetailMetaDataStatusValueType = (typeof devicesStatuses)[keyof typeof devicesStatuses]
export type DevicesDetailMetaDataStatusValueType = (typeof devicesStatuses)[keyof typeof devicesStatuses]

export type ResourcesType = {
    deviceId: string
    href: string
    interfaces: string[]
    resourceTypes: string[]
}

export type DeviceResourcesCrudType = {
    onCreate: (href: string) => Promise<void>
    onDelete: (href: string) => void
    onUpdate: ({ deviceId, href }: { deviceId?: string; currentInterface?: string; href: string }) => void | Promise<void>
}

export type DeviceDataType = {
    id: string
    types: string[]
    endpoints: string[]
    name: string
    metadata: {
        status: {
            value: DevicesDetailMetaDataStatusValueType
        }
        connection?: {
            status?: string
            onlineValidUntil: number | string
        }
        twinEnabled?: boolean
    }
}
