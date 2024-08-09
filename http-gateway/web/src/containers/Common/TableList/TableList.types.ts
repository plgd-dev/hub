export type Props = {
    columns: any
    data: any
    dataTestId?: string
    defaultPageSize?: number
    defaultSortBy: {
        id: string
        desc?: boolean
    }[]
    globalSearch?: boolean
    i18n: {
        delete?: string
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
