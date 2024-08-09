import React, { FC, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Button from '@shared-ui/components/Atomic/Button'
import { IconPlus } from '@shared-ui/components/Atomic/Icon'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { useIsMounted } from '@shared-ui/common/hooks'

import { pages } from '@/routes'
import { messages as t } from '../ApiTokens.i18n'
import PageLayout from '@/containers/Common/PageLayout'
import PageListTemplate from '@/containers/Common/PageListTemplate/PageListTemplate'
import { messages as g } from '@/containers/Global.i18n'
import notificationId from '@/notificationId'
import AddNewTokenModal from '@/containers/ApiTokens/AddNewTokenModal'
import { useApiTokensList } from '@/containers/ApiTokens/hooks'
import { formatDateVal } from '@/containers/PendingCommands/DateFormat'
import { removeApiTokenApi } from '@/containers/ApiTokens/rest'
import { getExpiration } from '@/containers/ApiTokens/utils'
import testId from '@/testId'

const ListPage: FC<any> = () => {
    const { formatMessage: _, formatDate, formatTime } = useIntl()
    const isMounted = useIsMounted()
    const navigate = useNavigate()
    const { data, loading, error, refresh } = useApiTokensList(!!isMounted)

    // eslint-disable-next-line react-hooks/exhaustive-deps
    const breadcrumbs = useMemo(() => [{ label: _(t.apiTokens), link: pages.API_TOKENS.LINK }], [])

    const [addModal, setAddModal] = useState(false)

    useEffect(() => {
        error &&
            Notification.error({ title: _(t.apiTokensError), message: getApiErrorMessage(error) }, { notificationId: notificationId.HUB_API_TOKENS_LIST_ERROR })
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    const columns = useMemo(
        () => [
            {
                Header: _(g.name),
                accessor: 'name',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a
                        data-test-id={`${testId.apiTokens.list.table}-row-${row.id}-name`}
                        href={generatePath(pages.API_TOKENS.DETAIL, { apiTokenId: row.original.id })}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(generatePath(pages.API_TOKENS.DETAIL, { apiTokenId: row.original.id }))
                        }}
                    >
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(t.expiration),
                accessor: 'expiration',
                Cell: ({ value }: { value: string }) =>
                    getExpiration(value, formatDate, formatTime, {
                        expiredText: (formatedDate) => _(t.expiredDate, { date: formatedDate }),
                        expiresOn: (formatedDate) => _(t.expiresOn, { date: formatedDate }),
                        noExpirationDate: _(t.noExpirationDate),
                    }),
            },
            {
                Header: _(t.issuedAt),
                accessor: 'issuedAt',
                disableSortBy: true,
                Cell: ({ value }: { value: any }) => {
                    const val = new Date(value * 1000)
                    return `${formatDateVal(val, formatDate, formatTime)}`
                },
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={
                <Button dataTestId={testId.apiTokens.list.createTokenButton} icon={<IconPlus />} onClick={() => setAddModal(true)} variant='primary'>
                    {_(t.generateNewToken)}
                </Button>
            }
            loading={loading}
            title={_(t.apiTokens)}
        >
            <PageListTemplate
                columns={columns}
                data={data}
                dataTestId={testId.apiTokens.list.page}
                deleteApiMethod={removeApiTokenApi}
                i18n={{
                    singleSelected: _(t.apiToken),
                    multiSelected: _(t.apiTokens),
                    tablePlaceholder: _(t.noApiTokens),
                    id: _(g.id),
                    name: _(g.name),
                    cancel: _(g.cancel),
                    action: _(g.action),
                    delete: _(g.delete),
                    loading: _(g.loading),
                    deleteModalSubtitle: _(g.undoneAction),
                    view: _(g.view),
                    deleteModalTitle: (selected: number) => (selected === 1 ? _(t.deleteApiTokenMessage) : _(t.deleteApiTokensMessage, { count: selected })),
                }}
                loading={loading}
                onDeletionError={(e) => {
                    Notification.error(
                        { title: _(t.apiTokensError), message: getApiErrorMessage(e) },
                        { notificationId: notificationId.HUB_API_TOKENS_LIST_DELETE_ERROR }
                    )
                }}
                onDeletionSuccess={() => {
                    Notification.success(
                        { title: _(t.apiTokenDeleted), message: _(t.apiTokenDeletedMessage) },
                        { notificationId: notificationId.HUB_API_TOKENS_LIST_DELETE_SUCCESS }
                    )
                }}
                onDetailClick={(id: string) => navigate(generatePath(pages.API_TOKENS.DETAIL, { apiTokenId: id }))}
                refresh={refresh}
                tableDataTestId={testId.apiTokens.list.table}
            />
            <AddNewTokenModal showToken dataTestId={testId.apiTokens.list.addModal} handleClose={() => setAddModal(false)} refresh={refresh} show={addModal} />
        </PageLayout>
    )
}

ListPage.displayName = 'ListPage'

export default ListPage
