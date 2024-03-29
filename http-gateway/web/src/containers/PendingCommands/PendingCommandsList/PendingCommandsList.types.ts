export type PendingCommandsListRefType = {
    setDetailsModalData: (data: ModalData | null) => void
    setConfirmModalData: (data: ConfirmModalData | null) => void
}

export type ModalData = {
    content: any
    commandType?: any
}

export type ConfirmModalData = {
    deviceId: string
    href: string
    correlationId: string
}

export type Props = {
    columns: any
    deviceId?: string
    embedded?: boolean
    isPage?: boolean
    onLoading?: (loadingPendingCommands: boolean) => void
}
