import { HubData } from '../../../ProvisioningRecords/ProvisioningRecordsListPage.types'

export type Props = {
    hubsData?: HubData[]
    hubId?: string
    owner: string
    value: string
}
