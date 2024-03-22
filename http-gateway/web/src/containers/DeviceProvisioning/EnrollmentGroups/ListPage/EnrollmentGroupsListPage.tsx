import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { DeleteModal, IconArrowDetail, IconTrash } from '@shared-ui/components/Atomic'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import Tag from '@shared-ui/components/Atomic/Tag'
import TagGroup from '@shared-ui/components/Atomic/TagGroup'

import PageLayout from '@/containers/Common/PageLayout'
import TableList from '@/containers/Common/TableList/TableList'
import { messages as dpsT } from '../../DeviceProvisioning.i18n'
import { useEnrollmentGroupDataList } from '../../hooks'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../EnrollmentGroups.i18n'
import { deleteEnrollmentGroupsApi } from '@/containers/DeviceProvisioning/rest'
import notificationId from '@/notificationId'
import ListHeader from '../ListHeader'
import { pages } from '@/routes'

const EnrollmentGroupsListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    const { data, loading, error, refresh } = useEnrollmentGroupDataList()
    const navigate = useNavigate()

    const [selected, setSelected] = useState<string[]>([])
    const [unselectRowsToken, setUnselectRowsToken] = useState(1)
    const [deleting, setDeleting] = useState(false)

    // eslint-disable-next-line react-hooks/exhaustive-deps
    const breadcrumbs = useMemo(() => [{ label: _(dpsT.deviceProvisioning), link: '/device-provisioning' }, { label: _(dpsT.enrollmentGroups) }], [])

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
            await deleteEnrollmentGroupsApi(selected)
            handleCloseDeleteModal()
            refresh()
            setDeleting(false)
        } catch (e: any) {
            setDeleting(false)

            Notification.error(
                { title: _(t.enrollmentGroupsError), message: getApiErrorMessage(e) },
                { notificationId: notificationId.HUB_DPS_ENROLLMENT_GROUP_LIST_PAGE_ERROR }
            )
        }
    }

    useEffect(() => {
        error &&
            Notification.error(
                { title: _(t.enrollmentGroupsError), message: error },
                { notificationId: notificationId.HUB_DPS_ENROLLMENT_GROUP_LIST_PAGE_ERROR }
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
                        data-test-id={`enrollment-group-${row.id}`}
                        href={`/device-provisioning/enrollment-groups/${row.original.id}`}
                        onClick={(e) => {
                            e.preventDefault()
                            navigate(generatePath(pages.DPS.ENROLLMENT_GROUPS.DETAIL, { enrollmentId: row.original.id }))
                        }}
                    >
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(dpsT.linkedHub),
                accessor: 'hubsData',
                Cell: ({ value, row }: { value: { name: string }[]; row: any }) => (
                    <TagGroup
                        i18n={{
                            more: _(g.more),
                            modalHeadline: _(dpsT.linkedHubs),
                        }}
                    >
                        {value?.map?.((hub: { name: string }) => (
                            <Tag className='tree-custom-tag' key={`${hub.name}-${row.id}`}>
                                {hub.name}
                            </Tag>
                        ))}
                    </TagGroup>
                ),
            },
            {
                Header: _(dpsT.ownerId),
                accessor: 'owner',
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
                                    onClick: () => navigate(generatePath(pages.DPS.ENROLLMENT_GROUPS.DETAIL, { enrollmentId: id })),
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
        <PageLayout breadcrumbs={breadcrumbs} header={<ListHeader />} loading={loading || deleting} title={_(dpsT.enrollmentGroups)}>
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
                    multiSelected: _(dpsT.enrollmentGroups),
                    singleSelected: _(dpsT.enrollmentGroup),
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
                subTitle={_(t.deleteEnrollmentGroupsSubTitle)}
                title={selectedCount === 1 ? _(t.deleteEnrollmentGroupMessage) : _(t.deleteEnrollmentGroupsMessage, { count: selectedCount })}
            />
        </PageLayout>
    )
}

EnrollmentGroupsListPage.displayName = 'EnrollmentGroupsListPage'

export default EnrollmentGroupsListPage
