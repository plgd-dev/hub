import { UpdateProvisioningRecordNameBodyType } from '../../rest'

export type Props = {
    enrollmentGroupData: UpdateProvisioningRecordNameBodyType
    enrollmentGroupId?: string
    id?: string
    loading?: boolean
    name?: string
    refresh: () => void
}
