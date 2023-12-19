import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { useNavigate } from 'react-router-dom'

import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { DeleteModal, IconArrowDetail, IconTrash } from '@shared-ui/components/Atomic'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import PageLayout from '@/containers/Common/PageLayout'
import TableList from '@/containers/Common/TableList/TableList'
import { messages as dpsT } from '../../DeviceProvisioning.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../Certificates.i18n'
import notificationId from '@/notificationId'
import ListHeader from '../ListHeader'

const CertificatesListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    // tmp
    const data: any = useMemo(() => [], [])
    const loading = false
    const error = false
    const refresh = () => {}

    const navigate = useNavigate()

    const [selected, setSelected] = useState<string[]>([])
    const [unselectRowsToken, setUnselectRowsToken] = useState(1)
    const [deleting, setDeleting] = useState(false)

    // eslint-disable-next-line react-hooks/exhaustive-deps
    const breadcrumbs = useMemo(() => [{ label: _(dpsT.deviceProvisioning), link: '/device-provisioning' }, { label: _(t.certificate) }], [])

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
                accessor: 'name',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a
                        data-test-id={`dps-certificates-${row.id}`}
                        href={`/device-provisioning/certificates/${row.original.id}`}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(`/device-provisioning/certificates/${row.original.id}`)
                        }}
                    >
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(g.created),
                accessor: 'created',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(g.expires),
                accessor: 'expires',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
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
                                    onClick: () => navigate(`/device-provisioning/certificates/${id}`),
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
