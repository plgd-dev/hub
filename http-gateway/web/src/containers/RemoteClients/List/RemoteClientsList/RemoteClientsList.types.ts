import { RemoteClientType } from '@shared-ui/app/clientApp/RemoteClients/RemoteClients.types'

export type Props = {
    data: RemoteClientType[]
    handleOpenDeleteModal: (data?: any) => void
    handleOpenEditModal: (data?: any) => void
    isAllSelected: boolean
    selectedClients: string[]
    setIsAllSelected?: (isAllSelected: boolean) => void
    setSelectedClients: (data?: any) => void
    unselectRowsToken: number
}
