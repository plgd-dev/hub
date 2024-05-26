import { DataType } from '@/containers/DeviceProvisioning/ProvisioningRecords/ProvisioningRecordsListPage.types'

export type Props = {
    data: DataType
    isDeviceMode?: boolean
    refs?: {
        cloud: any
        ownership: any
        time: any
    }
}
