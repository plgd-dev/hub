export type Props = {
    columns: any
    data: any
    dataTestId?: string
    deleteApiMethod?: (selected: string[]) => Promise<any[]>
    globalSearch?: boolean
    i18n: {
        action: string
        delete: string
        name: string
        loading: string
        cancel: string
        id: string
        invoke?: string
        deleteModalSubtitle: string
        deleteModalTitle: (selected: number) => string
        view: string
        singleSelected: string
        multiSelected: string
        tablePlaceholder: string
    }
    loading?: boolean
    onDeletionError: (e: any) => void
    onDeletionSuccess: () => void
    onDetailClick: (id: string) => void
    onInvoke?: (id: string) => void
    refresh: () => void
    tableDataTestId?: string
}

export const defaultProps: Partial<Props> = {
    globalSearch: true,
}
