import React, { FC, useEffect, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Button from '@shared-ui/components/Atomic/Button'
import { IconPlus } from '@shared-ui/components/Atomic/Icon'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'

import PageLayout from '@/containers/Common/PageLayout'
import { messages as confT } from '../../SnippetService.i18n'
import { useResourcesConfigList } from '@/containers/SnippetService/hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import { pages } from '@/routes'
import PageListTemplate from '@/containers/Common/PageListTemplate/PageListTemplate'
import { deleteResourcesConfigApi } from '@/containers/SnippetService/rest'
import { getConfigResourcesPageListI18n } from '@/containers/SnippetService/utils'
import DateFormat from '@/containers/PendingCommands/DateFormat'

const ListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    const { data, loading, error, refresh } = useResourcesConfigList()
    const navigate = useNavigate()

    // eslint-disable-next-line react-hooks/exhaustive-deps
    const breadcrumbs = useMemo(() => [{ label: _(confT.conditions), link: '/conditions' }, { label: _(confT.resourcesConfiguration) }], [])

    useEffect(() => {
        error &&
            Notification.error(
                { title: _(confT.resourcesConfigurationError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_RESOURCES_CONFIGURATION_LIST_PAGE_ERROR }
            )
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    console.log(data)

    const columns = useMemo(
        () => [
            {
                Header: _(g.name),
                accessor: 'name',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a
                        href={generatePath(pages.CONDITIONS.RESOURCES_CONFIG.DETAIL.LINK, { resourcesConfigId: row.original.id, tab: '' })}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(generatePath(pages.CONDITIONS.RESOURCES_CONFIG.DETAIL.LINK, { resourcesConfigId: row.original.id, tab: '' }))
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
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={
                <Button icon={<IconPlus />} onClick={() => navigate(generatePath(pages.CONDITIONS.RESOURCES_CONFIG.ADD.LINK, { tab: '' }))} variant='primary'>
                    {_(confT.configuration)}
                </Button>
            }
            loading={loading}
            title={_(confT.resourcesConfiguration)}
        >
            <PageListTemplate
                columns={columns}
                data={data}
                deleteApiMethod={deleteResourcesConfigApi}
                i18n={getConfigResourcesPageListI18n(_)}
                loading={loading}
                onDeletionError={(e) => {
                    Notification.error(
                        { title: _(confT.resourcesConfigurationError), message: getApiErrorMessage(e) },
                        { notificationId: notificationId.HUB_SNIPPET_SERVICE_RESOURCES_CONFIGURATION_LIST_PAGE_DELETE_ERROR }
                    )
                }}
                onDeletionSuccess={() => {
                    Notification.success(
                        { title: _(confT.resourcesConfigurationDeleted), message: _(confT.resourcesConfigurationDeletedMessage) },
                        { notificationId: notificationId.HUB_SNIPPET_SERVICE_RESOURCES_CONFIGURATION_LIST_PAGE_DELETE_SUCCESS }
                    )
                }}
                onDetailClick={(id: string) => navigate(generatePath(pages.CONDITIONS.RESOURCES_CONFIG.DETAIL.LINK, { resourcesConfigId: id, tab: '' }))}
                refresh={() => refresh()}
            />
        </PageLayout>
    )
}

ListPage.displayName = 'ListPage'

export default ListPage
