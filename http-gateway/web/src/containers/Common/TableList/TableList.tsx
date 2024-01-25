import React, { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useResizeDetector } from 'react-resize-detector'
import { useIntl } from 'react-intl'
import ReactDOM from 'react-dom'

import Table, { TableSelectionPanel } from '@shared-ui/components/Atomic/TableNew'
import { useIsMounted } from '@shared-ui/common/hooks'
import Button from '@shared-ui/components/Atomic/Button'
import AppContext from '@shared-ui/app/share/AppContext'

import { defaultPops, Props } from './TableList.types'
import { messages as g } from '../../Global.i18n'

const DEFAULT_PAGE_SIZE = 10

const TableList: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { collapsed } = useContext(AppContext)
    const {
        columns,
        data,
        defaultPageSize,
        defaultSortBy,
        i18n,
        iframeMode,
        paginationPortalTargetId,
        primaryAttribute,
        onDeleteClick,
        unselectRowsToken,
        tableSelectionPanelPortalTargetId,
    } = {
        ...defaultPops,
        ...props,
    }
    const { ref, height } = useResizeDetector({
        refreshRate: 500,
    })

    const [isAllSelected, setIsAllSelected] = useState(false)
    const [selected, setSelected] = useState([])

    const selectedCount = useMemo(() => Object.keys(selected).length, [selected])
    const validData = useCallback((data: any) => (!data || data[0] === undefined ? [] : data), [])
    const isMounted = useIsMounted()

    useEffect(() => {
        setIsAllSelected(false)
        setSelected([])
    }, [unselectRowsToken])

    return (
        <div
            ref={ref}
            style={{
                height: '100%',
                width: '100%',
                display: 'flex',
                flexDirection: 'column',
                overflow: 'hidden',
            }}
        >
            <Table
                columns={columns}
                data={validData(data)}
                defaultPageSize={defaultPageSize ?? DEFAULT_PAGE_SIZE}
                defaultSortBy={defaultSortBy}
                height={height}
                i18n={{
                    search: _(g.search),
                }}
                onRowsSelect={(isAllRowsSelected, selection) => {
                    isAllRowsSelected !== isAllSelected && setIsAllSelected && setIsAllSelected(isAllRowsSelected)
                    setSelected(selection)
                }}
                paginationPortalTargetId={paginationPortalTargetId}
                primaryAttribute={primaryAttribute}
                unselectRowsToken={unselectRowsToken}
            />

            {isMounted &&
                tableSelectionPanelPortalTargetId &&
                document.getElementById(tableSelectionPanelPortalTargetId) &&
                ReactDOM.createPortal(
                    <TableSelectionPanel
                        actionPrimary={
                            <Button onClick={() => onDeleteClick(isAllSelected, selected)} variant='primary'>
                                {_(g.delete)}
                            </Button>
                        }
                        i18n={{
                            select: _(g.select),
                        }}
                        iframeMode={iframeMode}
                        leftPanelCollapsed={collapsed}
                        selectionInfo={`${selectedCount} ${selectedCount > 1 ? i18n.multiSelected : i18n.singleSelected}`}
                        show={selectedCount > 0}
                    />,
                    document.getElementById(tableSelectionPanelPortalTargetId) as Element
                )}
        </div>
    )
}

TableList.displayName = 'TableList'

export default TableList
