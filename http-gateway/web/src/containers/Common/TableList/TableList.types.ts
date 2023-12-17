export type Props = {
    columns: any
    data: any
    defaultPageSize?: number
    defaultSortBy: {
        id: string
        desc?: boolean
    }[]
    i18n: {
        singleSelected: string
        multiSelected: string
    }

    onDeleteClick: (isAllSelected: boolean, selected: string[]) => void
    paginationPortalTargetId?: string
    primaryAttribute?: string
    unselectRowsToken?: string | number
}

export const defaultPops: Partial<Props> = {
    paginationPortalTargetId: 'paginationPortalTarget',
    primaryAttribute: 'id',
}
