import React, { FC, useCallback, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { useResizeDetector } from 'react-resize-detector'
import { generatePath, useNavigate } from 'react-router-dom'

import Headline from '@shared-ui/components/Atomic/Headline'
import Table from '@shared-ui/components/Atomic/TableNew'
import IconArrowDetail from '@shared-ui/components/Atomic/Icon/components/IconArrowDetail'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import { tagVariants as statusTagVariants } from '@shared-ui/components/Atomic/StatusTag/constants'
import Tooltip from '@shared-ui/components/Atomic/Tooltip'
import IconQuestion from '@shared-ui/components/Atomic/Icon/components/IconQuestion'
import StatusPill from '@shared-ui/components/Atomic/StatusPill'
import Tag from '@shared-ui/components/Atomic/Tag'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import IconLink from '@shared-ui/components/Atomic/Icon/components/IconLink'

import { Props } from './Tab3.types'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { pages } from '@/routes'
import { getAppliedConfigurationStatusStatus, getAppliedConfigurationStatusValue } from '@/containers/SnippetService/utils'

const Tab3: FC<Props> = (props) => {
    const { data, loading, isActiveTab } = props

    const { formatMessage: _ } = useIntl()
    const navigate = useNavigate()

    const { ref, height } = useResizeDetector({
        refreshRate: 500,
    })

    const getValue = useCallback((status: number) => getAppliedConfigurationStatusValue(status, _), [_])
    const getStatus = useCallback((status: number) => getAppliedConfigurationStatusStatus(status), [])

    const columns = useMemo(
        () => [
            {
                Header: _(confT.configurationName),
                accessor: 'configurationName',
                Cell: ({ value, row }: { value: string; row: any }) => (
                    <a
                        href={generatePath(pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.DETAIL.LINK, {
                            appliedConfigurationId: row.original.configurationId.id,
                            tab: '',
                        })}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(
                                generatePath(pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.DETAIL.LINK, {
                                    appliedConfigurationId: row.original.configurationId.id,
                                    tab: '',
                                })
                            )
                        }}
                    >
                        {value}
                    </a>
                ),
            },
            {
                Header: _(confT.configurationVersion),
                accessor: 'configurationId.version',
                Cell: ({ value }: { value: string }) => <StatusTag variant={statusTagVariants.NORMAL}>{value}</StatusTag>,
            },
            {
                Header: _(g.deviceName),
                accessor: 'name',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <Tooltip content={row.original.deviceId} delay={0} portalTarget={document.body}>
                        <span style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                            {`${value} - ${row.original.deviceId.substr(0, 8)} ...`}
                            <IconQuestion />
                        </span>
                    </Tooltip>
                ),
            },
            {
                Header: _(g.status),
                accessor: 'status',
                Cell: ({ value }: { value: number }) => <StatusPill label={getValue(value)} status={getStatus(value)} />,
            },
            {
                Header: _(confT.condition),
                accessor: 'conditionName',
                Cell: ({ value, row }: { value: string; row: any }) => {
                    if (row.original.onDemand) {
                        return 'on demand'
                    } else {
                        return (
                            <Tag
                                onClick={() =>
                                    `${navigate(
                                        generatePath(pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.LINK, { conditionId: row.original.conditionId.id, tab: '' })
                                    )}?version=${row.original.conditionId.version}`
                                }
                                variant={tagVariants.BLUE}
                            >
                                <IconLink />
                                <Spacer type='pl-2'>{value}</Spacer>
                            </Tag>
                        )
                    }
                },
                disableSortBy: true,
            },
            {
                Header: _(g.action),
                accessor: 'action',
                disableSortBy: true,
                Cell: ({ row }: any) => (
                    <TableActionButton
                        items={[
                            {
                                onClick: () =>
                                    navigate(
                                        generatePath(pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.DETAIL.LINK, {
                                            appliedConfigurationId: row.original.id,
                                            tab: '',
                                        })
                                    ),
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
                <Headline type='h5'>{_(confT.listOfAppliedConfigurations)}</Headline>
            </Spacer>
            <div ref={ref} style={{ flex: '1 1 auto' }}>
                <Table
                    columns={columns}
                    data={data || []}
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
                        placeholder: _(confT.noAppliedConfiguration),
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
