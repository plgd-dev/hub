export type Props = {
    columns: any
    data: any
    defaultPageSize?: number
    defaultSortBy: {
        id: string
        desc?: boolean
    }[]
    globalSearch?: boolean
    i18n: {
        singleSelected: string
        multiSelected: string
        tablePlaceholder?: string
    }
    iframeMode?: boolean | 'absolute'
    loading?: boolean
    onDeleteClick: (isAllSelected: boolean, selected: string[]) => void
    paginationPortalTargetId?: string
    primaryAttribute?: string
    tableSelectionPanelPortalTargetId?: string
    unselectRowsToken?: string | number
}

export const defaultPops: Partial<Props> = {
    globalSearch: true,
    paginationPortalTargetId: 'paginationPortalTarget',
    primaryAttribute: 'id',
    tableSelectionPanelPortalTargetId: 'root',
}
