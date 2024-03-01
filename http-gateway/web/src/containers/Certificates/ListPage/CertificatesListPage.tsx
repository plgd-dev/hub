import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { useNavigate } from 'react-router-dom'

import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { DeleteModal, IconArrowDetail, IconTrash, StatusTag } from '@shared-ui/components/Atomic'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { parseCertificate } from '@shared-ui/common/services/certificates'

import PageLayout from '@/containers/Common/PageLayout'
import TableList from '@/containers/Common/TableList/TableList'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../Certificates.i18n'
import notificationId from '@/notificationId'
import { useCertificatesList } from '@/containers/Certificates/hooks'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import { deleteCertificatesApi } from '@/containers/Certificates/rest'

const CertificatesListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    const { data, error, loading, refresh } = useCertificatesList()

    const [displayData, setDisplayData] = useState<any>(undefined)

    useEffect(() => {
        const parseCerts = async (certs: any) => {
            const parsed = certs?.map(async (certsData: { credential: { certificatePem: string } }, key: number) => {
                try {
                    return await parseCertificate(certsData?.credential.certificatePem, key, certsData)
                } catch (e: any) {
                    let error = e
                    if (!(error instanceof Error)) {
                        error = new Error(e)
                    }

                    Notification.error(
                        { title: _(t.certificationParsingError), message: error.message },
                        { notificationId: notificationId.HUB_DPS_CERTIFICATES_LIST_CERT_PARSE_ERROR }
                    )
                }
            })

            return await Promise.all(parsed)
        }

        if (data) {
            parseCerts(data).then((d) => {
                setDisplayData(d)
            })
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [data])

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
            await deleteCertificatesApi(selected)

            Notification.success(
                { title: _(t.certificatesDeleted), message: _(t.certificatesDeletedMessage) },
                { notificationId: notificationId.HUB_DEVICES_LIST_PAGE_DELETE_DEVICES }
            )

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
                accessor: 'name',
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
                Header: _(t.serialNumber),
                accessor: 'serialNumber',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(t.type),
                accessor: 'type',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(t.status),
                accessor: 'status',
                Cell: ({ row }: { row: any }) => {
                    const now = new Date()
                    const isValid = row.original.notBeforeUTC <= now && now <= row.original.notAfterUTC
                    return <StatusTag variant={isValid ? 'success' : 'error'}>{isValid ? _(g.valid) : _(g.expired)}</StatusTag>
                },
            },
            {
                Header: _(t.notBefore),
                accessor: 'notBeforeUTC',
                Cell: ({ value }: { value: string | number }) => (value ? <DateFormat rawValue value={value} /> : '-'),
            },
            {
                Header: _(t.notAfter),
                accessor: 'notAfterUTC',
                Cell: ({ value }: { value: string | number }) => (value ? <DateFormat rawValue value={value} /> : '-'),
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
        () => (selectedCount === 1 && displayData ? displayData?.find?.((d: any) => d.id === selected[0])?.name : null),
        [selectedCount, selected, displayData]
    )

    return (
        <PageLayout breadcrumbs={breadcrumbs} loading={loading || deleting || !displayData} title={_(t.certificates)}>
            <TableList
                columns={columns}
                data={displayData}
                defaultSortBy={[
                    {
                        id: 'name',
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
