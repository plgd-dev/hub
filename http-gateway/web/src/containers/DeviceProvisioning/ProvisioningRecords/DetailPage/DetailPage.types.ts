import { DataType } from '@/containers/DeviceProvisioning/ProvisioningRecords/ProvisioningRecordsListPage.types'
import { ItemType } from '@shared-ui/components/Atomic/ContentMenu/ContentMenu.types'

export type Props = {
    provisioningRecord: DataType
    onItemClick?: (item: ItemType) => void
    currentTab: any
}
