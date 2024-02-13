import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { useNavigate } from 'react-router-dom'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import TableActionButton from '@shared-ui//components/Organisms/TableActionButton'
import { DeleteModal, IconArrowDetail, IconPlus, IconTrash } from '@shared-ui/components/Atomic'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import Button from '@shared-ui/components/Atomic/Button'

import { messages as dpsT } from '../../DeviceProvisioning.i18n'
import { messages as t } from '../LinkedHubs.i18n'
import PageLayout from '@/containers/Common/PageLayout'
import { useLinkedHubsList } from '@/containers/DeviceProvisioning/hooks'
import { messages as g } from '@/containers/Global.i18n'
import notificationId from '@/notificationId'
import TableList from '@/containers/Common/TableList/TableList'
import { deleteLinkedHubsApi } from '@/containers/DeviceProvisioning/rest'

const LinkedHubsListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    const { data, loading, error, refresh } = useLinkedHubsList()
    const navigate = useNavigate()

    const [selected, setSelected] = useState<string[]>([])
    const [unselectRowsToken, setUnselectRowsToken] = useState(1)
    const [deleting, setDeleting] = useState(false)

    // eslint-disable-next-line react-hooks/exhaustive-deps
    const breadcrumbs = useMemo(() => [{ label: _(dpsT.deviceProvisioning), link: '/device-provisioning' }, { label: _(t.linkedHubs) }], [])

    useEffect(() => {
        error && Notification.error({ title: _(t.linkedHubsError), message: error }, { notificationId: notificationId.HUB_DPS_LINKED_HUBS_LIST_PAGE_ERROR })
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    const handleOpenDeleteModal = useCallback((_isAllSelected: boolean, selection: string[]) => {
        setSelected(selection)
    }, [])

    const handleCloseDeleteModal = useCallback(() => {
        setSelected([])
        setUnselectRowsToken((prev) => prev + 1)
    }, [])

    const columns = useMemo(
        () => [
            {
                Header: _(g.name),
                accessor: 'name',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a
                        href={`/device-provisioning/linked-hubs/${row.original.id}`}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(`/device-provisioning/linked-hubs/${row.original.id}`)
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
                Header: _(t.deviceGatewayAddress),
                accessor: 'coapGateway',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(g.action),
                accessor: 'action',
                disableSortBy: true,
                Cell: ({ row }: any) => {
                    const {
                        original: { id },
                    } = row
                    return (
                        <TableActionButton
                            items={[
                                {
                                    onClick: () => handleOpenDeleteModal(false, [id]),
                                    label: _(g.delete),
                                    icon: <IconTrash />,
                                },
                                {
                                    onClick: () => navigate(`/device-provisioning/linked-hubs/${id}`),
                                    label: _(g.view),
                                    icon: <IconArrowDetail />,
                                },
                            ]}
                        />
                    )
                },
                className: 'actions',
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const deleteHubs = async () => {
        try {
            setDeleting(true)

            await deleteLinkedHubsApi(selected)

            handleCloseDeleteModal()
            refresh()
            setDeleting(false)
        } catch (e) {
            setDeleting(false)

            Notification.error(
                { title: _(t.linkedHubsError), message: getApiErrorMessage(e) },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_LIST_PAGE_ERROR }
            )
        }
    }

    const selectedCount = useMemo(() => selected.length, [selected])
    const selectedName = useMemo(
        () => (selectedCount === 1 && data ? data?.find?.((d: any) => d.id === selected[0])?.name : null),
        [selectedCount, selected, data]
    )

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={
                <Button icon={<IconPlus />} onClick={() => navigate(`/device-provisioning/linked-hubs/link-new-hub`)} variant='primary'>
                    {_(t.linkedHub)}
                </Button>
            }
            loading={loading || deleting}
            title={_(t.linkedHubs)}
        >
            <TableList
                columns={columns}
                data={data}
                defaultSortBy={[
                    {
                        id: 'name',
                        desc: false,
                    },
                ]}
                i18n={{
                    singleSelected: _(t.linkedHub),
                    multiSelected: _(t.linkedHubs),
                }}
                onDeleteClick={handleOpenDeleteModal}
                unselectRowsToken={unselectRowsToken}
            />

            <DeleteModal
                deleteInformation={
                    selectedCount === 1
                        ? [
                              { label: _(g.name), value: selectedName },
                              { label: _(g.id), value: selected[0] },
                          ]
                        : undefined
                }
                footerActions={[
                    {
                        label: _(g.cancel),
                        onClick: handleCloseDeleteModal,
                        variant: 'tertiary',
                        disabled: loading,
                    },
                    {
                        label: _(g.delete),
                        onClick: deleteHubs,
                        variant: 'primary',
                        loading: deleting,
                        loadingText: _(g.loading),
                    },
                ]}
                fullSizeButtons={selectedCount > 1}
                maxWidth={440}
                maxWidthTitle={320}
                onClose={handleCloseDeleteModal}
                show={selectedCount > 0}
                subTitle={_(t.deleteRecordMessageSubTitle)}
                title={selectedCount === 1 ? _(t.deleteLinkedHubMessage) : _(t.deleteLinkedHubsMessage, { count: selectedCount })}
            />
        </PageLayout>
    )
}

LinkedHubsListPage.displayName = 'LinkedHubsListPage'

export default LinkedHubsListPage
