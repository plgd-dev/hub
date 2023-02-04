import {
  DeviceDataType,
  ResourcesType,
} from '@/containers/Devices/Devices.types'
import { devicesStatuses } from '../../constants'

export type DevicesDetailMetaDataStatusValueType =
  typeof devicesStatuses[keyof typeof devicesStatuses]

export type Props = {
  data: DeviceDataType
  deviceId: string
  deviceOnboardingResourceData: any
  isOwned: boolean
  loading: boolean
  onboardResourceLoading: boolean
  resources: ResourcesType[]
  setTwinSynchronization: () => void
  twinSyncLoading: boolean
}
