import { RemoteClientType } from '@/containers/RemoteClients/slice'

export type Props = {
    data: RemoteClientType[]
    handleOpenDeleteModal: (data?: any) => void
    isAllSelected: boolean
    selectedClients: string[]
    setIsAllSelected?: (isAllSelected: boolean) => void
    setSelectedClients: (data?: any) => void
    unselectRowsToken: number
}
