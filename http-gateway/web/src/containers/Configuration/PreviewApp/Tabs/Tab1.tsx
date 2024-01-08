import React, { FC, useEffect, useMemo, useState } from 'react'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import TableList from '@/containers/Common/TableList/TableList'
import { useIntl } from 'react-intl'
import { messages as g } from '@/containers/Global.i18n'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { IconArrowDetail, IconTrash } from '@shared-ui/components/Atomic'

const LIST_COLUMNS_COUNT = 5
const LIST_ROWS_COUNT = 25

type Props = {
    isActiveTab: boolean
}

const Tab1: FC<Props> = (props) => {
    const { isActiveTab } = props
    const { formatMessage: _ } = useIntl()

    const [unselectRowsToken, setUnselectRowsToken] = useState(0)

    useEffect(() => {
        setUnselectRowsToken((prev) => prev + 1)
    }, [isActiveTab])

    const columns = useMemo(() => {
        const cols: any = Array(LIST_COLUMNS_COUNT)
            .fill(null)
            .map((_, key) => ({
                Header: `Column ${key + 1}`,
                accessor: `column${key + 1}`,
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            }))

        cols.push({
            Header: _(g.action),
            accessor: 'action',
            disableSortBy: true,
            Cell: () => (
                <TableActionButton
                    items={[
                        {
                            onClick: () => console.log('Delete'),
                            label: _(t.delete),
                            icon: <IconTrash />,
                        },
                        {
                            onClick: () => console.log('Detail'),
                            label: _(t.view),
                            icon: <IconArrowDetail />,
                        },
                    ]}
                />
            ),
            className: 'actions',
        })

        return cols
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const data = useMemo(
        () =>
            Array(LIST_ROWS_COUNT)
                .fill(null)
                .map((_, key) => {
                    let row = {}
                    Array(LIST_COLUMNS_COUNT)
                        .fill(null)
                        .forEach((_, c) => {
                            // @ts-ignore
                            row[`column${c + 1}`] = `Row ${key + 1} : Column ${c + 1}`
                        })

                    return row
                }),
        []
    )

    return (
        <TableList
            columns={columns}
            data={data}
            defaultSortBy={[
                {
                    id: 'column1',
                    desc: false,
                },
            ]}
            i18n={{
                multiSelected: _(t.devices),
                singleSelected: _(t.device),
            }}
            iframeMode='absolute'
            onDeleteClick={() => console.log('Delete')}
            paginationPortalTargetId={isActiveTab ? 'paginationPortalTargetPreviewApp' : undefined}
            primaryAttribute='column1'
            tableSelectionPanelPortalTargetId='rootPreviewApp'
            unselectRowsToken={unselectRowsToken}
        />
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
