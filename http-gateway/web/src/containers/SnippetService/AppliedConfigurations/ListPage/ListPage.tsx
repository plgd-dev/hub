import React, { FC, useCallback, useEffect, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import StatusPill from '@shared-ui/components/Atomic/StatusPill'
import Tag from '@shared-ui/components/Atomic/Tag'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import IconLink from '@shared-ui/components/Atomic/Icon/components/IconLink'
import { states } from '@shared-ui/components/Atomic/StatusPill/constants'

import PageLayout from '@/containers/Common/PageLayout'
import { messages as confT } from '../../SnippetService.i18n'
import { useAppliedConfigurationsList } from '@/containers/SnippetService/hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import { pages } from '@/routes'
import PageListTemplate from '@/containers/Common/PageListTemplate/PageListTemplate'
import { deleteAppliedDeviceConfigApi } from '@/containers/SnippetService/rest'
import { APPLIED_CONFIGURATIONS_STATUS } from '@/containers/SnippetService/constants'

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

    const getValue = useCallback(
        (status: number) => {
            switch (status) {
                case APPLIED_CONFIGURATIONS_STATUS.ERROR:
                    return _(g.error)
                case APPLIED_CONFIGURATIONS_STATUS.PENDING:
                    return _(g.pending)
                case APPLIED_CONFIGURATIONS_STATUS.SUCCESS:
                default:
                    return _(g.success)
            }
        },
        [_]
    )

    const getStatus = useCallback((status: number) => {
        switch (status) {
            case APPLIED_CONFIGURATIONS_STATUS.ERROR:
                return states.OFFLINE
            case APPLIED_CONFIGURATIONS_STATUS.PENDING:
                return states.OCCUPIED
            case APPLIED_CONFIGURATIONS_STATUS.SUCCESS:
            default:
                return states.ONLINE
        }
    }, [])

    const columns = useMemo(
        () => [
            {
                Header: _(g.deviceName),
                accessor: 'name',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a
                        href={generatePath(pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.DETAIL.LINK, { appliedConfigurationId: row.original.id })}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(generatePath(pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.DETAIL.LINK, { appliedConfigurationId: row.original.id }))
                        }}
                    >
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(g.deviceId),
                accessor: 'deviceId',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(g.status),
                accessor: 'status',
                Cell: ({ value }: { value: number }) => <StatusPill label={getValue(value)} status={getStatus(value)} />,
            },
            {
                Header: _(confT.condition),
                accessor: 'conditionName',
                Cell: ({ value, row }: { value: string; row: any }) => (
                    <Tag
                        onClick={() =>
                            `${navigate(
                                generatePath(pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.LINK, { conditionId: row.original.conditionId.id, tab: '' })
                            )}?version=${row.original.conditionId.version}`
                        }
                        variant={tagVariants.BLUE}
                    >
                        <IconLink />
                        &nbsp;{value}
                    </Tag>
                ),
                disableSortBy: true,
            },
            {
                Header: _(g.configuration),
                accessor: 'configurationName',
                Cell: ({ value, row }: { value: string; row: any }) => (
                    <Tag
                        onClick={() =>
                            navigate(
                                generatePath(pages.SNIPPET_SERVICE.CONFIGURATIONS.DETAIL.LINK, { configurationId: row.original.configurationId.id, tab: '' })
                            )
                        }
                        variant={tagVariants.BLUE}
                    >
                        <IconLink />
                        &nbsp;{value}
                    </Tag>
                ),
                disableSortBy: true,
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <PageLayout breadcrumbs={breadcrumbs} loading={loading} title={_(confT.appliedConfiguration)}>
            <PageListTemplate
                columns={columns}
                data={data}
                deleteApiMethod={deleteAppliedDeviceConfigApi}
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
            />
        </PageLayout>
    )
}

ListPage.displayName = 'ListPage'

export default ListPage
