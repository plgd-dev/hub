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
  onUpdate: (data: { href: string; currentInterface?: string }) => void
  onDelete: (href: string) => void
} & DeviceResourcesCrudType
