import React, { FC, useCallback, useMemo, useState } from 'react'
import pick from 'lodash/pick'
import isFunction from 'lodash/isFunction'

import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import IconArrowDetail from '@shared-ui/components/Atomic/Icon/components/IconArrowDetail'
import IconIntegrations from '@shared-ui/components/Atomic/Icon/components/IconIntegrations'
import IconTrash from '@shared-ui/components/Atomic/Icon/components/IconTrash'
import DeleteModal from '@shared-ui/components/Atomic/Modal/components/DeleteModal'

import { Props, defaultProps } from './PageListTemplate.types'
import TableList from '@/containers/Common/TableList/TableList'

const PageListTemplate: FC<Props> = (props) => {
    const {
        columns: columnsProp,
        data,
        dataTestId,
        globalSearch,
        loading,
        deleteApiMethod,
        onDeletionSuccess,
        onDeletionError,
        onInvoke,
        refresh,
        i18n,
        onDetailClick,
        tableDataTestId,
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

            if (isFunction(deleteApiMethod)) {
                await deleteApiMethod(selected)
            }

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
                            ...(isFunction(onInvoke) && i18n.invoke
                                ? [
                                      {
                                          dataTestId: tableDataTestId?.concat(`-row-${row.id}`).concat('-invoke'),
                                          icon: <IconIntegrations />,
                                          label: i18n.invoke,
                                          onClick: () => onInvoke(row.original.id),
                                      },
                                  ]
                                : []),
                            ...(isFunction(deleteApiMethod)
                                ? [
                                      {
                                          dataTestId: tableDataTestId?.concat(`-row-${row.id}`).concat('-delete'),
                                          icon: <IconTrash />,
                                          label: i18n.delete,
                                          onClick: () => handleOpenDeleteModal(false, [row.original.id]),
                                      },
                                  ]
                                : []),
                            {
                                dataTestId: tableDataTestId?.concat(`-row-${row.id}`).concat('-detail'),
                                icon: <IconArrowDetail />,
                                label: i18n.view,
                                onClick: () => onDetailClick(row.original.id),
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
                dataTestId={tableDataTestId}
                defaultSortBy={[
                    {
                        id: 'name',
                        desc: false,
                    },
                ]}
                globalSearch={globalSearch}
                i18n={pick(i18n, ['singleSelected', 'multiSelected', 'tablePlaceholder', 'delete'])}
                loading={loading}
                onDeleteClick={handleOpenDeleteModal}
                unselectRowsToken={unselectRowsToken}
            />

            <DeleteModal
                dataTestId={dataTestId?.concat('-delete-modal')}
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
                        dataTestId: dataTestId?.concat('-delete-modal-cancel'),
                        disabled: loading,
                        label: i18n.cancel,
                        onClick: handleCloseDeleteModal,
                        variant: 'tertiary',
                    },
                    {
                        dataTestId: dataTestId?.concat('-delete-modal-delete'),
                        label: i18n.delete,
                        loading: deleting,
                        loadingText: i18n.loading,
                        onClick: deleteMethod,
                        variant: 'primary',
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
