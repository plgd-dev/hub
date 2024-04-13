import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'

import Loadable from '@shared-ui/components/Atomic/Loadable'
import { IconArrowDetail, IconTrash } from '@shared-ui/components/Atomic/Icon'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import DeleteModal from '@shared-ui/components/Atomic/Modal/components/DeleteModal'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { parseCertificate } from '@shared-ui/common/services/certificates'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { getApiErrorMessage } from '@shared-ui/common/utils'

import { messages as t } from '@/containers/Certificates/Certificates.i18n'
import TableList from '@/containers/Common/TableList/TableList'
import { messages as g } from '@/containers/Global.i18n'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import { Props } from './CertificateList.types'

const CertificatesList: FC<Props> = (props) => {
    const { data, deleting, loading, onDelete, onView, refresh, notificationIds, setDeleting } = props

    const { formatMessage: _ } = useIntl()

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

                    console.error(error)
                    Notification.error({ title: _(t.certificationParsingError), message: error.message }, { notificationId: notificationIds.parsingError })
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

    const [selected, setSelected] = useState<string[]>([])
    const [unselectRowsToken, setUnselectRowsToken] = useState(1)

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
            await onDelete(displayData.filter((d: any) => selected.includes(d.id)).map((d: any) => d.origin.id))

            Notification.success(
                { title: _(t.certificatesDeleted), message: _(t.certificatesDeletedMessage) },
                { notificationId: notificationIds.deleteSuccess }
            )

            handleCloseDeleteModal()
            refresh()
            setDeleting(false)
        } catch (e: any) {
            setDeleting(false)

            Notification.error({ title: _(t.certificatesDeleteError), message: getApiErrorMessage(e) }, { notificationId: notificationIds.deleteError })
        }
    }

    const handleView = useCallback(
        (id: string, rowId: number) => {
            onView(id, displayData[rowId])
        },
        [displayData, onView]
    )

    const columns = useMemo(
        () => [
            {
                Header: _(g.name),
                accessor: 'name',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a
                        data-test-id={`dps-certificates-${row.id}`}
                        href={row.original.origin.id}
                        onClick={(e) => {
                            e.preventDefault()
                            handleView(row.original.origin.id, row.id)
                        }}
                    >
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(t.type),
                accessor: 'type',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(t.status),
                accessor: 'status',
                Cell: ({ value }: { value: boolean }) => <StatusTag variant={value ? 'success' : 'error'}>{value ? _(g.valid) : _(g.expired)}</StatusTag>,
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
                                    onClick: () => {
                                        handleView(row.original.origin.id, row.id)
                                    },
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
        [displayData, handleView]
    )

    const selectedCount = useMemo(() => selected.length, [selected])
    const selectedInfo = useMemo(
        () => (selectedCount === 1 && displayData ? displayData?.find?.((d: any) => d.id === selected[0]) : null),
        [selectedCount, selected, displayData]
    )

    return (
        <>
            <Loadable condition={!!displayData}>
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
            </Loadable>

            <DeleteModal
                deleteInformation={
                    selectedCount === 1
                        ? [
                              { label: _(g.name), value: selectedInfo.name },
                              { label: _(g.id), value: selectedInfo.origin.id },
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
        </>
    )
}

CertificatesList.displayName = 'CertificatesList'

export default CertificatesList
