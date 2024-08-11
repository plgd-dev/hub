import { ReactNode } from 'react'

import { DeleteInformationType } from '@shared-ui/components/Atomic/Modal/components/DeleteModal/DeleteModal.types'

export type Props = {
    customButton?: ReactNode
    deleteApiMethod?: (ids: string[]) => Promise<any[]>
    deleteInformation?: DeleteInformationType[]
    id: string
    loading: boolean
    i18n: {
        cancel: string
        delete: string
        deleting: string
        id: string
        name: string
        subTitle: string
        title: string
    }
    onDeleteSuccess?: () => void
    onDeleteError?: (error: any) => void
    testIds?: {
        deleteButton?: string
        deleteButtonCancel?: string
        deleteButtonConfirm?: string
        deleteModal?: string
    }
}
