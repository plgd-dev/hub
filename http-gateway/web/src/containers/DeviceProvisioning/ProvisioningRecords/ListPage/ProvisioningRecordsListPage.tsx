import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { useNavigate } from 'react-router-dom'

import { DeleteModal, IconArrowDetail, IconTrash } from '@shared-ui/components/Atomic'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'

import { messages as g } from '../../../Global.i18n'
import { messages as dpsT } from '../../DeviceProvisioning.i18n'
import { messages as t } from '../ProvisioningRecords.i18n'
import { useProvisioningRecordsList } from '../../hooks'
import { deleteProvisioningRecordsApi } from '../../rest'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import notificationId from '@/notificationId'
import PageLayout from '@/containers/Common/PageLayout'
import TableList from '@/containers/Common/TableList/TableList'

const ProvisioningRecordsListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    const { data, loading, error, refresh } = useProvisioningRecordsList()

    const navigate = useNavigate()

    const [selected, setSelected] = useState<string[]>([])
    const [unselectRowsToken, setUnselectRowsToken] = useState(1)
    const [deleting, setDeleting] = useState(false)

    // eslint-disable-next-line react-hooks/exhaustive-deps
    const breadcrumbs = useMemo(() => [{ label: _(dpsT.deviceProvisioning), link: '/device-provisioning' }, { label: _(t.provisioningRecords) }], [])

    useEffect(() => {
        error &&
            Notification.error(
                { title: _(t.provisioningRecordsError), message: error },
                { notificationId: notificationId.HUB_DPS_PROVISIONING_RECORDS_LIST_PAGE_ERROR }
            )
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
                Header: _(t.attestationID),
                accessor: 'attestation.x509.commonName',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value || '-'}</span>,
            },
            {
                Header: _(t.enrollmentGroup),
                accessor: 'enrollmentGroupData.name',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a
                        href={`/device-provisioning/provisioning-records/${row.original.id}`}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(`/device-provisioning/provisioning-records/${row.original.id}`)
                        }}
                    >
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(t.deviceID),
                accessor: 'deviceId',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(g.status),
                accessor: 'status',
                Cell: ({ value }: { value: string | number }) => <StatusTag variant='success'>TODO</StatusTag>,
            },
            {
                Header: _(t.firstAttestation),
                accessor: 'creationDate',
                Cell: ({ value }: { value: string | number }) => <DateFormat value={value} />,
            },
            {
                Header: _(t.latestAttestation),
                accessor: 'attestation.date',
                Cell: ({ value }: { value: string | number }) => <DateFormat value={value} />,
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
                                    onClick: () => navigate(`/device-provisioning/provisioning-records/${id}`),
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

    const deleteRecords = async () => {
        try {
            setDeleting(true)
            await deleteProvisioningRecordsApi(selected)

            handleCloseDeleteModal()
            refresh()
            setDeleting(false)
        } catch (e) {
            setDeleting(false)

            Notification.error(
                { title: _(t.provisioningRecordsError), message: getApiErrorMessage(e) },
                { notificationId: notificationId.HUB_DPS_PROVISIONING_RECORDS_LIST_PAGE_ERROR }
            )
        }
    }

    const selectedCount = useMemo(() => selected.length, [selected])
    const selectedName = useMemo(
        () => (selectedCount === 1 && data ? data?.find?.((d: any) => d.id === selected[0])?.enrollmentGroupData?.name : null),
        [selectedCount, selected, data]
    )

    return (
        <PageLayout breadcrumbs={breadcrumbs} loading={loading || deleting} title={_(t.provisioningRecords)}>
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
                    singleSelected: _(t.provisioningRecord),
                    multiSelected: _(t.provisioningRecords),
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
                        onClick: deleteRecords,
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
                title={selectedCount === 1 ? _(t.deleteRecordMessage) : _(t.deleteRecordsMessage, { count: selectedCount })}
            />
        </PageLayout>
    )
}

ProvisioningRecordsListPage.displayName = 'ProvisioningRecordsListPage'

export default ProvisioningRecordsListPage
