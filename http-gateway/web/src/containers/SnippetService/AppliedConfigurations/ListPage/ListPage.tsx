import React, { FC, useCallback, useEffect, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import StatusPill from '@shared-ui/components/Atomic/StatusPill'
import Tag from '@shared-ui/components/Atomic/Tag'
import IconLink from '@shared-ui/components/Atomic/Icon/components/IconLink'
import { tagVariants as statusTagVariants } from '@shared-ui/components/Atomic/StatusTag/constants'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import Tooltip from '@shared-ui/components/Atomic/Tooltip'
import IconQuestion from '@shared-ui/components/Atomic/Icon/components/IconQuestion'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'

import PageLayout from '@/containers/Common/PageLayout'
import { messages as confT } from '../../SnippetService.i18n'
import { useAppliedConfigurationsList } from '@/containers/SnippetService/hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import { pages } from '@/routes'
import PageListTemplate from '@/containers/Common/PageListTemplate/PageListTemplate'
import { deleteAppliedConfigurationApi } from '@/containers/SnippetService/rest'
import { getAppliedConfigurationStatusStatus, getAppliedConfigurationStatusValue } from '@/containers/SnippetService/utils'
import { AppliedConfigurationStatusType } from '@/containers/SnippetService/ServiceSnippet.types'
import testId from '@/testId'

const ListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    const { data, loading, error, refresh } = useAppliedConfigurationsList()
    const navigate = useNavigate()

    const breadcrumbs = useMemo(() => [{ label: _(confT.snippetService), link: pages.SNIPPET_SERVICE.LINK }, { label: _(confT.appliedConfiguration) }], [_])

    useEffect(() => {
        error &&
            Notification.error(
                { title: _(confT.appliedConfigurationsError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_CONFIGURATIONS_LIST_PAGE_ERROR }
            )
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    const getValue = useCallback((status: AppliedConfigurationStatusType) => getAppliedConfigurationStatusValue(status, _), [_])
    const getStatus = useCallback((status: AppliedConfigurationStatusType) => getAppliedConfigurationStatusStatus(status), [])

    const columns = useMemo(
        () => [
            {
                Header: _(confT.configurationName),
                accessor: 'configurationName',
                Cell: ({ value, row }: { value: string; row: any }) => (
                    <a
                        data-test-id={`${testId.snippetService.appliedConfigurations.list.table}-row-${row.id}-name`}
                        href={generatePath(pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.DETAIL.LINK, { appliedConfigurationId: row.original.id, tab: '' })}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(
                                generatePath(pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.DETAIL.LINK, { appliedConfigurationId: row.original.id, tab: '' })
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
                Cell: ({ value }: { value: AppliedConfigurationStatusType }) => <StatusPill label={getValue(value)} status={getStatus(value)} />,
            },
            {
                Header: _(confT.condition),
                accessor: 'conditionName',
                Cell: ({ value, row }: { value: string; row: any }) => {
                    if (row.original.onDemand) {
                        return <StatusTag variant={statusTagVariants.NORMAL}>{_(confT.onDemand)}</StatusTag>
                    } else {
                        return (
                            <Tag
                                dataTestId={`${testId.snippetService.appliedConfigurations.list.table}-row-${row.id}-condition`}
                                onClick={() =>
                                    `${navigate(
                                        generatePath(pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.LINK, { conditionId: row.original.conditionId.id, tab: '' })
                                    )}?version=${row.original.conditionId.version}`
                                }
                                variant={tagVariants.BLUE}
                            >
                                <IconLink />
                                <Spacer type='pl-2'>{value}</Spacer>
                                <Spacer type='ml-2'>(v.{row.original?.conditionId.version})</Spacer>
                            </Tag>
                        )
                    }
                },
                disableSortBy: true,
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <PageLayout breadcrumbs={breadcrumbs} loading={loading} title={_(confT.appliedConfigurations)}>
            <PageListTemplate
                columns={columns}
                data={data}
                dataTestId={testId.snippetService.appliedConfigurations.list.pageTemplate}
                deleteApiMethod={deleteAppliedConfigurationApi}
                i18n={{
                    singleSelected: _(confT.appliedConfiguration),
                    multiSelected: _(confT.appliedConfigurations),
                    tablePlaceholder: _(confT.noAppliedConfiguration),
                    id: _(g.id),
                    name: _(g.name),
                    cancel: _(g.cancel),
                    action: _(g.action),
                    delete: _(g.delete),
                    loading: _(g.loading),
                    deleteModalSubtitle: _(g.undoneAction),
                    view: _(g.view),
                    deleteModalTitle: (selected: number) =>
                        selected === 1 ? _(confT.deleteAppliedConfigurationMessage) : _(confT.deleteAppliedConfigurationsMessage, { count: selected }),
                }}
                loading={loading}
                onDeletionError={(e) => {
                    Notification.error(
                        { title: _(confT.appliedConfigurationError), message: getApiErrorMessage(e) },
                        { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_CONFIGURATIONS_LIST_PAGE_DELETE_ERROR }
                    )
                }}
                onDeletionSuccess={() => {
                    Notification.success(
                        { title: _(confT.appliedConfigurationDeleted), message: _(confT.appliedConfigurationDeletedMessage) },
                        { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_CONFIGURATIONS_LIST_PAGE_DELETE_SUCCESS }
                    )
                }}
                onDetailClick={(id: string) =>
                    navigate(generatePath(pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.DETAIL.LINK, { appliedConfigurationId: id, tab: '' }))
                }
                refresh={() => refresh()}
                tableDataTestId={testId.snippetService.appliedConfigurations.list.table}
            />
        </PageLayout>
    )
}

ListPage.displayName = 'ListPage'

export default ListPage
