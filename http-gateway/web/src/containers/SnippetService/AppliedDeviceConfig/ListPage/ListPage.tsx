import React, { FC, useEffect, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Button from '@shared-ui/components/Atomic/Button'
import { IconPlus } from '@shared-ui/components/Atomic/Icon'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import StatusPill from '@shared-ui/components/Atomic/StatusPill'

import PageLayout from '@/containers/Common/PageLayout'
import { messages as confT } from '../../SnippetService.i18n'
import { useAppliedDeviceConfigList } from '@/containers/SnippetService/hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import { pages } from '@/routes'
import PageListTemplate from '@/containers/Common/PageListTemplate/PageListTemplate'
import { deleteAppliedDeviceConfigApi } from '@/containers/SnippetService/rest'
import Tag from '@shared-ui/components/Atomic/Tag'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import IconLink from '@shared-ui/components/Atomic/Icon/components/IconLink'

const ListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    const { data, loading, error, refresh } = useAppliedDeviceConfigList()
    const navigate = useNavigate()

    // eslint-disable-next-line react-hooks/exhaustive-deps
    const breadcrumbs = useMemo(() => [{ label: _(confT.conditions), link: pages.CONDITIONS.LINK }, { label: _(confT.appliedDeviceConfiguration) }], [])

    useEffect(() => {
        error &&
            Notification.error(
                { title: _(confT.resourcesConfigurationError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_DEVICE_CONFIGURATION_LIST_PAGE_ERROR }
            )
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    console.log(data)

    const columns = useMemo(
        () => [
            {
                Header: _(g.deviceName),
                accessor: 'name',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a
                        href={generatePath(pages.CONDITIONS.APPLIED_DEVICE_CONFIG.DETAIL.LINK, { appliedDeviceConfigId: row.original.id, section: '' })}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(generatePath(pages.CONDITIONS.APPLIED_DEVICE_CONFIG.DETAIL.LINK, { appliedDeviceConfigId: row.original.id, section: '' }))
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
                Cell: ({ value }: { value: string | number }) => (
                    <StatusPill
                        label='Offline'
                        pending={{
                            onClick: () => console.log('on click'),
                            text: '3 pending commands',
                        }}
                        status='offline'
                    />
                ),
            },
            {
                Header: _(confT.condition),
                accessor: 'conditionId',
                Cell: ({ value, row }: { value: string; row: any }) => (
                    <Tag
                        onClick={() => navigate(generatePath(pages.CONDITIONS.CONDITIONS.DETAIL.LINK, { conditionId: row.original.id, tab: '' }))}
                        variant={tagVariants.BLUE}
                    >
                        <IconLink />
                        &nbsp;{_(confT.conditionLink)}
                    </Tag>
                ),
                disableSortBy: true,
            },
            {
                Header: _(g.configuration),
                accessor: 'configurationId',
                Cell: ({ value, row }: { value: string; row: any }) => (
                    <Tag
                        onClick={() => navigate(generatePath(pages.CONDITIONS.RESOURCES_CONFIG.DETAIL.LINK, { resourcesConfigId: row.original.id, tab: '' }))}
                        variant={tagVariants.BLUE}
                    >
                        <IconLink />
                        &nbsp;{_(confT.configLink)}
                    </Tag>
                ),
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
                    icon={<IconPlus />}
                    onClick={() => navigate(generatePath(pages.CONDITIONS.APPLIED_DEVICE_CONFIG.ADD.LINK, { tab: '' }))}
                    variant='primary'
                >
                    {_(confT.configuration)}
                </Button>
            }
            loading={loading}
            title={_(confT.appliedDeviceConfiguration)}
        >
            <PageListTemplate
                columns={columns}
                data={data}
                deleteApiMethod={deleteAppliedDeviceConfigApi}
                i18n={{
                    singleSelected: _(confT.appliedDeviceConfiguration),
                    multiSelected: _(confT.appliedDeviceConfigurations),
                    tablePlaceholder: _(confT.noAppliedDeviceConfiguration),
                    id: _(g.id),
                    name: _(g.name),
                    cancel: _(g.cancel),
                    action: _(g.action),
                    delete: _(g.delete),
                    loading: _(g.loading),
                    deleteModalSubtitle: _(g.undoneAction),
                    view: _(g.view),
                    deleteModalTitle: (selected: number) =>
                        selected === 1
                            ? _(confT.deleteAppliedDeviceConfigurationMessage)
                            : _(confT.deleteAppliedDeviceConfigurationsMessage, { count: selected }),
                }}
                loading={loading}
                onDeletionError={(e) => {
                    Notification.error(
                        { title: _(confT.appliedDeviceConfigurationError), message: getApiErrorMessage(e) },
                        { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_DEVICE_CONFIGURATION_LIST_PAGE_DELETE_ERROR }
                    )
                }}
                onDeletionSuccess={() => {
                    Notification.success(
                        { title: _(confT.appliedDeviceConfigurationDeleted), message: _(confT.appliedDeviceConfigurationDeletedMessage) },
                        { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_DEVICE_CONFIGURATION_LIST_PAGE_DELETE_SUCCESS }
                    )
                }}
                onDetailClick={(id: string) =>
                    navigate(generatePath(pages.CONDITIONS.APPLIED_DEVICE_CONFIG.DETAIL.LINK, { appliedDeviceConfigId: id, tab: '' }))
                }
                refresh={() => refresh()}
            />
        </PageLayout>
    )
}

ListPage.displayName = 'ListPage'

export default ListPage
