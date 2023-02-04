import { DevicesDetailMetaDataStatusValueType } from './/Detail/DevicesDetails/DevicesDetails.types'

export type ResourcesType = {
  deviceId: string
  href: string
  interfaces: string[]
  resourceTypes: string[]
}

export type StreamApiPropsType = {
  data: any
  updateData: any
  loading: boolean
  error: string | null
  refresh: () => void
}

export type DeviceResourcesCrudType = {
  onCreate: (href: string) => Promise<void>
  onDelete: (href: string) => void
  onUpdate: ({
    deviceId,
    href,
  }: {
    deviceId?: string
    currentInterface?: string
    href: string
  }) => void | Promise<void>
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
    }
    twinEnabled?: boolean
  }
}
