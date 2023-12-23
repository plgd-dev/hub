import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { useNavigate } from 'react-router-dom'

import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { DeleteModal, IconArrowDetail, IconTrash } from '@shared-ui/components/Atomic'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import PageLayout from '@/containers/Common/PageLayout'
import TableList from '@/containers/Common/TableList/TableList'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../Certificates.i18n'
import notificationId from '@/notificationId'
import ListHeader from '../ListHeader'
import { useCertificatesList } from '@/containers/Certificates/hooks'
import DateFormat from '@/containers/PendingCommands/DateFormat'

const CertificatesListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    const { data, error, loading, refresh } = useCertificatesList()

    const navigate = useNavigate()

    const [selected, setSelected] = useState<string[]>([])
    const [unselectRowsToken, setUnselectRowsToken] = useState(1)
    const [deleting, setDeleting] = useState(false)

    // eslint-disable-next-line react-hooks/exhaustive-deps
    const breadcrumbs = useMemo(() => [{ label: _(t.certificate) }], [])

    const handleOpenDeleteModal = useCallback((_isAllSelected: boolean, selection: string[]) => {
        setSelected(selection)
    }, [])

    const handleCloseDeleteModal = useCallback(() => {
        setSelected([])
        setUnselectRowsToken((prev) => prev + 1)
    }, [])

    const handleDelete = async () => {
        try {
            setDeleting(true)

            console.log('DELELTE')

            handleCloseDeleteModal()
            refresh()
            setDeleting(false)
        } catch (e: any) {
            setDeleting(false)

            Notification.error(
                { title: _(t.certificatesError), message: getApiErrorMessage(e) },
                { notificationId: notificationId.HUB_DPS_CERTIFICATES_LIST_PAGE_ERROR }
            )
        }
    }

    useEffect(() => {
        error && Notification.error({ title: _(t.certificatesError), message: error }, { notificationId: notificationId.HUB_DPS_CERTIFICATES_LIST_PAGE_ERROR })
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    const columns = useMemo(
        () => [
            {
                Header: _(g.name),
                accessor: 'commonName',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a
                        data-test-id={`dps-certificates-${row.id}`}
                        href={`/certificates/${row.original.id}`}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(`/certificates/${row.original.id}`)
                        }}
                    >
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(g.created),
                accessor: 'creationDate',
                Cell: ({ value }: { value: string | number }) => <DateFormat value={value} />,
            },
            {
                Header: _(g.expires),
                accessor: 'credential.validUntilDate',
                Cell: ({ value }: { value: string | number }) => (value ? <DateFormat value={value} /> : '-'),
            },
            {
                Header: _(t.subject),
                accessor: 'subject',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(t.thumbprint),
                accessor: 'thumbprint',
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
                                    onClick: () => navigate(`/certificates/${id}`),
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

    const selectedCount = useMemo(() => selected.length, [selected])
    const selectedName = useMemo(
        () => (selectedCount === 1 && data ? data?.find?.((d: any) => d.id === selected[0])?.name : null),
        [selectedCount, selected, data]
    )

    return (
        <PageLayout breadcrumbs={breadcrumbs} header={<ListHeader />} loading={loading || deleting} title={_(t.certificates)}>
            <TableList
                columns={columns}
                data={data}
                defaultSortBy={[
                    {
                        id: 'commonName',
                        desc: false,
                    },
                ]}
                i18n={{
                    multiSelected: _(t.certificates),
                    singleSelected: _(t.certificate),
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
                        onClick: handleDelete,
                        variant: 'primary',
                        loading: deleting,
                        loadingText: _(g.loading),
                        disabled: loading,
                    },
                ]}
                fullSizeButtons={selectedCount > 1}
                maxWidth={440}
                onClose={handleCloseDeleteModal}
                show={selectedCount > 0}
                subTitle={_(t.deleteCertificatesSubTitle)}
                title={selectedCount === 1 ? _(t.deleteCertificateMessage) : _(t.deleteCertificatesMessage, { count: selectedCount })}
            />
        </PageLayout>
    )
}

CertificatesListPage.displayName = 'CertificatesListPage'

export default CertificatesListPage
