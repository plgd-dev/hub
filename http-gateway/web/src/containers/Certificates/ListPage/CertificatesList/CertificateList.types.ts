export type Props = {
    data: []
    loading: boolean
    onDelete: (certificateIds: string[]) => Promise<any[]>
    onView: (certificateId: string, parsedCert: any) => void
    refresh: () => void
    error?: string
    notificationIds: {
        deleteError: string
        deleteSuccess: string
        parsingError: string
    }
    deleting: boolean
    setDeleting: (deleting: boolean) => void
}
