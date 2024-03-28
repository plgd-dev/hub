import { HubData } from '../../../ProvisioningRecords/ProvisioningRecordsListPage.types'

export type Props = {
    deviceId?: string
    hubId?: string
    hubsData?: HubData[]
    owner: string
    value: string
}
