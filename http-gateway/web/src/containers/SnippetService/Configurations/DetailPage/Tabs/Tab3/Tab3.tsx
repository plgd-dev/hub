import React, { FC, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { useResizeDetector } from 'react-resize-detector'
import { generatePath, useNavigate } from 'react-router-dom'

import Headline from '@shared-ui/components/Atomic/Headline'
import Table from '@shared-ui/components/Atomic/TableNew'
import IconArrowDetail from '@shared-ui/components/Atomic/Icon/components/IconArrowDetail'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'

import { Props } from './Tab3.types'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { messages as g } from '@/containers/Global.i18n'

import { pages } from '@/routes'
import Spacer from '@shared-ui/components/Atomic/Spacer'

const Tab3: FC<Props> = (props) => {
    const { data, loading, isActiveTab } = props

    const { formatMessage: _ } = useIntl()
    const navigate = useNavigate()

    const { ref, height } = useResizeDetector({
        refreshRate: 500,
    })

    const columns = useMemo(
        () => [
            {
                Header: _(g.deviceName),
                accessor: 'name',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a href='#'>
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(g.deviceId),
                accessor: 'id',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a href='#'>
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(g.status),
                accessor: 'status',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a href='#'>
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(g.action),
                accessor: 'action',
                disableSortBy: true,
                Cell: ({ row }: any) => (
                    <TableActionButton
                        items={[
                            {
                                onClick: () => navigate(generatePath(pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.LINK, { conditionId: row.original.id, tab: '' })),
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

    return (
        <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            <Spacer type='mb-6'>
                <Headline type='h5'>{_(confT.listOfDeviceAppliedConfigurations)}</Headline>
            </Spacer>
            <div ref={ref} style={{ flex: '1 1 auto' }}>
                <Table
                    columns={columns}
                    data={[]}
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
                        placeholder: _(confT.noAppliedDeviceConfiguration),
                    }}
                    loading={loading}
                    paginationPortalTargetId={isActiveTab ? 'paginationPortalTarget' : undefined}
                />
            </div>
        </div>
    )
}

Tab3.displayName = 'Tab3'

export default Tab3
