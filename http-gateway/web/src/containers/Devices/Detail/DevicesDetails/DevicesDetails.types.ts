import { DeviceDataType } from '@/containers/Devices/Devices.types'
import { devicesStatuses } from '../../constants'

export type DevicesDetailMetaDataStatusValueType = typeof devicesStatuses[keyof typeof devicesStatuses]

export type Props = {
    data: DeviceDataType
    isTwinEnabled: boolean
    loading: boolean
    setTwinSynchronization: () => void
    twinSyncLoading: boolean
}
