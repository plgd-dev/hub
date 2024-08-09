import React, { FC, useEffect, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Button from '@shared-ui/components/Atomic/Button'
import { IconPlus } from '@shared-ui/components/Atomic/Icon'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import IconLink from '@shared-ui/components/Atomic/Icon/components/IconLink'
import Tag from '@shared-ui/components/Atomic/Tag'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import StatusPill from '@shared-ui/components/Atomic/StatusPill'
import { states } from '@shared-ui/components/Atomic/StatusPill/constants'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import { tagVariants as statusTagVariants } from '@shared-ui/components/Atomic/StatusTag/constants'

import PageLayout from '@/containers/Common/PageLayout'
import { messages as confT } from '../../SnippetService.i18n'
import { useConditionsList } from '@/containers/SnippetService/hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import { pages } from '@/routes'
import PageListTemplate from '@/containers/Common/PageListTemplate/PageListTemplate'
import { deleteConditionsApi } from '@/containers/SnippetService/rest'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import testId from '@/testId'

const ListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    const { data, loading, error, refresh } = useConditionsList()

    const navigate = useNavigate()

    // eslint-disable-next-line react-hooks/exhaustive-deps
    const breadcrumbs = useMemo(() => [{ label: _(confT.snippetService), link: pages.SNIPPET_SERVICE.LINK }, { label: _(confT.conditions) }], [])

    useEffect(() => {
        error &&
            Notification.error(
                { title: _(confT.conditionsError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONDITIONS_LIST_PAGE_ERROR }
            )
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    const columns = useMemo(
        () => [
            {
                Header: _(g.name),
                accessor: 'name',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a
                        data-test-id={`${testId.snippetService.conditions.list.table}-row-${row.id}-name`}
                        href={generatePath(pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.LINK, { conditionId: row.original.id, tab: '' })}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(generatePath(pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.LINK, { conditionId: row.original.id, tab: '' }))
                        }}
                    >
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
                Header: _(g.version),
                accessor: 'version',
                Cell: ({ value }: { value: string | number }) => <StatusTag variant={statusTagVariants.NORMAL}>{value}</StatusTag>,
            },
            {
                Header: _(g.link),
                accessor: 'configurationId',
                Cell: ({ value, row }: { value: string; row: any }) => {
                    if (row.original.configurationId && row.original.configurationName) {
                        return (
                            <Tag
                                onClick={() =>
                                    navigate(
                                        generatePath(pages.SNIPPET_SERVICE.CONFIGURATIONS.DETAIL.LINK, {
                                            configurationId: row.original.configurationId,
                                            tab: '',
                                        })
                                    )
                                }
                                variant={tagVariants.BLUE}
                            >
                                <IconLink />
                                <Spacer type='ml-2'>{row.original.configurationName}</Spacer>
                            </Tag>
                        )
                    }

                    return '-'
                },
                disableSortBy: true,
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={
                <Button
                    dataTestId={testId.snippetService.conditions.list.addButton}
                    icon={<IconPlus />}
                    onClick={() => navigate(generatePath(pages.SNIPPET_SERVICE.CONDITIONS.ADD.LINK, { tab: '' }))}
                    variant='primary'
                >
                    {_(confT.conditions)}
                </Button>
            }
            loading={loading}
            title={_(confT.conditions)}
        >
            <PageListTemplate
                columns={columns}
                data={data}
                dataTestId={testId.snippetService.conditions.list.pageTemplate}
                deleteApiMethod={deleteConditionsApi}
                i18n={{
                    singleSelected: _(confT.condition),
                    multiSelected: _(confT.conditions),
                    tablePlaceholder: _(confT.noConditions),
                    id: _(g.id),
                    name: _(g.name),
                    cancel: _(g.cancel),
                    action: _(g.action),
                    invoke: _(g.invoke),
                    delete: _(g.delete),
                    loading: _(g.loading),
                    deleteModalSubtitle: _(g.undoneAction),
                    view: _(g.view),
                    deleteModalTitle: (selected: number) =>
                        selected === 1 ? _(confT.deleteConditionMessage) : _(confT.deleteConditionsMessage, { count: selected }),
                }}
                loading={loading}
                onDeletionError={(e) => {
                    Notification.error(
                        { title: _(confT.conditionsError), message: getApiErrorMessage(e) },
                        { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONFIGURATIONS_LIST_PAGE_DELETE_ERROR }
                    )
                }}
                onDeletionSuccess={() => {
                    Notification.success(
                        { title: _(confT.conditionDeleted), message: _(confT.conditionsDeletedMessage) },
                        { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONFIGURATIONS_LIST_PAGE_DELETE_SUCCESS }
                    )
                }}
                onDetailClick={(id: string) => navigate(generatePath(pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.LINK, { conditionId: id, tab: '' }))}
                refresh={() => refresh()}
                tableDataTestId={testId.snippetService.conditions.list.table}
            />
        </PageLayout>
    )
}

ListPage.displayName = 'ListPage'

export default ListPage
