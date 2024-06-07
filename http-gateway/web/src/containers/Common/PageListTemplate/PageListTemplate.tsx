import React, { FC, useCallback, useMemo, useState } from 'react'
import TableList from '@/containers/Common/TableList/TableList'
import pick from 'lodash/pick'

import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { IconArrowDetail, IconTrash } from '@shared-ui/components/Atomic/Icon'
import DeleteModal from '@shared-ui/components/Atomic/Modal/components/DeleteModal'

import { Props, defaultProps } from './PageListTemplate.types'

const PageListTemplate: FC<Props> = (props) => {
    const {
        columns: columnsProp,
        data,
        globalSearch,
        loading,
        deleteApiMethod,
        onDeletionSuccess,
        onDeletionError,
        refresh,
        i18n,
        onDetailClick,
    } = { ...defaultProps, ...props }

    const [selected, setSelected] = useState<string[]>([])
    const [unselectRowsToken, setUnselectRowsToken] = useState(0)
    const [deleting, setDeleting] = useState(false)

    const handleOpenDeleteModal = useCallback((_isAllSelected: boolean, selection: string[]) => {
        setSelected(selection)
    }, [])

    const handleCloseDeleteModal = useCallback(() => {
        setSelected([])
        setUnselectRowsToken((prev) => prev + 1)
    }, [])

    const deleteMethod = async () => {
        try {
            setDeleting(true)

            await deleteApiMethod(selected)

            handleCloseDeleteModal()

            onDeletionSuccess()

            setSelected([])
            setUnselectRowsToken((prevValue) => prevValue + 1)

            refresh()
            setDeleting(false)
        } catch (e) {
            setDeleting(false)

            onDeletionError(e)
        }
    }

    const selectedCount = useMemo(() => selected.length, [selected])
    const selectedName = useMemo(
        () => (selectedCount === 1 && data ? data?.find?.((d: any) => d.id === selected[0])?.name : null),
        [selectedCount, selected, data]
    )

    const columns = useMemo(
        () => [
            ...columnsProp,
            {
                Header: i18n.action,
                accessor: 'action',
                disableSortBy: true,
                Cell: ({ row }: any) => (
                    <TableActionButton
                        items={[
                            {
                                onClick: () => handleOpenDeleteModal(false, [row.original.id]),
                                label: i18n.delete,
                                icon: <IconTrash />,
                            },
                            {
                                onClick: () => onDetailClick(row.original.id),
                                label: i18n.view,
                                icon: <IconArrowDetail />,
                            },
                        ]}
                    />
                ),
                className: 'actions',
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <>
            <TableList
                columns={columns}
                data={data}
                defaultSortBy={[
                    {
                        id: 'name',
                        desc: false,
                    },
                ]}
                globalSearch={globalSearch}
                i18n={pick(i18n, ['singleSelected', 'multiSelected', 'tablePlaceholder'])}
                loading={loading}
                onDeleteClick={handleOpenDeleteModal}
                unselectRowsToken={unselectRowsToken}
            />

            <DeleteModal
                deleteInformation={
                    selectedCount === 1
                        ? [
                              { label: i18n.name, value: selectedName },
                              { label: i18n.id, value: selected[0] },
                          ]
                        : undefined
                }
                footerActions={[
                    {
                        label: i18n.cancel,
                        onClick: handleCloseDeleteModal,
                        variant: 'tertiary',
                        disabled: loading,
                    },
                    {
                        label: i18n.delete,
                        onClick: deleteMethod,
                        variant: 'primary',
                        loading: deleting,
                        loadingText: i18n.loading,
                    },
                ]}
                fullSizeButtons={selectedCount > 1}
                maxWidth={440}
                maxWidthTitle={320}
                onClose={handleCloseDeleteModal}
                show={selectedCount > 0}
                subTitle={i18n.deleteModalSubtitle}
                title={i18n.deleteModalTitle(selectedCount)}
            />
        </>
    )
}

PageListTemplate.displayName = 'PageListTemplate'

export default PageListTemplate
