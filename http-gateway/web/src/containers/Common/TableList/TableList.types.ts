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
    iframeMode?: boolean | 'absolute'
    onDeleteClick: (isAllSelected: boolean, selected: string[]) => void
    paginationPortalTargetId?: string
    primaryAttribute?: string
    unselectRowsToken?: string | number
    tableSelectionPanelPortalTargetId?: string
}

export const defaultPops: Partial<Props> = {
    paginationPortalTargetId: 'paginationPortalTarget',
    primaryAttribute: 'id',
    tableSelectionPanelPortalTargetId: 'root',
}
