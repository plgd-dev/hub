import React, { FC, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Button from '@shared-ui/components/Atomic/Button'
import { IconPlus } from '@shared-ui/components/Atomic/Icon'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'

import PageLayout from '@/containers/Common/PageLayout'
import { messages as confT } from '../../SnippetService.i18n'
import { useConfigurationList } from '@/containers/SnippetService/hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import { pages } from '@/routes'
import PageListTemplate from '@/containers/Common/PageListTemplate/PageListTemplate'
import { deleteConfigurationsApi, invokeConfigurationApi } from '@/containers/SnippetService/rest'
import { getConfigurationsPageListI18n } from '@/containers/SnippetService/utils'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import InvokeModal from '@/containers/SnippetService/Configurations/InvokeModal'
import testId from '@/testId'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import { tagVariants as statusTagVariants } from '@shared-ui/components/Atomic/StatusTag/constants'

const ListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    const { data, loading, error, refresh } = useConfigurationList()
    const navigate = useNavigate()

    const [showInvoke, setShowInvoke] = useState<string | undefined>(undefined)
    const [pageLoading, setPageLoading] = useState(false)

    // eslint-disable-next-line react-hooks/exhaustive-deps
    const breadcrumbs = useMemo(() => [{ label: _(confT.snippetService), link: pages.SNIPPET_SERVICE.LINK }, { label: _(confT.configurations) }], [])

    useEffect(() => {
        error &&
            Notification.error(
                { title: _(confT.configurationsError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONFIGURATIONS_LIST_PAGE_ERROR }
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
                        href={generatePath(pages.SNIPPET_SERVICE.CONFIGURATIONS.DETAIL.LINK, { configurationId: row.original.id, tab: '' })}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(generatePath(pages.SNIPPET_SERVICE.CONFIGURATIONS.DETAIL.LINK, { configurationId: row.original.id, tab: '' }))
                        }}
                    >
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(g.id),
                accessor: 'id',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(confT.timeLastModification),
                accessor: 'timestamp',
                Cell: ({ value }: { value: string | number }) => <DateFormat value={value} />,
            },
            {
                Header: _(g.version),
                accessor: 'version',
                Cell: ({ value }: { value: string | number }) => <StatusTag variant={statusTagVariants.NORMAL}>{value}</StatusTag>,
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const handleInvoke = async (deviceIds: string[], force: boolean) => {
        setPageLoading(true)

        try {
            if (showInvoke) {
                const results = deviceIds.map((deviceId) => {
                    return invokeConfigurationApi(showInvoke, {
                        deviceId,
                        force,
                    })
                })

                await Promise.all(results).then(() => {
                    setPageLoading(false)

                    Notification.success(
                        {
                            title: _(confT.conditionInvokeSuccess),
                            message: _(confT.conditionInvokeSuccessMessage),
                        },
                        { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONFIGURATIONS_LIST_PAGE_INVOKE_SUCCESS }
                    )

                    setShowInvoke(undefined)
                })
            }
        } catch (error: any) {
            let e = error
            if (!(error instanceof Error)) {
                e = new Error(error)
            }
            Notification.error(
                { title: _(confT.conditionInvokeError), message: e.message },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONFIGURATIONS_LIST_PAGE_INVOKE_ERROR }
            )
            setPageLoading(false)
        }
    }

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={
                <Button
                    dataTestId={testId.snippetService.configurations.list.addConfigurationButton}
                    icon={<IconPlus />}
                    onClick={() => navigate(generatePath(pages.SNIPPET_SERVICE.CONFIGURATIONS.ADD.LINK, { tab: '' }))}
                    variant='primary'
                >
                    {_(confT.configuration)}
                </Button>
            }
            loading={loading || pageLoading}
            title={_(confT.configurations)}
        >
            <PageListTemplate
                columns={columns}
                data={data}
                deleteApiMethod={deleteConfigurationsApi}
                i18n={getConfigurationsPageListI18n(_)}
                loading={loading}
                onDeletionError={(e) => {
                    Notification.error(
                        { title: _(confT.configurationsError), message: getApiErrorMessage(e) },
                        { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONFIGURATIONS_LIST_PAGE_DELETE_ERROR }
                    )
                }}
                onDeletionSuccess={() => {
                    Notification.success(
                        { title: _(confT.configurationsDeleted), message: _(confT.configurationsDeletedMessage) },
                        { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONFIGURATIONS_LIST_PAGE_DELETE_SUCCESS }
                    )
                }}
                onDetailClick={(id: string) => navigate(generatePath(pages.SNIPPET_SERVICE.CONFIGURATIONS.DETAIL.LINK, { configurationId: id, tab: '' }))}
                onInvoke={(id: string) => setShowInvoke(id)}
                refresh={() => refresh()}
                tableDataTestId={testId.snippetService.configurations.list.table}
            />
            <InvokeModal
                dataTestId={testId.snippetService.configurations.list.invokeModal}
                handleClose={() => setShowInvoke(undefined)}
                handleInvoke={handleInvoke}
                show={!!showInvoke}
            />
        </PageLayout>
    )
}

ListPage.displayName = 'ListPage'

export default ListPage
