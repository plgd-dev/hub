import { DeviceDataType } from '@/containers/Devices/Devices.types'

export type Props = {
  data: DeviceDataType | null
  loading: boolean
  onDeleteClick: (deviceId?: string) => void
  selectedDevices: string[]
  setSelectedDevices: (data?: any) => void
  unselectRowsToken?: string | number
}

export const defaultProps = {
  data: undefined,
}
