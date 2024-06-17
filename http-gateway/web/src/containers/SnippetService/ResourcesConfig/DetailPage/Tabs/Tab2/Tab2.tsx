import React, { FC, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { useResizeDetector } from 'react-resize-detector'
import { generatePath, useNavigate } from 'react-router-dom'

import Headline from '@shared-ui/components/Atomic/Headline'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import { states } from '@shared-ui/components/Atomic/StatusPill/constants'
import StatusPill from '@shared-ui/components/Atomic/StatusPill'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import IconArrowDetail from '@shared-ui/components/Atomic/Icon/components/IconArrowDetail'
import Table from '@shared-ui/components/Atomic/TableNew'

import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props } from './Tab2.types'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import { pages } from '@/routes'

const Tab2: FC<Props> = (props) => {
    const { data, loading, isActiveTab } = props

    const { formatMessage: _ } = useIntl()
    const navigate = useNavigate()

    const columns = useMemo(
        () => [
            {
                Header: _(g.name),
                accessor: 'name',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a href='#'>
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(g.status),
                accessor: 'enabled',
                Cell: ({ value }: { value: boolean }) => (
                    <StatusPill label={value ? _(g.enabled) : _(g.disabled)} status={value ? states.ONLINE : states.OFFLINE} />
                ),
            },
            {
                Header: _(g.lastModified),
                accessor: 'timestamp',
                Cell: ({ value }: { value: string }) => <DateFormat value={value} />,
            },
            {
                Header: _(g.action),
                accessor: 'action',
                disableSortBy: true,
                Cell: ({ row }: any) => (
                    <TableActionButton
                        items={[
                            {
                                onClick: () => navigate(generatePath(pages.CONDITIONS.CONDITIONS.DETAIL.LINK, { conditionId: row.original.id, tab: '' })),
                                label: _(g.view),
                                icon: <IconArrowDetail />,
                            },
                        ]}
                    />
                ),
                className: 'actions',
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const { ref, height } = useResizeDetector({
        refreshRate: 500,
    })

    return (
        <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            <Spacer type='mb-6'>
                <Headline type='h5'>{_(confT.listOfConditions)}</Headline>
            </Spacer>
            <div ref={ref} style={{ flex: '1 1 auto' }}>
                <Table
                    columns={columns}
                    data={data}
                    defaultPageSize={10}
                    defaultSortBy={[
                        {
                            id: 'href',
                            desc: false,
                        },
                    ]}
                    globalSearch={false}
                    height={height}
                    i18n={{
                        search: '',
                        placeholder: _(confT.noConditions),
                    }}
                    loading={loading}
                    paginationPortalTargetId={isActiveTab ? 'paginationPortalTarget' : undefined}
                />
            </div>
        </div>
    )
}

Tab2.displayName = 'Tab2'

export default Tab2
